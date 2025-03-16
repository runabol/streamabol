package hmac_test

import (
	"net/url"
	"testing"

	"github.com/runabol/streamabol/hmac"
	"github.com/stretchr/testify/assert"
)

func TestVerify_ValidHMAC(t *testing.T) {
	secret := "mysecret"
	message := "https://example.com/path?param=value"
	expectedHMAC := hmac.Generate(message, secret)

	u, err := url.Parse(message + "&hmac=" + expectedHMAC)
	assert.NoError(t, err)

	valid := hmac.Verify(u, secret)
	assert.True(t, valid)
}

func TestVerify_InvalidHMAC(t *testing.T) {
	secret := "mysecret"
	message := "https://example.com/path?param=value"
	invalidHMAC := "invalidhmac"

	u, err := url.Parse(message + "&hmac=" + invalidHMAC)
	assert.NoError(t, err)

	valid := hmac.Verify(u, secret)
	assert.False(t, valid)
}

func TestVerify_MissingHMAC(t *testing.T) {
	secret := "mysecret"
	message := "https://example.com/path?param=value"

	u, err := url.Parse(message)
	assert.NoError(t, err)

	valid := hmac.Verify(u, secret)
	assert.False(t, valid)
}

func TestVerify_InvalidURL(t *testing.T) {
	secret := "mysecret"
	invalidURL := "https://example.com/path?param=value&hmac=invalid%ZZ"

	u, err := url.Parse(invalidURL)
	assert.NoError(t, err)

	valid := hmac.Verify(u, secret)
	assert.False(t, valid)
}

func TestGenerate(t *testing.T) {
	secret := "mysecret"
	message := "https://example.com/path?param=value"

	hmacSignature := hmac.Generate(message, secret)
	assert.NotEmpty(t, hmacSignature)

	u, err := url.Parse(message + "&hmac=" + hmacSignature)
	assert.NoError(t, err)

	valid := hmac.Verify(u, secret)
	assert.True(t, valid)
}
