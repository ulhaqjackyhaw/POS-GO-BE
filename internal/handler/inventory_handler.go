package handler

import (
	"strconv"

	"go-backend/internal/middleware"
	"go-backend/internal/model"
	"go-backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type InventoryHandler struct {
	invService *service.InventoryService
}

func NewInventoryHandler(invService *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{invService: invService}
}

func (h *InventoryHandler) AdjustStock(c *fiber.Ctx) error {
	var req model.StockAdjustmentRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	if req.StoreID == "" {
		req.StoreID = c.Query("store_id")
	}
	if req.StoreID == "" {
		req.StoreID = c.Get("X-Store-ID")
	}

	userID, _, _, _ := middleware.GetAuthUser(c)
	if err := h.invService.AdjustStock(c.Context(), req, userID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message": "Penyesuaian stok berhasil dicatat",
	})
}

func (h *InventoryHandler) ListLedgers(c *fiber.Ctx) error {
	storeID := c.Query("store_id")
	if storeID == "" {
		storeID = c.Query("warehouse_id")
	}
	if storeID == "" {
		storeID = c.Get("X-Store-ID")
	}
	productID := c.Query("product_id")
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	ledgers, err := h.invService.ListLedgers(c.Context(), storeID, productID, limit, offset)
	if err != nil {
		return err
	}
	return c.JSON(ledgers)
}

func (h *InventoryHandler) ListStock(c *fiber.Ctx) error {
	storeID := c.Query("store_id")
	if storeID == "" {
		storeID = c.Query("warehouse_id")
	}
	if storeID == "" {
		storeID = c.Get("X-Store-ID")
	}
	stock, err := h.invService.ListStock(c.Context(), storeID)
	if err != nil {
		return err
	}
	return c.JSON(stock)
}
