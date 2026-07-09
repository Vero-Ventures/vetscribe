package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("VETSCRIBE_LISTEN_ADDR", "")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ListenAddr != ":8080" {
		t.Fatalf("expected default listen addr :8080, got %q", cfg.ListenAddr)
	}
}

func TestLoadOverride(t *testing.T) {
	t.Setenv("VETSCRIBE_LISTEN_ADDR", "127.0.0.1:9000")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ListenAddr != "127.0.0.1:9000" {
		t.Fatalf("expected overridden listen addr, got %q", cfg.ListenAddr)
	}
}
