package handler

import (
	"go-backend/internal/model"
	"go-backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type OrgHandler struct {
	orgService *service.OrgService
}

func NewOrgHandler(orgService *service.OrgService) *OrgHandler {
	return &OrgHandler{orgService: orgService}
}

func (h *OrgHandler) ListStores(c *fiber.Ctx) error {
	stores, err := h.orgService.ListStores(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(stores)
}

func (h *OrgHandler) GetStore(c *fiber.Ctx) error {
	id := c.Params("store_id")
	store, err := h.orgService.GetStore(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(store)
}

func (h *OrgHandler) CreateStore(c *fiber.Ctx) error {
	var store model.Store
	if err := c.BodyParser(&store); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	if err := h.orgService.CreateStore(c.Context(), &store); err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(store)
}

func (h *OrgHandler) ListWarehouses(c *fiber.Ctx) error {
	storeID := c.Params("store_id")
	warehouses, err := h.orgService.ListWarehouses(c.Context(), storeID)
	if err != nil {
		return err
	}
	return c.JSON(warehouses)
}

func (h *OrgHandler) ListTerminals(c *fiber.Ctx) error {
	storeID := c.Params("store_id")
	terminals, err := h.orgService.ListTerminals(c.Context(), storeID)
	if err != nil {
		return err
	}
	return c.JSON(terminals)
}
