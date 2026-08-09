package functions

import (
	"crypto/rand"
	"math/big"
)

func GenerateOTPCode(length int) (string, error) {
	const digits = "0123456789"
	otp := make([]byte, length)
	_, err := rand.Read(otp)
	if err != nil {
		return "", err
	}
	for i := 0; i < length; i++ {
		otp[i] = digits[otp[i]%byte(len(digits))]
	}
	return string(otp), nil
}

func GenerateAuthorizationCode() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	minLength := 20
	maxLength := 25

	// Generate a random length between 20 and 25
	lengthRange := maxLength - minLength + 1
	randomLength, err := rand.Int(rand.Reader, big.NewInt(int64(lengthRange)))
	if err != nil {
		return "", err
	}
	length := int(randomLength.Int64()) + minLength

	// Generate the random code
	code := make([]byte, length)
	randomBytes := make([]byte, length)
	_, err = rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	for i := 0; i < length; i++ {
		code[i] = charset[randomBytes[i]%byte(len(charset))]
	}

	return string(code), nil
}
