package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-backend/internal/middleware"
	"go-backend/internal/model"

	"github.com/gofiber/fiber/v2"
)

func TestAPIContractErrorFormat(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.CustomErrorHandler,
	})

	app.Get("/test/validation-error", func(c *fiber.Ctx) error {
		return model.ErrValidation("Input nama tidak boleh kosong")
	})

	app.Get("/test/unauthenticated", func(c *fiber.Ctx) error {
		return model.ErrUnauthenticated("Token autentikasi diperlukan")
	})

	app.Get("/test/forbidden", func(c *fiber.Ctx) error {
		return model.ErrForbidden("Hak akses tidak mencukupi")
	})

	app.Get("/test/not-found", func(c *fiber.Ctx) error {
		return model.ErrNotFound("Produk tidak ditemukan")
	})

	app.Get("/test/insufficient-stock", func(c *fiber.Ctx) error {
		return model.ErrInsufficientStock("Stok tidak mencukupi")
	})

	app.Get("/test/idempotency-conflict", func(c *fiber.Ctx) error {
		return model.ErrIdempotencyConflict("Idempotency key telah digunakan")
	})

	cases := []struct {
		path         string
		expectedCode int
		expectedMsg  string
		expectedErr  string
	}{
		{"/test/validation-error", 400, "Input nama tidak boleh kosong", "VALIDATION_ERROR"},
		{"/test/unauthenticated", 401, "Token autentikasi diperlukan", "UNAUTHENTICATED"},
		{"/test/forbidden", 403, "Hak akses tidak mencukupi", "FORBIDDEN"},
		{"/test/not-found", 404, "Produk tidak ditemukan", "NOT_FOUND"},
		{"/test/insufficient-stock", 400, "Stok tidak mencukupi", "INSUFFICIENT_STOCK"},
		{"/test/idempotency-conflict", 409, "Idempotency key telah digunakan", "IDEMPOTENCY_CONFLICT"},
	}

	for _, tc := range cases {
		t.Run(tc.expectedErr, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test request: %v", err)
			}

			if resp.StatusCode != tc.expectedCode {
				t.Errorf("Expected status %d, got %d", tc.expectedCode, resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			var resMap map[string]interface{}
			if err := json.Unmarshal(body, &resMap); err != nil {
				t.Fatalf("Response is not valid JSON: %s", string(body))
			}

			if resMap["error"] != tc.expectedMsg {
				t.Errorf("Expected error '%s', got '%v'", tc.expectedMsg, resMap["error"])
			}
			if resMap["code"] != tc.expectedErr {
				t.Errorf("Expected code '%s', got '%v'", tc.expectedErr, resMap["code"])
			}
		})
	}
}
