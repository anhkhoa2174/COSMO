// Command devtoken mints a JWT for local development and evaluation tooling.
// Usage: go run ./cmd/devtoken <user-uuid> <email>
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/rockship/cosmo-agents-go/pkg/auth"
	"github.com/rockship/cosmo-agents-go/pkg/config"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: devtoken <user-uuid> <email>")
		os.Exit(2)
	}
	_ = godotenv.Load(".env")
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	// Mirror cmd/server/app.go: the server validates with the access-token
	// secret when it is set, falling back to JWT_SECRET.
	secret := cfg.Auth.AccessTokenSecret
	if secret == "" {
		secret = cfg.Auth.JWTSecret
	}
	m := auth.NewJWTManager(secret, 24*time.Hour, secret, 24*time.Hour, cfg.Auth.JWTAlgorithm)
	if cfg.Auth.JWTIssuer != "" {
		m.SetIssuer(cfg.Auth.JWTIssuer)
	}
	if cfg.Auth.JWTAudience != "" {
		m.SetAudience(cfg.Auth.JWTAudience)
	}
	tok, err := m.GenerateToken(uuid.MustParse(os.Args[1]), os.Args[2], uuid.Nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sign:", err)
		os.Exit(1)
	}
	fmt.Print(tok)
}
