package middleware

import (
	"go-backend/internal/model"

	"github.com/gofiber/fiber/v2"
)

func RequireStoreAccess() fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, roleCode, storeIDs, _ := GetAuthUser(c)

		// Owner and Auditor have unrestricted global access across all branches
		if roleCode == "owner" || roleCode == "auditor" {
			return c.Next()
		}

		targetStoreID := c.Params("store_id")
		if targetStoreID == "" {
			targetStoreID = c.Query("store_id")
		}
		if targetStoreID == "" {
			targetStoreID = c.Get("X-Store-ID")
		}

		// If no specific store is targeted in path/query/header, let request proceed (service will filter)
		if targetStoreID == "" {
			return c.Next()
		}

		for _, sid := range storeIDs {
			if sid == targetStoreID {
				return c.Next()
			}
		}

		return model.ErrForbidden("Anda tidak memiliki izin untuk mengakses cabang ini")
	}
}
