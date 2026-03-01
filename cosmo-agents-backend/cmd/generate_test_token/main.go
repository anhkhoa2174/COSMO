package main

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/pkg/auth"
)

func main() {
	// Read JWT secret from environment or use default
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-super-secret-jwt-key-change-in-production"
	}

	// Test user ID from database
	userID := uuid.MustParse("d074e45e-8c11-44c4-9ef0-94e83e563bda")
	email := "tkhoa4040@gmail.com"
	orgID := uuid.Nil // or set actual organization ID if needed

	// Create JWT manager
	jwtManager := auth.NewJWTManager(secret, 24*time.Hour, "", 0, "HS256")

	// Generate token
	token, err := jwtManager.GenerateToken(userID, email, orgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(token)
}
