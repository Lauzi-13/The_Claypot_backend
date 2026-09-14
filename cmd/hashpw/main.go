// hashpw generates a bcrypt hash for a new staff password, so a new staff
// account can be added directly in the database (via DBeaver) without ever
// storing the plain password anywhere. Run with: go run ./cmd/hashpw
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	fmt.Print("New staff password (min 6 characters): ")
	reader := bufio.NewReader(os.Stdin)
	password, _ := reader.ReadString('\n')
	password = strings.TrimRight(password, "\r\n")

	if len(password) < 6 {
		fmt.Println("Password must be at least 6 characters.")
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Error hashing password:", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Paste this into the password_hash column in DBeaver:")
	fmt.Println(string(hash))
}
