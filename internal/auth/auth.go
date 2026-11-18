// Package auth protects WiFerry with a per-launch secret token.
//
// LAN clients get the token from the QR code URL (?t=...), which is also
// stored in an HttpOnly cookie for API calls. The host's own browser on loopback is trusted
// without a token, provided the Host header is also local (guards against
// DNS rebinding).
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net"
	"net/http"
)

// CookieName is the cookie that carries the token after the first visit.
const CookieName = "wiferry_token"

// Auth holds the secret token for this run.
type Auth struct {
	token string
}

// New creates an Auth with a fresh 128-bit random token.
func New() (*Auth, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return &Auth{token: base64.RawURLEncoding.EncodeToString(b)}, nil
}

// NewWithToken creates an Auth with a fixed token (for tests).
func NewWithToken(token string) *Auth { return &Auth{token: token} }

// Token returns the secret token.
func (a *Auth) Token() string { return a.token }

func (a *Auth) valid(s string) bool {
	return s != "" && subtle.ConstantTimeCompare([]byte(s), []byte(a.token)) == 1
}

// IsLoopback reports whether r comes from this machine and names a local host.
func IsLoopback(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return false
	}
	h := r.Host
	if hh, _, err := net.SplitHostPort(h); err == nil {
		h = hh
	}
	switch h {
	case "127.0.0.1", "localhost", "::1", "[::1]":
		return true
	}
	return false
}

// Middleware rejects requests without a valid token (403), except /healthz
// and trusted loopback requests.
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || IsLoopback(r) {
			next.ServeHTTP(w, r)
			return
		}
		q := r.URL.Query()
		if t := q.Get("t"); t != "" {
			if !a.valid(t) {
				forbidden(w)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:     CookieName,
				Value:    a.token,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
			// Serve directly instead of redirecting to a clean URL: the token
			// must stay in the address bar so "Open in Safari" from an in-app
			// QR viewer (separate cookie jar) still connects.
			next.ServeHTTP(w, r)
			return
		}
		if c, err := r.Cookie(CookieName); err == nil && a.valid(c.Value) {
			next.ServeHTTP(w, r)
			return
		}
		forbidden(w)
	})
}

func forbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`<!doctype html><meta name="viewport" content="width=device-width,initial-scale=1">` +
		`<title>WiFerry</title><body style="font-family:system-ui;padding:2rem;text-align:center">` +
		`<h1>Not connected</h1><p>Scan the current QR code shown on the PC running WiFerry.</p></body>`))
}
