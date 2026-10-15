package handler

import (
	"go-backend/internal/middleware"
	"go-backend/internal/model"
	"go-backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ShiftHandler struct {
	shiftService *service.ShiftService
}

func NewShiftHandler(shiftService *service.ShiftService) *ShiftHandler {
	return &ShiftHandler{shiftService: shiftService}
}

func (h *ShiftHandler) GetActiveShift(c *fiber.Ctx) error {
	storeID := c.Query("store_id")
	if storeID == "" {
		storeID = c.Get("X-Store-ID")
	}
	terminalID := c.Query("terminal_id")
	userID, _, _, _ := middleware.GetAuthUser(c)

	shift, err := h.shiftService.GetActiveShift(c.Context(), storeID, terminalID, userID)
	if err != nil {
		return err
	}
	if shift == nil {
		return model.ErrNotFound("Tidak ada shift aktif untuk pengguna ini")
	}
	return c.JSON(shift)
}

func (h *ShiftHandler) GetShiftDetail(c *fiber.Ctx) error {
	id := c.Params("id")
	detail, err := h.shiftService.GetShiftDetail(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(detail)
}

func (h *ShiftHandler) OpenShift(c *fiber.Ctx) error {
	var req model.OpenShiftRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	if req.StoreID == "" {
		req.StoreID = c.Get("X-Store-ID")
	}

	userID, _, _, _ := middleware.GetAuthUser(c)
	shift, err := h.shiftService.OpenShift(c.Context(), req, userID)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(shift)
}

func (h *ShiftHandler) CloseShift(c *fiber.Ctx) error {
	shiftID := c.Params("id")
	var req model.CloseShiftRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	userID, _, _, _ := middleware.GetAuthUser(c)
	shift, err := h.shiftService.CloseShift(c.Context(), shiftID, req, userID)
	if err != nil {
		return err
	}
	return c.JSON(shift)
}

func (h *ShiftHandler) CreateCashMovement(c *fiber.Ctx) error {
	shiftID := c.Params("id")
	var req model.CreateCashMovementRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	userID, _, _, _ := middleware.GetAuthUser(c)
	if err := h.shiftService.CreateCashMovement(c.Context(), req, shiftID, userID); err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Pergerakan kas berhasil dicatat",
	})
}
