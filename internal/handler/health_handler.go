package handler

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	pool *pgxpool.Pool
}

func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{pool: pool}
}

func (h *HealthHandler) Health(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "healthy"
	if err := h.pool.Ping(ctx); err != nil {
		dbStatus = "unhealthy"
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status":   "unhealthy",
			"database": dbStatus,
		})
	}

	return c.JSON(fiber.Map{
		"status":   "healthy",
		"database": dbStatus,
		"time":     time.Now().UTC().Format(time.RFC3339),
	})
}
