package server

import (
	"net/http"
	"strings"

	"github.com/runabol/streamabol/hmac"
)

type HMACMiddleware struct {
	SecretKey string
}

func NewHMACMiddleware(secretKey string) *HMACMiddleware {
	return &HMACMiddleware{SecretKey: secretKey}
}

func (m *HMACMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m.SecretKey == "" {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path != "/manifest.m3u8" && !strings.HasPrefix(r.URL.Path, "/playlist/") && !strings.HasPrefix(r.URL.Path, "/segment/") {
			next.ServeHTTP(w, r)
			return
		}
		verified := hmac.Verify(r.URL, m.SecretKey)
		if !verified {
			http.Error(w, "HMAC verification failed", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
