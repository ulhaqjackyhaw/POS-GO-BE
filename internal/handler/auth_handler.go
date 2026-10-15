package handler

import (
	"go-backend/internal/middleware"
	"go-backend/internal/model"
	"go-backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	resp, err := h.authService.Login(c.Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(resp)
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID, roleCode, storeIDs, permissions := middleware.GetAuthUser(c)
	username, _ := c.Locals(middleware.ContextUsername).(string)

	return c.JSON(model.UserInfo{
		ID:          userID,
		Username:    username,
		Role:        roleCode,
		Permissions: permissions,
		StoreIDs:    storeIDs,
	})
}
