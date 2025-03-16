package hmac

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
)

// Verify checks if the HMAC signature in the URL matches the expected signature
// It returns true if the HMAC signature is valid, and false otherwise
func Verify(url *url.URL, secret string) bool {
	// Get the hmac signature from query params
	queryParams := url.Query()
	hmacSignature := queryParams.Get("hmac")
	if hmacSignature == "" {
		return false
	}

	// Remove the hmac parameter to reconstruct the original message
	queryParams.Del("hmac")
	url.RawQuery = queryParams.Encode()
	message := url.String()

	// Decode the provided HMAC signature
	decodedSignature, err := hex.DecodeString(hmacSignature)
	if err != nil {
		return false
	}

	// Create a new HMAC hasher with the secret
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	expectedMAC := mac.Sum(nil)

	// Compare the signatures
	return hmac.Equal(decodedSignature, expectedMAC)
}

// Generate generates an HMAC signature for the given message and secret
// It returns the HMAC signature as a hex-encoded string
func Generate(message, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}
