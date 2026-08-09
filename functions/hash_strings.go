package functions

import "golang.org/x/crypto/bcrypt"

func HashString(s string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}
