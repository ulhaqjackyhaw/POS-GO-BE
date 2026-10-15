package handler

import (
	"go-backend/internal/model"
	"go-backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type CatalogHandler struct {
	catalogService *service.CatalogService
}

func NewCatalogHandler(catalogService *service.CatalogService) *CatalogHandler {
	return &CatalogHandler{catalogService: catalogService}
}

func (h *CatalogHandler) ListProducts(c *fiber.Ctx) error {
	storeID := c.Query("store_id")
	if storeID == "" {
		storeID = c.Get("X-Store-ID")
	}
	warehouseID := c.Query("warehouse_id")
	search := c.Query("search")

	products, err := h.catalogService.ListProducts(c.Context(), storeID, warehouseID, search)
	if err != nil {
		return err
	}
	return c.JSON(products)
}

func (h *CatalogHandler) GetProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	storeID := c.Query("store_id")
	if storeID == "" {
		storeID = c.Get("X-Store-ID")
	}
	warehouseID := c.Query("warehouse_id")

	product, err := h.catalogService.GetProduct(c.Context(), id, storeID, warehouseID)
	if err != nil {
		return err
	}
	return c.JSON(product)
}

func (h *CatalogHandler) CreateProduct(c *fiber.Ctx) error {
	var req model.CreateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	product, err := h.catalogService.CreateProduct(c.Context(), req)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(product)
}

func (h *CatalogHandler) SetBranchPrice(c *fiber.Ctx) error {
	storeID := c.Params("store_id")
	if storeID == "" {
		storeID = c.Get("X-Store-ID")
	}
	productID := c.Params("id")

	var req model.SetBranchPriceRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	effectivePrice := req.GetEffectivePrice()
	if err := h.catalogService.SetStorePrice(c.Context(), storeID, productID, effectivePrice); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message":    "Harga cabang berhasil diperbarui",
		"store_id":   storeID,
		"product_id": productID,
		"price":      effectivePrice,
	})
}

func (h *CatalogHandler) ListCategories(c *fiber.Ctx) error {
	categories, err := h.catalogService.ListCategories(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(categories)
}
