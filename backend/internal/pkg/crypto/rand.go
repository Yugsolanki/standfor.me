package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
)

func CryptoFloat64() (float64, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}

	// Convert to uint64 and mask to get 53 bits of randomness
	// (float64 has 53 bits of mantissa precision)
	u := binary.BigEndian.Uint64(b[:]) & ((1 << 53) - 1)

	// Convert to float64 in range [0,1)
	return float64(u) / (1 << 53), nil
}

func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func EncodeBase64(b []byte) string {
	return base64.URLEncoding.EncodeToString(b)
}
