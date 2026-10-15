package handler

import (
	"go-backend/internal/middleware"
	"go-backend/internal/model"
	"go-backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ApprovalHandler struct {
	approvalService *service.ApprovalService
}

func NewApprovalHandler(approvalService *service.ApprovalService) *ApprovalHandler {
	return &ApprovalHandler{approvalService: approvalService}
}

func (h *ApprovalHandler) CreateRequest(c *fiber.Ctx) error {
	var req model.CreateApprovalRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	userID, _, _, _ := middleware.GetAuthUser(c)
	resp, err := h.approvalService.CreateRequest(c.Context(), req, userID)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *ApprovalHandler) ReviewRequest(c *fiber.Ctx) error {
	id := c.Params("id")
	var req model.ReviewApprovalRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	approverID, _, _, _ := middleware.GetAuthUser(c)
	if err := h.approvalService.ReviewRequest(c.Context(), id, req, approverID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message": "Permintaan approval berhasil ditinjau",
		"status":  req.Status,
	})
}

func (h *ApprovalHandler) ListRequests(c *fiber.Ctx) error {
	storeID := c.Query("store_id")
	status := c.Query("status")

	list, err := h.approvalService.ListRequests(c.Context(), storeID, status)
	if err != nil {
		return err
	}
	return c.JSON(list)
}

func (h *ApprovalHandler) GetApproval(c *fiber.Ctx) error {
	id := c.Params("id")
	item, err := h.approvalService.GetByID(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(item)
}
