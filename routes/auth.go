package routes

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"claypot-backend/models"
)

const sessionLifetime = 24 * time.Hour

type AuthHandler struct {
	DB *gorm.DB
}

func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/login", h.login)
	r.Post("/logout", h.logout)
	return r
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type loginBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// POST /api/auth/login { username, password } — replaces checkStaffPin().
// On success, creates a session row and returns its token; the frontend
// sends that back as "Authorization: Bearer <token>" on every request from
// then on (see RequireAuth below).
func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var body loginBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body.")
		return
	}
	if body.Username == "" || body.Password == "" {
		writeError(w, http.StatusBadRequest, "Username and password are required.")
		return
	}

	var user models.StaffUser
	if err := h.DB.Where("username = ? AND active = ?", body.Username, true).First(&user).Error; err != nil {
		writeError(w, http.StatusUnauthorized, "Incorrect username or password.")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(body.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "Incorrect username or password.")
		return
	}

	token, err := generateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}
	now := time.Now()
	session := models.StaffSession{
		Token: token, StaffUserID: user.ID, CreatedAt: now, ExpiresAt: now.Add(sessionLifetime),
	}
	if err := h.DB.Create(&session).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
		return
	}

	// Lazy cleanup: no harm in a few stale rows sitting around, but no
	// reason to let them build up forever either.
	h.DB.Where("staff_user_id = ? AND expires_at < ?", user.ID, now).Delete(&models.StaffSession{})

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(auth, "Bearer ")
}

// POST /api/auth/logout — deletes the session for whatever token was sent.
// Idempotent: logging out twice, or with an already-expired/unknown
// token, is not an error.
func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r)
	if token != "" {
		h.DB.Delete(&models.StaffSession{}, "token = ?", token)
	}
	w.WriteHeader(http.StatusNoContent)
}

// RequireAuth protects a route: no valid, unexpired session token means a
// 401 and the wrapped handler never runs. This is the real security
// boundary — the frontend's page-level redirect is just a UX convenience,
// not what's actually keeping anyone out.
func RequireAuth(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				writeError(w, http.StatusUnauthorized, "Please log in.")
				return
			}
			var session models.StaffSession
			if err := db.First(&session, "token = ?", token).Error; err != nil {
				writeError(w, http.StatusUnauthorized, "Please log in.")
				return
			}
			if time.Now().After(session.ExpiresAt) {
				db.Delete(&session)
				writeError(w, http.StatusUnauthorized, "Your session has expired. Please log in again.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
