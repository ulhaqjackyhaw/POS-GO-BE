package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"go-backend/internal/model"
	"go-backend/internal/repository"
)

type POSService struct {
	txRepo      *repository.TransactionRepository
	invRepo     *repository.InventoryRepository
	catalogRepo *repository.CatalogRepository
	shiftRepo   *repository.ShiftRepository
	auditRepo   *repository.AuditRepository
	taxRateBps  int64 // Basis points: 1100 = 11.00%
}

func NewPOSService(
	txRepo *repository.TransactionRepository,
	invRepo *repository.InventoryRepository,
	catalogRepo *repository.CatalogRepository,
	shiftRepo *repository.ShiftRepository,
	auditRepo *repository.AuditRepository,
	taxRateBps int64,
) *POSService {
	return &POSService{
		txRepo:      txRepo,
		invRepo:     invRepo,
		catalogRepo: catalogRepo,
		shiftRepo:   shiftRepo,
		auditRepo:   auditRepo,
		taxRateBps:  taxRateBps,
	}
}

// CalculateTaxInclusive calculates tax component from tax-inclusive amount deterministically
// Formula from PRD 6.3: tax = amount * r / (1 + r)
// Using basis points to avoid floating-point math:
// tax = (amount * bps) / (10000 + bps)
func (s *POSService) CalculateTaxInclusive(amount int64) int64 {
	if amount <= 0 || s.taxRateBps <= 0 {
		return 0
	}
	return (amount * s.taxRateBps) / (10000 + s.taxRateBps)
}

func (s *POSService) ComputePayloadHash(req model.CheckoutRequest) string {
	req.Normalize()
	type itemPayload struct {
		ProductID      string `json:"product_id"`
		Quantity       int64  `json:"quantity"`
		DiscountAmount int64  `json:"discount_amount"`
	}
	items := make([]itemPayload, len(req.Items))
	for i, it := range req.Items {
		items[i] = itemPayload{
			ProductID:      it.ProductID,
			Quantity:       it.Quantity,
			DiscountAmount: it.DiscountAmount,
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ProductID < items[j].ProductID
	})

	data := struct {
		StoreID       string        `json:"store_id"`
		WarehouseID   string        `json:"warehouse_id"`
		PaidAmount    int64         `json:"paid_amount"`
		PaymentMethod string        `json:"payment_method"`
		Items         []itemPayload `json:"items"`
	}{
		StoreID:       req.StoreID,
		WarehouseID:   req.WarehouseID,
		PaidAmount:    req.PaidAmount,
		PaymentMethod: req.PaymentMethod,
		Items:         items,
	}

	bytes, _ := json.Marshal(data)
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:])
}

// GenerateQuote calculates prices, discounts, and tax breakdown for checkout preview
func (s *POSService) GenerateQuote(ctx context.Context, req model.QuoteRequest, warehouseID string) (*model.QuoteResponse, error) {
	if req.StoreID == "" || len(req.Items) == 0 {
		return nil, model.ErrValidation("StoreID dan daftar item tidak boleh kosong")
	}

	var subtotalTotal int64 = 0
	var discountTotal int64 = 0
	var taxTotal int64 = 0
	var grandTotal int64 = 0
	var responseItems []model.QuoteItemResponse

	for _, it := range req.Items {
		if it.Quantity <= 0 {
			return nil, model.ErrValidation("Kuantitas produk harus lebih dari 0")
		}

		productWithPrice, err := s.catalogRepo.GetProductWithEffectivePrice(ctx, it.ProductID, req.StoreID, warehouseID)
		if err != nil || productWithPrice == nil {
			return nil, model.ErrNotFound(fmt.Sprintf("Produk dengan ID %s tidak ditemukan", it.ProductID))
		}

		itemSubtotal := productWithPrice.EffectivePrice * it.Quantity
		itemTotal := itemSubtotal - it.DiscountAmount
		if itemTotal < 0 {
			itemTotal = 0
		}

		itemTax := s.CalculateTaxInclusive(itemTotal)

		subtotalTotal += itemSubtotal
		discountTotal += it.DiscountAmount
		taxTotal += itemTax
		grandTotal += itemTotal

		responseItems = append(responseItems, model.QuoteItemResponse{
			ProductID:      productWithPrice.ID,
			ProductName:    productWithPrice.Name,
			UnitPrice:      productWithPrice.EffectivePrice,
			UnitPriceMinor: productWithPrice.EffectivePrice,
			Quantity:       it.Quantity,
			Subtotal:       itemSubtotal,
			DiscountAmount: it.DiscountAmount,
			DiscountMinor:  it.DiscountAmount,
			TaxAmount:      itemTax,
			TaxMinor:       itemTax,
			Total:          itemTotal,
			LineTotalMinor: itemTotal,
			AvailableStock: productWithPrice.CurrentStock,
		})
	}

	return &model.QuoteResponse{
		StoreID:        req.StoreID,
		Subtotal:       subtotalTotal,
		SubtotalMinor:  subtotalTotal,
		DiscountAmount: discountTotal,
		DiscountMinor:  discountTotal,
		TaxAmount:      taxTotal,
		TaxMinor:       taxTotal,
		TotalAmount:    grandTotal,
		TotalMinor:     grandTotal,
		Items:          responseItems,
		Lines:          responseItems,
	}, nil
}

