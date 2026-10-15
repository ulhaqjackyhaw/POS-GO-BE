package handler

import (
	"strconv"

	"go-backend/internal/middleware"
	"go-backend/internal/model"
	"go-backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type POSHandler struct {
	posService *service.POSService
}

func NewPOSHandler(posService *service.POSService) *POSHandler {
	return &POSHandler{posService: posService}
}

func (h *POSHandler) GenerateQuote(c *fiber.Ctx) error {
	var req model.QuoteRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	if req.StoreID == "" {
		req.StoreID = c.Get("X-Store-ID")
	}

	warehouseID := c.Query("warehouse_id")
	resp, err := h.posService.GenerateQuote(c.Context(), req, warehouseID)
	if err != nil {
		return err
	}
	return c.JSON(resp)
}

func (h *POSHandler) Checkout(c *fiber.Ctx) error {
	var req model.CheckoutRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	req.Normalize()

	// Store ID fallback to header
	if req.StoreID == "" {
		req.StoreID = c.Get("X-Store-ID")
	}

	// Idempotency key can also come from header
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = c.Get("Idempotency-Key")
	}

	cashierID, _, _, _ := middleware.GetAuthUser(c)
	resp, err := h.posService.CheckoutOnline(c.Context(), req, cashierID)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *POSHandler) ListTransactions(c *fiber.Ctx) error {
	storeID := c.Query("store_id")
	if storeID == "" {
		storeID = c.Get("X-Store-ID")
	}
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	list, err := h.posService.ListTransactions(c.Context(), storeID, limit, offset)
	if err != nil {
		return err
	}
	return c.JSON(list)
}

func (h *POSHandler) GetTransaction(c *fiber.Ctx) error {
	id := c.Params("id")
	tx, err := h.posService.GetTransaction(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(tx)
}

func (h *POSHandler) VoidTransaction(c *fiber.Ctx) error {
	id := c.Params("id")
	var req model.VoidRefundRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	warehouseID := c.Query("warehouse_id")
	cashierID, _, _, _ := middleware.GetAuthUser(c)

	if err := h.posService.VoidTransaction(c.Context(), id, req.Reason, cashierID, warehouseID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message": "Transaksi berhasil dibatalkan (void)",
		"status":  "VOIDED",
	})
}

func (h *POSHandler) RefundTransaction(c *fiber.Ctx) error {
	id := c.Params("id")
	var req model.VoidRefundRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	warehouseID := c.Query("warehouse_id")
	cashierID, _, _, _ := middleware.GetAuthUser(c)

	if err := h.posService.RefundTransaction(c.Context(), id, req.Reason, cashierID, warehouseID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message": "Transaksi berhasil di-refund",
		"status":  "REFUNDED",
	})
}
