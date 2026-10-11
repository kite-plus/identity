package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func write(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadExtendsAndExpands(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "base.yaml", `
orgs: {operators: Ops, readers: Readers}
loginPolicy: {passwordCheckLifetime: 720h, allowRegister: true}
idps:
  github: {name: GitHub, clientId: "${GH_ID:-}", clientSecret: "${GH_SECRET:-}"}
`)
	p := write(t, dir, "dev.yaml", `
extends: base.yaml
environment: dev
issuer: http://localhost:${TEST_PORT:-8090}
loginPolicy: {allowRegister: false}
apps:
  - {key: explore, name: Explore, redirectUris: [http://localhost/cb]}
`)
	t.Setenv("TEST_PORT", "8091")
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Issuer != "http://localhost:8091" {
		t.Errorf("issuer %q", c.Issuer)
	}
	if c.LoginPolicy.AllowRegister {
		t.Error("extending file should override allowRegister")
	}
	if c.LoginPolicy.PasswordCheckLifetime.Std() != 720*time.Hour {
		t.Errorf("lifetime from base lost: %v", c.LoginPolicy.PasswordCheckLifetime.Std())
	}
	if c.IdPs.GitHub.Enabled() {
		t.Error("GitHub enabled without credentials")
	}
	if c.Apps[0].Project != "Explore" {
		t.Errorf("project should default to the app name, got %q", c.Apps[0].Project)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, "x.yaml", "environment: x\nissuer: http://x\norgs: {operators: a, readers: b}\nloginPolicie: {}\n")
	if _, err := Load(p); err == nil {
		t.Fatal("a misspelt key must fail instead of being ignored")
	}
}

func TestRepositoryConfigsLoad(t *testing.T) {
	for _, env := range []string{"dev", "staging", "production"} {
		if _, err := Load(filepath.Join("..", "..", "config", env+".yaml")); err != nil {
			t.Errorf("%s: %v", env, err)
		}
	}
}
