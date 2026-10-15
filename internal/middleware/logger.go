package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

func StructuredLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		reqID := GetRequestID(c)

		err := c.Next()

		latency := time.Since(start)
		status := c.Response().StatusCode()
		method := c.Method()
		path := c.Path()
		ip := c.IP()

		log.Printf("[HTTP] req_id=%s method=%s path=%s status=%d latency=%s ip=%s",
			reqID, method, path, status, latency, ip)

		return err
	}
}
