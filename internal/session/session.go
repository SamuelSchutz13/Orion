// Package session guarda o token da sessão local e a configuração do usuário.
package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

type Session struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
	PID   int    `json:"pid"`
}

// Dir é %APPDATA%\orion-logger (pasta só do usuário).
func Dir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		d = os.TempDir()
	}
	return filepath.Join(d, "orion-logger")
}

func New() (*Session, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return &Session{Token: hex.EncodeToString(b), PID: os.Getpid()}, nil
}

func (s *Session) Save() error { return writeJSON("session.json", s) }

func Remove() { os.Remove(filepath.Join(Dir(), "session.json")) }

func Load() (*Session, error) {
	var s Session
	return &s, readJSON("session.json", &s)
}

type Config struct {
	LogFiles []string `json:"logFiles"`
}

// LoadConfig nunca falha: config ausente ou inválida vira config vazia.
func LoadConfig() *Config {
	c := &Config{}
	_ = readJSON("config.json", c)
	return c
}

func (c *Config) Save() error { return writeJSON("config.json", c) }

func writeJSON(name string, v any) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(Dir(), name+".tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(Dir(), name))
}

func readJSON(name string, v any) error {
	b, err := os.ReadFile(filepath.Join(Dir(), name))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
