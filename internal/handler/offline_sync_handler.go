package handler

import (
	"go-backend/internal/middleware"
	"go-backend/internal/model"
	"go-backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type OfflineSyncHandler struct {
	syncService *service.OfflineSyncService
}

func NewOfflineSyncHandler(syncService *service.OfflineSyncService) *OfflineSyncHandler {
	return &OfflineSyncHandler{syncService: syncService}
}

func (h *OfflineSyncHandler) SyncBatch(c *fiber.Ctx) error {
	var batch model.SyncBatchRequest
	if err := c.BodyParser(&batch); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	cashierID, _, _, _ := middleware.GetAuthUser(c)
	resp, err := h.syncService.ProcessBatch(c.Context(), batch, cashierID)
	if err != nil {
		return err
	}
	return c.JSON(resp)
}
