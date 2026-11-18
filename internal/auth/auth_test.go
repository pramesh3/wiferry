package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func do(t *testing.T, a *Auth, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	a.Middleware(okHandler()).ServeHTTP(rec, req)
	return rec
}

func TestNewTokenIsRandomAndURLSafe(t *testing.T) {
	a1, err := New()
	if err != nil {
		t.Fatal(err)
	}
	a2, _ := New()
	if len(a1.Token()) != 22 {
		t.Fatalf("token length = %d, want 22", len(a1.Token()))
	}
	if a1.Token() == a2.Token() {
		t.Fatal("tokens should differ")
	}
}

func TestHealthzNeedsNoToken(t *testing.T) {
	a := NewWithToken("secret")
	rec := do(t, a, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Code != 200 {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestNoTokenForbidden(t *testing.T) {
	a := NewWithToken("secret")
	rec := do(t, a, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("code = %d", rec.Code)
	}
}

// The token stays in the address bar (no redirect) so "Open in Safari" from
// an in-app viewer, which doesn't share cookies, still connects.
func TestQueryTokenSetsCookieAndServesPage(t *testing.T) {
	a := NewWithToken("secret")
	rec := do(t, a, httptest.NewRequest("GET", "/?t=secret&x=1", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != CookieName || cookies[0].Value != "secret" {
		t.Fatalf("cookies = %v", cookies)
	}
	if !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatal("cookie must be HttpOnly + SameSite=Strict")
	}
}

func TestWrongQueryTokenForbidden(t *testing.T) {
	a := NewWithToken("secret")
	rec := do(t, a, httptest.NewRequest("GET", "/?t=nope", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("code = %d", rec.Code)
	}
}

func TestCookie(t *testing.T) {
	a := NewWithToken("secret")
	for _, tc := range []struct {
		val  string
		want int
	}{{"secret", 200}, {"bad", 403}} {
		req := httptest.NewRequest("GET", "/api/files", nil)
		req.AddCookie(&http.Cookie{Name: CookieName, Value: tc.val})
		if rec := do(t, a, req); rec.Code != tc.want {
			t.Errorf("cookie %q: code = %d, want %d", tc.val, rec.Code, tc.want)
		}
	}
}

func TestLoopbackTrustedOnlyWithLocalHost(t *testing.T) {
	a := NewWithToken("secret")
	for _, tc := range []struct {
		remote, host string
		want         int
	}{
		{"127.0.0.1:5000", "127.0.0.1:8642", 200},
		{"127.0.0.1:5000", "localhost:8642", 200},
		{"[::1]:5000", "[::1]:8642", 200},
		{"127.0.0.1:5000", "evil.com", 403},
		{"192.168.1.9:5000", "127.0.0.1:8642", 403},
	} {
		req := httptest.NewRequest("GET", "/host", nil)
		req.RemoteAddr = tc.remote
		req.Host = tc.host
		if rec := do(t, a, req); rec.Code != tc.want {
			t.Errorf("%s via %s: code = %d, want %d", tc.remote, tc.host, rec.Code, tc.want)
		}
	}
}
