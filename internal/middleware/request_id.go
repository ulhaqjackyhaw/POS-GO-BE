package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"
const RequestIDContextKey = "request_id"

func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqID := c.Get(RequestIDHeader)
		if reqID == "" {
			reqID = uuid.NewString()
		}
		c.Locals(RequestIDContextKey, reqID)
		c.Set(RequestIDHeader, reqID)
		return c.Next()
	}
}

func GetRequestID(c *fiber.Ctx) string {
	if val, ok := c.Locals(RequestIDContextKey).(string); ok {
		return val
	}
	return ""
}
