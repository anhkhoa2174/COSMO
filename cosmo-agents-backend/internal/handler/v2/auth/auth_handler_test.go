package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestAdapterGoogle_ServiceUnavailable(t *testing.T) {
	handler := &AuthHandler{}
	app := fiber.New()
	app.Get("/", handler.AdapterGoogle)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", resp.StatusCode)
	}
}
