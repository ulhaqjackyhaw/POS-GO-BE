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
	storeID := req.GetStoreID()
	if storeID == "" || req.ProductID == "" || qty == 0 {
		return model.ErrValidation("Store ID, Product ID, dan kuantitas perubahan (!= 0) wajib diisi")
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

	// Lock row in deterministic sorted order
	balances, err := s.invRepo.LockBalancesForUpdate(ctx, tx, storeID, []string{req.ProductID})
	if err != nil {
		return model.ErrInternal("Gagal mengunci baris inventori")
	}

	currentStock := balances[req.ProductID]
	if currentStock+qty < 0 {
		return model.ErrInsufficientStock("Penyesuaian stok tidak dapat membuat saldo menjadi negatif")
	}

	_, err = s.invRepo.MutateStock(
		ctx, tx, storeID, req.ProductID, qty,
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
		StoreID:  &storeID,
		ActorID:  &actorID,
		Action:   "INVENTORY_ADJUST",
		Entity:   "inventory_balances",
		EntityID: req.ProductID,
		Details:  req.Reason,
	})

	return nil
}

func (s *InventoryService) ListLedgers(ctx context.Context, storeID, productID string, limit, offset int) ([]model.InventoryLedger, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.invRepo.ListLedgers(ctx, storeID, productID, limit, offset)
}

func (s *InventoryService) ListStock(ctx context.Context, storeID string) ([]model.CurrentStockItem, error) {
	return s.invRepo.ListStock(ctx, storeID)
}
