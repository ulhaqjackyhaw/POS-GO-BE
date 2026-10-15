package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"go-backend/internal/config"
	"go-backend/internal/model"
	"go-backend/internal/platform/database"
	"go-backend/internal/repository"
	"go-backend/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, *config.Config) {
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Skipf("Skipping integration test: PostgreSQL not available at %s (%v)", cfg.DatabaseURL, err)
		return nil, nil
	}

	// Apply migrations
	if err := database.RunMigrations(ctx, pool, "../migrations"); err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Seed initial data
	if err := database.SeedInitialData(ctx, pool); err != nil {
		t.Fatalf("Failed to seed initial data: %v", err)
	}

	return pool, cfg
}

func TestCompletePOSIntegration(t *testing.T) {
	pool, cfg := setupTestDB(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	ctx := context.Background()

	// Repositories
	userRepo := repository.NewUserRepository(pool)
	orgRepo := repository.NewOrgRepository(pool)
	catalogRepo := repository.NewCatalogRepository(pool)
	invRepo := repository.NewInventoryRepository(pool)
	shiftRepo := repository.NewShiftRepository(pool)
	txRepo := repository.NewTransactionRepository(pool)
	approvalRepo := repository.NewApprovalRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)

	// Services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpirationHours)
	catalogService := service.NewCatalogService(catalogRepo, orgRepo)
	shiftService := service.NewShiftService(shiftRepo, auditRepo)
	posService := service.NewPOSService(txRepo, invRepo, catalogRepo, shiftRepo, auditRepo, cfg.DefaultTaxRateBps)
	approvalService := service.NewApprovalService(approvalRepo, auditRepo)
	offlineSyncService := service.NewOfflineSyncService(posService, txRepo, auditRepo)

	// 1. Test Auth: Login as cashier
	cashierLogin, err := authService.Login(ctx, model.LoginRequest{
		Username: "kasir1",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Cashier login failed: %v", err)
	}
	if cashierLogin.AccessToken == "" {
		t.Fatal("Expected access token for cashier")
	}
	cashierID := cashierLogin.User.ID

	// Login as manager
	managerLogin, err := authService.Login(ctx, model.LoginRequest{
		Username: "manager1",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Manager login failed: %v", err)
	}
	managerID := managerLogin.User.ID

	// 2. Fetch Store, Warehouse, Terminal
	stores, err := orgRepo.ListStores(ctx)
	if err != nil || len(stores) == 0 {
		t.Fatalf("No stores found: %v", err)
	}
	storeID := stores[0].ID

	warehouses, err := orgRepo.ListWarehouses(ctx, storeID)
	if err != nil || len(warehouses) == 0 {
		t.Fatalf("No warehouses found: %v", err)
	}
	warehouseID := warehouses[0].ID

	terminals, err := orgRepo.ListTerminals(ctx, storeID)
	if err != nil || len(terminals) == 0 {
		t.Fatalf("No terminals found: %v", err)
	}
	terminalID := terminals[0].ID

	// 3. Test Products & Branch Price Override Resolution
	products, err := catalogService.ListProducts(ctx, storeID, warehouseID, "Teh Melati")
	if err != nil || len(products) == 0 {
		t.Fatalf("Failed to fetch products: %v", err)
	}
	tehMelati := products[0]
	// Seeded override price for Teh Melati is 9000 vs base 8000
	if tehMelati.EffectivePrice != 9000 {
		t.Errorf("Expected branch effective price 9000, got %d", tehMelati.EffectivePrice)
	}
	if !tehMelati.IsOverride {
		t.Errorf("Expected IsOverride = true for branch override price")
	}

	// Ensure stock is fresh and sufficient for repeated test runs
	_, _ = pool.Exec(ctx, `UPDATE inventory_balances SET quantity = 100 WHERE warehouse_id = $1 AND product_id = $2`, warehouseID, tehMelati.ID)

	// 4. Test Cashier Shift
	// Close any previously open shift if leftover
	if prevShift, _ := shiftRepo.GetActiveShift(ctx, storeID, terminalID, cashierID); prevShift != nil {
		_, _ = shiftService.CloseShift(ctx, prevShift.ID, model.CloseShiftRequest{
			EndingCashActual: prevShift.StartingCash,
			Notes:            "Cleaning previous test shift",
		}, cashierID)
	}

	shift, err := shiftService.OpenShift(ctx, model.OpenShiftRequest{
		StoreID:      storeID,
		TerminalID:   terminalID,
		StartingCash: 100000, // Rp 100.000 modal awal
	}, cashierID)
	if err != nil {
		t.Fatalf("Failed to open shift: %v", err)
	}
	if shift.Status != "OPEN" {
		t.Fatalf("Expected shift status OPEN, got %s", shift.Status)
	}

	// Record cash in movement
	err = shiftService.CreateCashMovement(ctx, model.CreateCashMovementRequest{
		MovementType: "IN",
		Amount:       50000,
		Category:     "MODAL_TAMBAHAN",
		Reason:       "Uang receh kembalian",
	}, shift.ID, cashierID)
	if err != nil {
		t.Fatalf("Failed to record cash movement: %v", err)
	}

	// 5. Test Quote Generation
	quoteResp, err := posService.GenerateQuote(ctx, model.QuoteRequest{
		StoreID: storeID,
		Items: []model.QuoteItemRequest{
			{ProductID: tehMelati.ID, Quantity: 2, DiscountAmount: 1000},
		},
	}, warehouseID)
	if err != nil {
		t.Fatalf("GenerateQuote failed: %v", err)
	}
	// Subtotal = 9000 * 2 = 18000
	// Total = 18000 - 1000 = 17000
	if quoteResp.Subtotal != 18000 {
		t.Errorf("Expected quote subtotal 18000, got %d", quoteResp.Subtotal)
	}
	if quoteResp.TotalAmount != 17000 {
		t.Errorf("Expected quote total 17000, got %d", quoteResp.TotalAmount)
	}

	// 6. Test Online Checkout
	idempotencyKey := fmt.Sprintf("IDEM-TEST-%d", time.Now().UnixNano())
	checkoutReq := model.CheckoutRequest{
		IdempotencyKey: idempotencyKey,
		StoreID:        storeID,
		TerminalID:     &terminalID,
		WarehouseID:    warehouseID,
		CustomerName:   "Pelanggan Budi",
		PaymentMethod:  "CASH",
		PaidAmount:     20000,
		Items: []model.CheckoutItemRequest{
			{ProductID: tehMelati.ID, Quantity: 2, DiscountAmount: 1000},
		},
	}

	checkoutResp, err := posService.CheckoutOnline(ctx, checkoutReq, cashierID)
	if err != nil {
		t.Fatalf("CheckoutOnline failed: %v", err)
	}
	if checkoutResp.TotalAmount != 17000 {
		t.Errorf("Expected total 17000, got %d", checkoutResp.TotalAmount)
	}
	if checkoutResp.ChangeAmount != 3000 { // 20000 - 17000 = 3000
		t.Errorf("Expected change 3000, got %d", checkoutResp.ChangeAmount)
	}
	txID := checkoutResp.Transaction.ID

	// 7. Test Idempotency: Retry with exact same key and payload
	retryResp, err := posService.CheckoutOnline(ctx, checkoutReq, cashierID)
	if err != nil {
		t.Fatalf("Idempotent retry failed: %v", err)
	}
	if retryResp.InvoiceNumber != checkoutResp.InvoiceNumber {
		t.Errorf("Idempotent retry must return identical invoice number, got %s vs %s", retryResp.InvoiceNumber, checkoutResp.InvoiceNumber)
	}

	// Test Idempotency Conflict: Same key with different payload
	conflictingReq := checkoutReq
	conflictingReq.PaidAmount = 50000 // changed payload
	_, err = posService.CheckoutOnline(ctx, conflictingReq, cashierID)
	if err == nil {
		t.Fatal("Expected IDEMPOTENCY_CONFLICT error when payload changes with same key")
	}
	appErr, ok := err.(*model.AppError)
	if !ok || appErr.Code != "IDEMPOTENCY_CONFLICT" {
		t.Errorf("Expected error code IDEMPOTENCY_CONFLICT, got %v", err)
	}

	// 8. Test Insufficient Stock rejection
	insufficientReq := checkoutReq
	insufficientReq.IdempotencyKey = fmt.Sprintf("IDEM-FAIL-%d", time.Now().UnixNano())
	insufficientReq.Items = []model.CheckoutItemRequest{
		{ProductID: tehMelati.ID, Quantity: 999999, DiscountAmount: 0},
	}
	_, err = posService.CheckoutOnline(ctx, insufficientReq, cashierID)
	if err == nil {
		t.Fatal("Expected INSUFFICIENT_STOCK error for excessive quantity")
	}
	appErr, ok = err.(*model.AppError)
	if !ok || appErr.Code != "INSUFFICIENT_STOCK" {
		t.Errorf("Expected error code INSUFFICIENT_STOCK, got %v", err)
	}

	// 9. Test Concurrency on Stock (deadlock-free and no negative stock)
	t.Run("Concurrency Stock Test", func(t *testing.T) {
		concurrencyCount := 10
		var wg sync.WaitGroup
		successCount := 0
		var mu sync.Mutex

		for i := 0; i < concurrencyCount; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				cReq := model.CheckoutRequest{
					IdempotencyKey: fmt.Sprintf("IDEM-CONC-%d-%d", time.Now().UnixNano(), index),
					StoreID:        storeID,
					TerminalID:     &terminalID,
					WarehouseID:    warehouseID,
					CustomerName:   fmt.Sprintf("Concurrent User %d", index),
					PaymentMethod:  "CASH",
					PaidAmount:     20000,
					Items: []model.CheckoutItemRequest{
						{ProductID: tehMelati.ID, Quantity: 1, DiscountAmount: 0},
					},
				}
				_, cErr := posService.CheckoutOnline(context.Background(), cReq, cashierID)
				if cErr == nil {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}(i)
		}
		wg.Wait()

		// Verify stock balance is >= 0
		finalStock, _ := invRepo.GetBalance(ctx, warehouseID, tehMelati.ID)
		if finalStock < 0 {
			t.Errorf("Negative stock detected! Current stock: %d", finalStock)
		}
	})

	// 10. Test Void Transaction
	err = posService.VoidTransaction(ctx, txID, "Salah input kasir", cashierID, warehouseID)
	if err != nil {
		t.Fatalf("VoidTransaction failed: %v", err)
	}
	voidedTx, _ := posService.GetTransaction(ctx, txID)
	if voidedTx.Status != "VOIDED" {
		t.Errorf("Expected transaction status VOIDED, got %s", voidedTx.Status)
	}

	// 11. Test Approval Workflow: Self-approval prevention
	approvalReq, err := approvalService.CreateRequest(ctx, model.CreateApprovalRequest{
		StoreID:    &storeID,
		ActionType: "SPECIAL_DISCOUNT",
		Payload:    `{"discount_percent": 50}`,
		Reason:     "Diskon keluarga staf",
	}, cashierID)
	if err != nil {
		t.Fatalf("Create approval request failed: %v", err)
	}

	// Cashier attempts to approve their own request -> MUST be rejected with CANNOT_APPROVE_OWN_REQUEST
	err = approvalService.ReviewRequest(ctx, approvalReq.ID, model.ReviewApprovalRequest{
		Status: "APPROVED",
	}, cashierID)
	if err == nil {
		t.Fatal("Self-approval must be forbidden")
	}
	appErr, ok = err.(*model.AppError)
	if !ok || appErr.Code != "CANNOT_APPROVE_OWN_REQUEST" {
		t.Errorf("Expected code CANNOT_APPROVE_OWN_REQUEST, got %v", err)
	}

	// Manager approves request -> succeeds
	err = approvalService.ReviewRequest(ctx, approvalReq.ID, model.ReviewApprovalRequest{
		Status: "APPROVED",
	}, managerID)
	if err != nil {
		t.Fatalf("Manager review failed: %v", err)
	}

	// 12. Test Offline Sync Batch
	syncResp, err := offlineSyncService.ProcessBatch(ctx, model.SyncBatchRequest{
		TerminalID: terminalID,
		Transactions: []model.SyncTransactionItem{
			{
				OfflineTxID:    "OFFLINE-001",
				IdempotencyKey: fmt.Sprintf("IDEM-OFFLINE-%d-1", time.Now().UnixNano()),
				StoreID:        storeID,
				TerminalID:     &terminalID,
				WarehouseID:    warehouseID,
				CustomerName:   "Offline Customer 1",
				PaymentMethod:  "CASH",
				PaidAmount:     20000,
				Items: []model.CheckoutItemRequest{
					{ProductID: tehMelati.ID, Quantity: 1, DiscountAmount: 0},
				},
			},
			{
				OfflineTxID:    "OFFLINE-002",
				IdempotencyKey: fmt.Sprintf("IDEM-OFFLINE-%d-2", time.Now().UnixNano()),
				StoreID:        storeID,
				TerminalID:     &terminalID,
				WarehouseID:    warehouseID,
				CustomerName:   "Offline Customer 2 Non-Cash",
				PaymentMethod:  "QRIS", // Non-cash should conflict in offline mode
				PaidAmount:     20000,
				Items: []model.CheckoutItemRequest{
					{ProductID: tehMelati.ID, Quantity: 1, DiscountAmount: 0},
				},
			},
		},
	}, cashierID)
	if err != nil {
		t.Fatalf("Offline batch sync failed: %v", err)
	}
	if syncResp.SyncedCount != 1 || syncResp.ConflictCount != 1 {
		t.Errorf("Expected 1 synced and 1 conflict, got %d synced, %d conflict", syncResp.SyncedCount, syncResp.ConflictCount)
	}

	// 13. Close Shift & Check Cash Reconciliation
	closedShift, err := shiftService.CloseShift(ctx, shift.ID, model.CloseShiftRequest{
		EndingCashActual: 170000,
		Notes:            "Kas penutupan shift",
	}, cashierID)
	if err != nil {
		t.Fatalf("CloseShift failed: %v", err)
	}
	if closedShift.Status != "CLOSED" {
		t.Errorf("Expected status CLOSED, got %s", closedShift.Status)
	}
	if closedShift.EndingCashExpected == nil || closedShift.CashDifference == nil {
		t.Fatal("Expected cash reconciliation metrics to be computed")
	}
}
