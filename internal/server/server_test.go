package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SamuelSchutz13/orion-logger/internal/session"
	"github.com/gin-gonic/gin"
)

func testEngine(t *testing.T, deps Deps) (*gin.Engine, *session.Session) {
	t.Helper()
	s, err := session.New()
	if err != nil {
		t.Fatal(err)
	}
	s.Port = 7777
	return New(s, deps), s
}

func do(e http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	e.ServeHTTP(w, req)
	return w
}

func req(method, path, host string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	r.Host = host
	return r
}

func TestNoTokenIs401(t *testing.T) {
	e, _ := testEngine(t, Deps{})
	if w := do(e, req("GET", "/api/health", "127.0.0.1:7777")); w.Code != 401 {
		t.Fatalf("code %d", w.Code)
	}
}

func TestWrongTokenIs401(t *testing.T) {
	e, _ := testEngine(t, Deps{})
	r := req("GET", "/api/health", "127.0.0.1:7777")
	r.Header.Set("X-Orion-Token", "nope")
	if w := do(e, r); w.Code != 401 {
		t.Fatalf("code %d", w.Code)
	}
}

func TestHeaderTokenOK(t *testing.T) {
	e, s := testEngine(t, Deps{})
	r := req("GET", "/api/health", "127.0.0.1:7777")
	r.Header.Set("X-Orion-Token", s.Token)
	if w := do(e, r); w.Code != 200 {
		t.Fatalf("code %d", w.Code)
	}
}

func TestCookieTokenOK(t *testing.T) {
	e, s := testEngine(t, Deps{})
	r := req("GET", "/api/health", "localhost:7777")
	r.AddCookie(&http.Cookie{Name: "orion_token", Value: s.Token})
	if w := do(e, r); w.Code != 200 {
		t.Fatalf("code %d", w.Code)
	}
}

func TestBadHostIs403(t *testing.T) {
	e, s := testEngine(t, Deps{})
	r := req("GET", "/api/health", "evil.com:7777")
	r.Header.Set("X-Orion-Token", s.Token)
	if w := do(e, r); w.Code != 403 {
		t.Fatalf("code %d", w.Code)
	}
}

func TestBadOriginIs403(t *testing.T) {
	e, s := testEngine(t, Deps{})
	r := req("GET", "/api/health", "127.0.0.1:7777")
	r.Header.Set("X-Orion-Token", s.Token)
	r.Header.Set("Origin", "http://evil.com")
	if w := do(e, r); w.Code != 403 {
		t.Fatalf("code %d", w.Code)
	}
}

func TestTokenQuerySetsCookie(t *testing.T) {
	e, s := testEngine(t, Deps{})
	w := do(e, req("GET", "/?t="+s.Token, "127.0.0.1:7777"))
	if w.Code != 302 || w.Header().Get("Location") != "/" {
		t.Fatalf("code %d loc %q", w.Code, w.Header().Get("Location"))
	}
	c := w.Header().Get("Set-Cookie")
	for _, want := range []string{"orion_token=" + s.Token, "HttpOnly", "SameSite=Strict"} {
		if !strings.Contains(c, want) {
			t.Fatalf("cookie %q missing %q", c, want)
		}
	}
}

// O cookie é por host: quem abre por localhost sem cookie vai para 127.0.0.1, onde o cookie está.
func TestLocalhostPageRedirectsTo127(t *testing.T) {
	e, _ := testEngine(t, Deps{})
	w := do(e, req("GET", "/algum/lugar?x=1", "localhost:7777"))
	if w.Code != 302 || w.Header().Get("Location") != "http://127.0.0.1:7777/algum/lugar?x=1" {
		t.Fatalf("code %d loc %q", w.Code, w.Header().Get("Location"))
	}
	// API continua respondendo 401 (sem redirecionar)
	if w := do(e, req("GET", "/api/health", "localhost:7777")); w.Code != 401 {
		t.Fatalf("api code %d", w.Code)
	}
}
