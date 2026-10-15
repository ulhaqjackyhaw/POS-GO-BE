package service

import (
	"context"
	"fmt"

	"go-backend/internal/model"
	"go-backend/internal/repository"
)

type OfflineSyncService struct {
	posService *POSService
	txRepo     *repository.TransactionRepository
	auditRepo  *repository.AuditRepository
}

func NewOfflineSyncService(posService *POSService, txRepo *repository.TransactionRepository, auditRepo *repository.AuditRepository) *OfflineSyncService {
	return &OfflineSyncService{
		posService: posService,
		txRepo:     txRepo,
		auditRepo:  auditRepo,
	}
}

func (s *OfflineSyncService) ProcessBatch(ctx context.Context, batch model.SyncBatchRequest, cashierID string) (*model.SyncBatchResponse, error) {
	if len(batch.Transactions) == 0 {
		return &model.SyncBatchResponse{TotalProcessed: 0, Results: []model.SyncResultItem{}}, nil
	}

	results := make([]model.SyncResultItem, 0, len(batch.Transactions))
	syncedCount := 0
	conflictCount := 0

	for _, txItem := range batch.Transactions {
		result := model.SyncResultItem{
			OfflineTxID: txItem.OfflineTxID,
		}

		// Mode offline only supports CASH (PRD 6.8 & AGENT.md 5)
		if txItem.PaymentMethod != "CASH" {
			result.Status = "CONFLICT"
			result.ErrorCode = "INVALID_PAYMENT_METHOD"
			result.ErrorMessage = "Mode offline hanya mendukung pembayaran tunai (CASH)"
			conflictCount++
			results = append(results, result)
			continue
		}

		// Idempotency check: see if already synced
		existing, err := s.txRepo.GetByIdempotencyKey(ctx, txItem.IdempotencyKey)
		if err == nil && existing != nil {
			result.Status = "SYNCED"
			result.InvoiceNumber = existing.InvoiceNumber
			syncedCount++
			results = append(results, result)
			continue
		}

		// Attempt checkout
		checkoutReq := model.CheckoutRequest{
			IdempotencyKey: txItem.IdempotencyKey,
			StoreID:        txItem.StoreID,
			TerminalID:     txItem.TerminalID,
			WarehouseID:    txItem.WarehouseID,
			CustomerName:   txItem.CustomerName,
			PaymentMethod:  "CASH",
			PaidAmount:     txItem.PaidAmount,
			Items:          txItem.Items,
		}

		resp, err := s.posService.CheckoutOnline(ctx, checkoutReq, cashierID)
		if err != nil {
			// Check if it's an AppError
			result.Status = "CONFLICT"
			if appErr, ok := err.(*model.AppError); ok {
				result.ErrorCode = appErr.Code
				result.ErrorMessage = appErr.Message
			} else {
				result.ErrorCode = "SYNC_FAILED"
				result.ErrorMessage = err.Error()
			}
			conflictCount++
		} else {
			result.Status = "SYNCED"
			result.InvoiceNumber = resp.InvoiceNumber
			syncedCount++
		}

		results = append(results, result)
	}

	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		ActorID:  &cashierID,
		Action:   "OFFLINE_SYNC_BATCH",
		Entity:   "sync_batches",
		EntityID: batch.TerminalID,
		Details:  fmt.Sprintf(`{"total": %d, "synced": %d, "conflict": %d}`, len(batch.Transactions), syncedCount, conflictCount),
	})

	return &model.SyncBatchResponse{
		TotalProcessed: len(batch.Transactions),
		SyncedCount:    syncedCount,
		ConflictCount:  conflictCount,
		Results:        results,
	}, nil
}
