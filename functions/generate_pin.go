// In functions package

package functions

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateNumericPIN generates a random numeric PIN of the given length,
// zero-padded. Used for the 8-digit Sales Associate PIN — admin-created,
// not self-service, per the brief.
func GenerateNumericPIN(length int) (string, error) {
	max := big.NewInt(1)
	for i := 0; i < length; i++ {
		max.Mul(max, big.NewInt(10))
	}

	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("could not generate pin: %w", err)
	}

	return fmt.Sprintf("%0*d", length, n.Int64()), nil
}
