package handler

import (
	"go-backend/internal/middleware"
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
	_, roleCode, storeIDs, _ := middleware.GetAuthUser(c)
	stores, err := h.orgService.ListStores(c.Context())
	if err != nil {
		return err
	}

	// Owner and auditor (or unauthenticated/admin without store limits) see all stores
	if roleCode == "owner" || roleCode == "auditor" || roleCode == "admin" || len(storeIDs) == 0 {
		return c.JSON(stores)
	}

	// For users with custom store placements, filter to only assigned stores
	storeMap := make(map[string]bool)
	for _, sid := range storeIDs {
		storeMap[sid] = true
	}

	filtered := make([]model.Store, 0)
	for _, s := range stores {
		if storeMap[s.ID] {
			filtered = append(filtered, s)
		}
	}
	return c.JSON(filtered)
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

func (h *OrgHandler) ListAllWarehouses(c *fiber.Ctx) error {
	storeID := c.Query("store_id")
	warehouses, err := h.orgService.ListWarehouses(c.Context(), storeID)
	if err != nil {
		return err
	}
	return c.JSON(warehouses)
}

func (h *OrgHandler) GetWarehouse(c *fiber.Ctx) error {
	id := c.Params("id")
	warehouse, err := h.orgService.GetWarehouse(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(warehouse)
}

func (h *OrgHandler) CreateWarehouse(c *fiber.Ctx) error {
	var warehouse model.Warehouse
	if err := c.BodyParser(&warehouse); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	if storeID := c.Params("store_id"); storeID != "" && warehouse.StoreID == "" {
		warehouse.StoreID = storeID
	}

	if err := h.orgService.CreateWarehouse(c.Context(), &warehouse); err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(warehouse)
}

func (h *OrgHandler) UpdateWarehouse(c *fiber.Ctx) error {
	id := c.Params("id")
	var warehouse model.Warehouse
	if err := c.BodyParser(&warehouse); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	if err := h.orgService.UpdateWarehouse(c.Context(), id, &warehouse); err != nil {
		return err
	}
	return c.JSON(warehouse)
}

func (h *OrgHandler) DeleteWarehouse(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.orgService.DeleteWarehouse(c.Context(), id); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *OrgHandler) ListTerminals(c *fiber.Ctx) error {
	storeID := c.Params("store_id")
	_, roleCode, storeIDs, _ := middleware.GetAuthUser(c)

	// Role manager ke bawah (non-owner & non-auditor) cannot view other branches' terminals
	if roleCode != "owner" && roleCode != "auditor" && len(storeIDs) > 0 {
		matched := false
		for _, sid := range storeIDs {
			if sid == storeID {
				matched = true
				break
			}
		}
		if !matched {
			return model.ErrForbidden("Anda hanya memiliki akses melihat terminal pada cabang penugasan Anda")
		}
	}

	terminals, err := h.orgService.ListTerminals(c.Context(), storeID)
	if err != nil {
		return err
	}
	return c.JSON(terminals)
}

func (h *OrgHandler) ListAllTerminals(c *fiber.Ctx) error {
	_, roleCode, storeIDs, _ := middleware.GetAuthUser(c)
	storeID := c.Query("store_id")

	// Role manager ke bawah: lock to their assigned branch only
	if roleCode != "owner" && roleCode != "auditor" && len(storeIDs) > 0 {
		if storeID == "" {
			storeID = storeIDs[0]
		} else {
			matched := false
			for _, sid := range storeIDs {
				if sid == storeID {
					matched = true
					break
				}
			}
			if !matched {
				storeID = storeIDs[0]
			}
		}
	}

	terminals, err := h.orgService.ListTerminals(c.Context(), storeID)
	if err != nil {
		return err
	}

	if roleCode != "owner" && roleCode != "auditor" && len(storeIDs) > 0 {
		allowedMap := make(map[string]bool)
		for _, sid := range storeIDs {
			allowedMap[sid] = true
		}
		var filtered []model.Terminal
		for _, t := range terminals {
			if allowedMap[t.StoreID] {
				filtered = append(filtered, t)
			}
		}
		return c.JSON(filtered)
	}

	return c.JSON(terminals)
}

func (h *OrgHandler) GetTerminal(c *fiber.Ctx) error {
	id := c.Params("id")
	terminal, err := h.orgService.GetTerminal(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(terminal)
}

func (h *OrgHandler) CreateTerminal(c *fiber.Ctx) error {
	_, roleCode, storeIDs, _ := middleware.GetAuthUser(c)
	var req model.CreateTerminalRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	if req.StoreID == "" {
		req.StoreID = c.Params("store_id")
	}

	if roleCode != "owner" && roleCode != "auditor" && len(storeIDs) > 0 {
		if req.StoreID == "" {
			req.StoreID = storeIDs[0]
		} else {
			matched := false
			for _, sid := range storeIDs {
				if sid == req.StoreID {
					matched = true
					break
				}
			}
			if !matched {
				return model.ErrForbidden("Hanya dapat mendaftarkan terminal untuk cabang yang ditugaskan")
			}
		}
	}

	terminal := model.Terminal{
		StoreID:           req.StoreID,
		Code:              req.Code,
		Name:              req.Name,
		AssignedCashierID: req.AssignedCashierID,
	}

	if err := h.orgService.CreateTerminal(c.Context(), &terminal); err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(terminal)
}

func (h *OrgHandler) UpdateTerminal(c *fiber.Ctx) error {
	id := c.Params("id")
	_, roleCode, storeIDs, _ := middleware.GetAuthUser(c)

	if roleCode != "owner" && roleCode != "auditor" && len(storeIDs) > 0 {
		term, err := h.orgService.GetTerminal(c.Context(), id)
		if err != nil {
			return err
		}
		if term == nil {
			return model.ErrNotFound("Terminal tidak ditemukan")
		}
		matched := false
		for _, sid := range storeIDs {
			if sid == term.StoreID {
				matched = true
				break
			}
		}
		if !matched {
			return model.ErrForbidden("Anda hanya dapat mengubah terminal cabang penugasan Anda")
		}
	}

	var req model.UpdateTerminalRequest
	if err := c.BodyParser(&req); err != nil {
		return model.ErrValidation("Format request JSON tidak valid")
	}

	terminal, err := h.orgService.UpdateTerminal(c.Context(), id, req)
	if err != nil {
		return err
	}
	return c.JSON(terminal)
}

func (h *OrgHandler) DeleteTerminal(c *fiber.Ctx) error {
	id := c.Params("id")
	_, roleCode, storeIDs, _ := middleware.GetAuthUser(c)

	if roleCode != "owner" && roleCode != "auditor" && len(storeIDs) > 0 {
		term, err := h.orgService.GetTerminal(c.Context(), id)
		if err != nil {
			return err
		}
		if term == nil {
			return model.ErrNotFound("Terminal tidak ditemukan")
		}
		matched := false
		for _, sid := range storeIDs {
			if sid == term.StoreID {
				matched = true
				break
			}
		}
		if !matched {
			return model.ErrForbidden("Anda hanya dapat menghapus terminal cabang penugasan Anda")
		}
	}

	if err := h.orgService.DeleteTerminal(c.Context(), id); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}
