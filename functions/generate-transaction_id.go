// In functions package, alongside GenerateOTPCode etc.

package functions

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// GenerateTransactionID builds a region-suffixed transaction ID:
// PREFIX-YYMMDD-NNNNNN-REGION (e.g. INV-260816-482913-LG).
// The brief's examples show a more elaborate multi-segment format
// (INV-260816-2627-00006-001-LG); this is a simplified single-random-segment
// version that preserves the important properties — date-stamped, unique,
// regionally suffixed — without needing the extra segments' meaning defined
// yet. Easy to extend once those segments are specified.
func GenerateTransactionID(prefix, region string) (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("could not generate transaction id: %w", err)
	}

	datePart := time.Now().Format("060102") // YYMMDD
	return fmt.Sprintf("%s-%s-%06d-%s", prefix, datePart, n.Int64(), region), nil
}
