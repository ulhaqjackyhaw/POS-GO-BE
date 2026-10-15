package service

import (
	"context"

	"go-backend/internal/model"
	"go-backend/internal/repository"
)

type InventoryService struct {
	invRepo   *repository.InventoryRepository
	auditRepo *repository.AuditRepository
}

func NewInventoryService(invRepo *repository.InventoryRepository, auditRepo *repository.AuditRepository) *InventoryService {
	return &InventoryService{invRepo: invRepo, auditRepo: auditRepo}
}

func (s *InventoryService) AdjustStock(ctx context.Context, req model.StockAdjustmentRequest, actorID string) error {
	qty := req.GetEffectiveQuantity()
	if req.WarehouseID == "" || req.ProductID == "" || qty == 0 {
		return model.ErrValidation("Warehouse ID, Product ID, dan kuantitas perubahan (!= 0) wajib diisi")
	}
	if req.Reason == "" {
		return model.ErrValidation("Alasan penyesuaian stok wajib diisi")
	}

	pool := s.invRepo.GetPool()
	tx, err := pool.Begin(ctx)
	if err != nil {
		return model.ErrInternal("Gagal memulai transaksi database")
	}
	defer tx.Rollback(ctx)

	// Lock row
	balances, err := s.invRepo.LockBalancesForUpdate(ctx, tx, req.WarehouseID, []string{req.ProductID})
	if err != nil {
		return model.ErrInternal("Gagal mengunci baris inventori")
	}

	currentStock := balances[req.ProductID]
	if currentStock+qty < 0 {
		return model.ErrInsufficientStock("Penyesuaian stok tidak dapat membuat saldo menjadi negatif")
	}

	_, err = s.invRepo.MutateStock(
		ctx, tx, req.WarehouseID, req.ProductID, qty,
		"ADJUSTMENT", "STOCK_ADJUSTMENT", "MANUAL", req.Reason, &actorID,
	)
	if err != nil {
		return model.ErrInternal("Gagal memperbarui saldo inventori")
	}

	if err := tx.Commit(ctx); err != nil {
		return model.ErrInternal("Gagal menyimpan penyesuaian stok")
	}

	// Audit trail
	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		ActorID:  &actorID,
		Action:   "INVENTORY_ADJUST",
		Entity:   "inventory_balances",
		EntityID: req.ProductID,
		Details:  req.Reason,
	})

	return nil
}

func (s *InventoryService) ListLedgers(ctx context.Context, warehouseID, productID string, limit, offset int) ([]model.InventoryLedger, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.invRepo.ListLedgers(ctx, warehouseID, productID, limit, offset)
}

func (s *InventoryService) ListStock(ctx context.Context, warehouseID string) ([]model.CurrentStockItem, error) {
	return s.invRepo.ListStock(ctx, warehouseID)
}
