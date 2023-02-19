package tiny

import (
	"fmt"
	"net/http"
	"time"
)

const (
	defaultMaxAge = 30 * 24 * time.Hour
)

// Cache cache static resources.
func Cache(maxAge time.Duration) func(http.Handler) http.Handler {
	if maxAge == 0 {
		maxAge = defaultMaxAge
	}
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int64(maxAge.Seconds())))
				h.ServeHTTP(w, r)
			})
	}
}

// AuthRequired provides middleware for redirecting user to login page if they have not logged in yet.
func AuthRequired(loginPath string, authInfoFunc AuthInfoFunc) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			if _, ok := authInfoFunc(r.Context()); !ok {
				http.Redirect(rw, r, fmt.Sprintf("%s?redirect=%s", loginPath, r.URL.Path), http.StatusFound)
				return
			}
			h.ServeHTTP(rw, r)
		})
	}
}
