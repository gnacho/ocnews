package auth

import "golang.org/x/crypto/bcrypt"

func bcryptGenerate(pw string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
}