// CheckoutOnline processes sales transactions atomically with strict idempotency and deadlock-free locking
func (s *POSService) CheckoutOnline(ctx context.Context, req model.CheckoutRequest, cashierID string) (*model.CheckoutResponse, error) {
	req.Normalize()
	if req.IdempotencyKey == "" {
		return nil, model.ErrValidation("Idempotency key wajib disertakan untuk checkout")
	}
	if req.StoreID == "" || req.WarehouseID == "" || len(req.Items) == 0 {
		return nil, model.ErrValidation("StoreID, WarehouseID, dan items tidak boleh kosong")
	}

	currentHash := s.ComputePayloadHash(req)

	// 1. Idempotency Check
	existingTx, err := s.txRepo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
	if err != nil {
		return nil, model.ErrInternal("Gagal memeriksa idempotency key")
	}
	if existingTx != nil {
		if existingTx.PayloadHash == currentHash {
			// Idempotent retry with identical payload: return cached existing transaction
			return &model.CheckoutResponse{
				TransactionID: existingTx.ID,
				InvoiceNumber: existingTx.InvoiceNumber,
				ReceiptNumber: existingTx.InvoiceNumber,
				Status:        existingTx.Status,
				Subtotal:      existingTx.Subtotal,
				DiscountTotal: existingTx.DiscountAmount,
				TaxAmount:     existingTx.TaxAmount,
				TotalAmount:   existingTx.TotalAmount,
				TotalMinor:    existingTx.TotalAmount,
				PaidAmount:    existingTx.PaidAmount,
				ChangeAmount:  existingTx.ChangeAmount,
				ChangeMinor:   existingTx.ChangeAmount,
				CreatedAt:     existingTx.CreatedAt,
				Transaction:   existingTx,
			}, nil
		}
		// Key matches but payload differs -> HTTP 409 conflict
		return nil, model.ErrIdempotencyConflict("Idempotency key telah digunakan dengan detail transaksi yang berbeda")
	}

	// 2. Validate active shift for cashier
	activeShift, err := s.shiftRepo.GetActiveShift(ctx, req.StoreID, "", cashierID)
	if err != nil {
		return nil, model.ErrInternal("Gagal memeriksa shift kasir")
	}
	if activeShift == nil {
		return nil, model.ErrShiftNotOpen("Kasir harus memiliki shift aktif yang terbuka untuk melakukan penjualan")
	}

	// 3. Aggregate quantities and prepare product IDs
	qtyMap := make(map[string]int64)
	productIDs := make([]string, 0, len(req.Items))
	for _, it := range req.Items {
		if it.Quantity <= 0 {
			return nil, model.ErrValidation("Kuantitas produk harus lebih dari 0")
		}
		if _, exists := qtyMap[it.ProductID]; !exists {
			productIDs = append(productIDs, it.ProductID)
		}
		qtyMap[it.ProductID] += it.Quantity
	}

	pool := s.invRepo.GetPool()
	dbTx, err := pool.Begin(ctx)
	if err != nil {
		return nil, model.ErrInternal("Gagal memulai transaksi database")
	}
	defer dbTx.Rollback(ctx)

	// 4. Deadlock-free row-level locking on inventory balances in deterministic sorted order
	balances, err := s.invRepo.LockBalancesForUpdate(ctx, dbTx, req.WarehouseID, productIDs)
	if err != nil {
		return nil, model.ErrInternal("Gagal mengunci stok inventori")
	}

	// 5. Verify stock availability and resolve prices
	var subtotalTotal int64 = 0
	var discountTotal int64 = 0
	var grandTotal int64 = 0
	var txItems []model.SalesTransactionItem

	for _, it := range req.Items {
		reqQty := it.Quantity
		currentStock := balances[it.ProductID]

		productWithPrice, err := s.catalogRepo.GetProductWithEffectivePrice(ctx, it.ProductID, req.StoreID, req.WarehouseID)
		if err != nil || productWithPrice == nil {
			return nil, model.ErrNotFound(fmt.Sprintf("Produk %s tidak ditemukan", it.ProductID))
		}

		if currentStock < qtyMap[it.ProductID] {
			return nil, model.ErrInsufficientStock(fmt.Sprintf("Stok tidak mencukupi untuk produk: %s (tersedia: %d, diminta: %d)", productWithPrice.Name, currentStock, qtyMap[it.ProductID]))
		}

		lineSubtotal := productWithPrice.EffectivePrice * reqQty
		lineTotal := lineSubtotal - it.DiscountAmount
		if lineTotal < 0 {
			lineTotal = 0
		}
		lineTax := s.CalculateTaxInclusive(lineTotal)

		subtotalTotal += lineSubtotal
		discountTotal += it.DiscountAmount
		grandTotal += lineTotal

		txItems = append(txItems, model.SalesTransactionItem{
			ProductID:      productWithPrice.ID,
			ProductName:    productWithPrice.Name,
			UnitPrice:      productWithPrice.EffectivePrice,
			Quantity:       reqQty,
			Subtotal:       lineSubtotal,
			DiscountAmount: it.DiscountAmount,
			TaxAmount:      lineTax,
			Total:          lineTotal,
		})
	}

	// 6. Validate payment amount for cash
	if req.PaidAmount < grandTotal {
		return nil, model.ErrValidation(fmt.Sprintf("Jumlah bayar (%d) kurang dari total belanja (%d)", req.PaidAmount, grandTotal))
	}
	changeAmount := req.PaidAmount - grandTotal
	totalTax := s.CalculateTaxInclusive(grandTotal)

	// 7. Deduct stock atomically and append immutable ledger entries
	invoiceNum := fmt.Sprintf("INV-%s-%d", time.Now().Format("20060102"), time.Now().UnixNano()%1000000)

	for pid, qty := range qtyMap {
		_, err := s.invRepo.MutateStock(
			ctx, dbTx, req.WarehouseID, pid, -qty,
			"SALE", "TRANSACTION", invoiceNum, "Penjualan kasir online", &cashierID,
		)
		if err != nil {
			return nil, model.ErrInternal(fmt.Sprintf("Gagal memotong stok produk %s: %v", pid, err))
		}
	}

	// 8. Insert Sales Transaction & Items
	salesTx := &model.SalesTransaction{
		InvoiceNumber:  invoiceNum,
		IdempotencyKey: req.IdempotencyKey,
		StoreID:        req.StoreID,
		TerminalID:     req.TerminalID,
		ShiftID:        &activeShift.ID,
		CashierID:      cashierID,
		CustomerName:   req.CustomerName,
		Subtotal:       subtotalTotal,
		DiscountAmount: discountTotal,
		TaxAmount:      totalTax,
		TotalAmount:    grandTotal,
		PaymentMethod:  "CASH",
		PaidAmount:     req.PaidAmount,
		ChangeAmount:   changeAmount,
		Status:         "COMPLETED",
		PayloadHash:    currentHash,
		Items:          txItems,
	}

	if err := s.txRepo.CreateTransaction(ctx, dbTx, salesTx); err != nil {
		return nil, model.ErrInternal("Gagal menyimpan transaksi penjualan")
	}

	// Commit DB Transaction
	if err := dbTx.Commit(ctx); err != nil {
		return nil, model.ErrInternal("Gagal commit transaksi checkout")
	}

	// 9. Audit trail
	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		StoreID:  &req.StoreID,
		ActorID:  &cashierID,
		Action:   "POS_CHECKOUT",
		Entity:   "sales_transactions",
		EntityID: salesTx.ID,
		Details:  fmt.Sprintf(`{"invoice": "%s", "total": %d}`, invoiceNum, grandTotal),
	})

	return &model.CheckoutResponse{
		TransactionID: salesTx.ID,
		InvoiceNumber: invoiceNum,
		ReceiptNumber: invoiceNum,
		Status:        "COMPLETED",
		Subtotal:      subtotalTotal,
		DiscountTotal: discountTotal,
		TaxAmount:     totalTax,
		TotalAmount:   grandTotal,
		TotalMinor:    grandTotal,
		PaidAmount:    req.PaidAmount,
		ChangeAmount:  changeAmount,
		ChangeMinor:   changeAmount,
		CreatedAt:     salesTx.CreatedAt,
		Transaction:   salesTx,
	}, nil
}

