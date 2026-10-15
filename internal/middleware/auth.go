package middleware

import (
	"strings"

	"go-backend/internal/model"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

const (
	ContextUserID      = "user_id"
	ContextUsername    = "username"
	ContextRoleCode    = "role_code"
	ContextPermissions = "permissions"
	ContextStoreIDs    = "store_ids"
)

type JWTClaims struct {
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	RoleCode    string   `json:"role_code"`
	Permissions []string `json:"permissions"`
	StoreIDs    []string `json:"store_ids"`
	jwt.RegisteredClaims
}

func AuthRequired(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return model.ErrUnauthenticated("Header Authorization diperlukan")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return model.ErrUnauthenticated("Format token Authorization harus 'Bearer <token>'")
		}

		tokenStr := parts[1]
		claims := &JWTClaims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, model.ErrUnauthenticated("Metode signing token tidak valid")
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			return model.ErrUnauthenticated("Token autentikasi tidak valid atau telah kedaluwarsa")
		}

		c.Locals(ContextUserID, claims.UserID)
		c.Locals(ContextUsername, claims.Username)
		c.Locals(ContextRoleCode, claims.RoleCode)
		c.Locals(ContextPermissions, claims.Permissions)
		c.Locals(ContextStoreIDs, claims.StoreIDs)

		return c.Next()
	}
}

func GetAuthUser(c *fiber.Ctx) (userID, roleCode string, storeIDs, permissions []string) {
	if uid, ok := c.Locals(ContextUserID).(string); ok {
		userID = uid
	}
	if role, ok := c.Locals(ContextRoleCode).(string); ok {
		roleCode = role
	}
	if sids, ok := c.Locals(ContextStoreIDs).([]string); ok {
		storeIDs = sids
	}
	if perms, ok := c.Locals(ContextPermissions).([]string); ok {
		permissions = perms
	}
	return
}
