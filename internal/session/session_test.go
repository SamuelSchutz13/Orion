package session

import (
	"regexp"
	"testing"
)

func TestTokenIs64Hex(t *testing.T) {
	s, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(s.Token) {
		t.Fatalf("token %q", s.Token)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	s, _ := New()
	s.Port = 7777
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.Token != s.Token || got.Port != 7777 || got.PID == 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	c := LoadConfig()
	if len(c.LogFiles) != 0 {
		t.Fatal("expected empty")
	}
	c.LogFiles = []string{`C:\x.log`}
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	if LoadConfig().LogFiles[0] != `C:\x.log` {
		t.Fatal("not saved")
	}
}
