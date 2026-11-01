// Package server expõe a API local e o frontend, só em 127.0.0.1 e com token.
package server

import (
	"crypto/subtle"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"

	"github.com/SamuelSchutz13/orion-logger/internal/inventory"
	"github.com/SamuelSchutz13/orion-logger/internal/session"
	"github.com/SamuelSchutz13/orion-logger/web"
	"github.com/gin-gonic/gin"
)

const cookieName = "orion_token"

// Deps são as peças que os handlers usam; campos nil desativam as rotas correspondentes.
type Deps struct {
	Collector *inventory.Collector
}

func New(s *session.Session, deps Deps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	e := gin.New()
	e.Use(gin.Recovery(), guard(s))
	api := e.Group("/api")
	api.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	registerAPI(api, deps)
	e.NoRoute(static())
	return e
}

// Listen tenta a porta pedida e cai para uma livre se estiver ocupada.
func Listen(port int) (net.Listener, error) {
	if l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port)); err == nil {
		return l, nil
	}
	return net.Listen("tcp", "127.0.0.1:0")
}

func guard(s *session.Session) gin.HandlerFunc {
	hosts := map[string]bool{
		fmt.Sprintf("127.0.0.1:%d", s.Port): true,
		fmt.Sprintf("localhost:%d", s.Port): true,
	}
	ok := func(v string) bool { return subtle.ConstantTimeCompare([]byte(v), []byte(s.Token)) == 1 }
	return func(c *gin.Context) {
		// Host e Origin barram DNS rebinding e chamadas de outros sites.
		if !hosts[c.Request.Host] {
			c.AbortWithStatusJSON(403, gin.H{"error": "host não permitido"})
			return
		}
		if o := c.GetHeader("Origin"); o != "" && !hosts[strings.TrimPrefix(o, "http://")] {
			c.AbortWithStatusJSON(403, gin.H{"error": "origem não permitida"})
			return
		}
		if c.Request.URL.Path == "/" && ok(c.Query("t")) {
			http.SetCookie(c.Writer, &http.Cookie{Name: cookieName, Value: s.Token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
			c.Redirect(302, "/")
			c.Abort()
			return
		}
		tok := c.GetHeader("X-Orion-Token")
		if tok == "" {
			tok, _ = c.Cookie(cookieName)
		}
		if !ok(tok) {
			c.AbortWithStatusJSON(401, gin.H{"error": "sem token: abra o Orion pelo link mostrado no terminal"})
			return
		}
		c.Next()
	}
}

func static() gin.HandlerFunc {
	dist, _ := fs.Sub(web.Dist, "dist")
	files := http.FileServer(http.FS(dist))
	return func(c *gin.Context) {
		p := strings.TrimPrefix(c.Request.URL.Path, "/")
		if strings.HasPrefix(p, "api/") {
			c.JSON(404, gin.H{"error": "rota não encontrada"})
			return
		}
		if p != "" {
			if f, err := dist.Open(p); err == nil {
				f.Close()
				files.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		// fallback SPA
		b, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			c.String(200, "Orion Logger: frontend não compilado (rode scripts/build.ps1).")
			return
		}
		c.Data(200, "text/html; charset=utf-8", b)
	}
}