// VoidTransaction compensates a transaction, reverses stock in ledger, and keeps immutable history
func (s *POSService) VoidTransaction(ctx context.Context, txID, reason, actorID string, warehouseID string) error {
	if reason == "" {
		return model.ErrValidation("Alasan pembatalan (void) wajib diisi")
	}

	txRecord, err := s.txRepo.GetByID(ctx, txID)
	if err != nil || txRecord == nil {
		return model.ErrNotFound("Transaksi penjualan tidak ditemukan")
	}

	if txRecord.Status != "COMPLETED" {
		return model.ErrValidation(fmt.Sprintf("Transaksi tidak dapat di-void karena berstatus '%s'", txRecord.Status))
	}

	pool := s.invRepo.GetPool()
	dbTx, err := pool.Begin(ctx)
	if err != nil {
		return model.ErrInternal("Gagal memulai transaksi database")
	}
	defer dbTx.Rollback(ctx)

	// Update status to VOIDED
	if err := s.txRepo.UpdateStatus(ctx, dbTx, txID, "VOIDED"); err != nil {
		return model.ErrInternal("Gagal mengubah status transaksi")
	}

	// Restore stock via compensating ledger mutation VOID
	for _, it := range txRecord.Items {
		_, err := s.invRepo.MutateStock(
			ctx, dbTx, warehouseID, it.ProductID, it.Quantity,
			"VOID", "TRANSACTION", txRecord.InvoiceNumber, "Kompensasi pembatalan: "+reason, &actorID,
		)
		if err != nil {
			return model.ErrInternal("Gagal mengembalikan stok kompensasi void")
		}
	}

	if err := dbTx.Commit(ctx); err != nil {
		return model.ErrInternal("Gagal commit transaksi void")
	}

	// Audit trail
	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		StoreID:  &txRecord.StoreID,
		ActorID:  &actorID,
		Action:   "TRANSACTION_VOID",
		Entity:   "sales_transactions",
		EntityID: txID,
		Details:  reason,
	})

	return nil
}

