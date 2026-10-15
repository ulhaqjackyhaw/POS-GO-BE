package middleware

import (
	"errors"
	"log"

	"go-backend/internal/model"

	"github.com/gofiber/fiber/v2"
)

func CustomErrorHandler(c *fiber.Ctx, err error) error {
	reqID := GetRequestID(c)

	var appErr *model.AppError
	if errors.As(err, &appErr) {
		return c.Status(appErr.StatusCode).JSON(fiber.Map{
			"error": appErr.Message,
			"code":  appErr.Code,
		})
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		code := "INTERNAL_ERROR"
		switch fiberErr.Code {
		case fiber.StatusBadRequest:
			code = "VALIDATION_ERROR"
		case fiber.StatusUnauthorized:
			code = "UNAUTHENTICATED"
		case fiber.StatusForbidden:
			code = "FORBIDDEN"
		case fiber.StatusNotFound:
			code = "NOT_FOUND"
		case fiber.StatusConflict:
			code = "CONFLICT"
		}

		return c.Status(fiberErr.Code).JSON(fiber.Map{
			"error": fiberErr.Message,
			"code":  code,
		})
	}

	// For unexpected internal errors: do NOT expose stack trace or SQL details to client
	log.Printf("[ERROR] req_id=%s unhandled_error: %v", reqID, err)

	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"error": "Terjadi kesalahan internal pada server",
		"code":  "INTERNAL_SERVER_ERROR",
	})
}
