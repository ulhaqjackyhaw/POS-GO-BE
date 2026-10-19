package handler

import (
	"go-backend/internal/middleware"
	"go-backend/internal/model"
	"go-backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) ListRoles(c *fiber.Ctx) error {
	_, creatorRole, _, _ := middleware.GetAuthUser(c)
	roles, err := h.userService.ListRoles(c.Context())
	if err != nil {
		return err
	}

	// Store manager can only assign subordinate roles
	if creatorRole == "store_manager" {
		allowed := map[string]bool{
			"cashier":         true,
			"inventory_staff": true,
			"supervisor":      true,
		}
		var filtered []model.Role
		for _, r := range roles {
			if allowed[r.Code] {
				filtered = append(filtered, r)
			}
		}
		return c.JSON(filtered)
	}

	return c.JSON(roles)
}

func (h *UserHandler) ListUsers(c *fiber.Ctx) error {
	_, creatorRole, creatorStoreIDs, _ := middleware.GetAuthUser(c)
	role := c.Query("role")
	storeID := c.Query("store_id")
	search := c.Query("search")

	if creatorRole == "store_manager" && len(creatorStoreIDs) > 0 && storeID == "" {
		storeID = creatorStoreIDs[0]
	}

	users, err := h.userService.ListUsers(c.Context(), role, storeID, search)
	if err != nil {
		return err
	}

	if creatorRole == "store_manager" && len(creatorStoreIDs) > 0 {
		storeMap := make(map[string]bool)
		for _, sid := range creatorStoreIDs {
			storeMap[sid] = true
		}
		var filtered []model.User
		for _, u := range users {
			if u.RoleCode == "owner" {
				continue
			}
			matched := false
			for _, sid := range u.StoreIDs {
				if storeMap[sid] {
					matched = true
					break
				}
			}
			if matched || len(u.StoreIDs) == 0 {
				filtered = append(filtered, u)
			}
		}
		return c.JSON(filtered)
	}

	return c.JSON(users)
}

func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	id := c.Params("id")
	user, err := h.userService.GetUser(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(user)
}

func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	_, creatorRole, creatorStoreIDs, _ := middleware.GetAuthUser(c)
	var req model.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	if creatorRole == "store_manager" {
		allowedSubordinates := map[string]bool{
			"cashier":         true,
			"inventory_staff": true,
			"supervisor":      true,
		}
		if req.RoleCode != "" && !allowedSubordinates[req.RoleCode] {
			return model.ErrForbidden("Store Manager hanya berhak mendaftarkan staf bawahan (Kasir, Staf Gudang, atau Supervisor)")
		}
		if len(creatorStoreIDs) > 0 {
			req.StoreIDs = creatorStoreIDs
		}
	}

	user, err := h.userService.CreateUser(c.Context(), req)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(user)
}

func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")
	var req model.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	user, err := h.userService.UpdateUser(c.Context(), id, req)
	if err != nil {
		return err
	}
	return c.JSON(user)
}

func (h *UserHandler) ToggleActive(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.BodyParser(&body); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	if err := h.userService.SetUserActive(c.Context(), id, body.IsActive); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusOK)
}