// RefundTransaction compensates a transaction with refund status
func (s *POSService) RefundTransaction(ctx context.Context, txID, reason, actorID string, warehouseID string) error {
	if reason == "" {
		return model.ErrValidation("Alasan refund wajib diisi")
	}

	txRecord, err := s.txRepo.GetByID(ctx, txID)
	if err != nil || txRecord == nil {
		return model.ErrNotFound("Transaksi penjualan tidak ditemukan")
	}

	if txRecord.Status != "COMPLETED" {
		return model.ErrValidation(fmt.Sprintf("Transaksi tidak dapat di-refund karena berstatus '%s'", txRecord.Status))
	}

	pool := s.invRepo.GetPool()
	dbTx, err := pool.Begin(ctx)
	if err != nil {
		return model.ErrInternal("Gagal memulai transaksi database")
	}
	defer dbTx.Rollback(ctx)

	if err := s.txRepo.UpdateStatus(ctx, dbTx, txID, "REFUNDED"); err != nil {
		return model.ErrInternal("Gagal mengubah status transaksi ke refund")
	}

	for _, it := range txRecord.Items {
		_, err := s.invRepo.MutateStock(
			ctx, dbTx, warehouseID, it.ProductID, it.Quantity,
			"REFUND", "TRANSACTION", txRecord.InvoiceNumber, "Kompensasi refund: "+reason, &actorID,
		)
		if err != nil {
			return model.ErrInternal("Gagal mengembalikan stok kompensasi refund")
		}
	}

	if err := dbTx.Commit(ctx); err != nil {
		return model.ErrInternal("Gagal commit transaksi refund")
	}

	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		StoreID:  &txRecord.StoreID,
		ActorID:  &actorID,
		Action:   "TRANSACTION_REFUND",
		Entity:   "sales_transactions",
		EntityID: txID,
		Details:  reason,
	})

	return nil
}

func (s *POSService) ListTransactions(ctx context.Context, storeID string, limit, offset int) ([]model.SalesTransaction, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.txRepo.List(ctx, storeID, limit, offset)
}

func (s *POSService) GetTransaction(ctx context.Context, id string) (*model.SalesTransaction, error) {
	tx, err := s.txRepo.GetByID(ctx, id)
	if err != nil || tx == nil {
		return nil, model.ErrNotFound("Transaksi tidak ditemukan")
	}
	return tx, nil
}
