package main

import (
	"log"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"claypot-backend/models"
)

// Bootstraps the very first staff login. Additional staff accounts are
// created directly in the database (e.g. via DBeaver), not through the
// app — see cmd/hashpw for generating the bcrypt hash to insert. Only
// ever hashed and stored, never logged or printed anywhere.
const initialStaffUsername = "Staff3"
const initialStaffPassword = "St@ff@123"

// seedStaffUser creates the first account only if no staff account exists
// yet at all — safe to run every time (won't reset the password on an
// account that's already been created, or add a duplicate).
func seedStaffUser(db *gorm.DB) bool {
	var count int64
	db.Model(&models.StaffUser{}).Count(&count)
	if count > 0 {
		return false
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(initialStaffPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hashing initial staff password: %v", err)
	}
	user := models.StaffUser{
		ID: uuid.NewString(), Username: initialStaffUsername, PasswordHash: string(hash),
		Role: models.StaffRoleStaff, Active: true,
	}
	if err := db.Create(&user).Error; err != nil {
		log.Fatalf("seeding initial staff user: %v", err)
	}
	return true
}
