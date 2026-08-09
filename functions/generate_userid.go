package functions

import (
	"crypto/rand"
	"math/big"
)

func GenerateUserID() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	lengthOptions := []int{10, 11, 12}

	// Randomly select a length from lengthOptions
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(lengthOptions))))
	if err != nil {
		return "", err
	}
	length := lengthOptions[n.Int64()]

	// Generate random characters
	userID := make([]byte, length)
	for i := 0; i < length; i++ {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		userID[i] = charset[index.Int64()]
	}
	return string(userID), nil
}
