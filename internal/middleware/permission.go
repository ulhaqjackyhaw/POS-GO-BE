package middleware

import (
	"go-backend/internal/model"

	"github.com/gofiber/fiber/v2"
)

func RequirePermission(permissionCode string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		_, roleCode, _, permissions := GetAuthUser(c)

		// Owner has unrestricted access to all permissions
		if roleCode == "owner" {
			return c.Next()
		}

		for _, p := range permissions {
			if p == permissionCode {
				return c.Next()
			}
		}

		return model.ErrForbidden("Anda tidak memiliki izin yang diperlukan: " + permissionCode)
	}
}
