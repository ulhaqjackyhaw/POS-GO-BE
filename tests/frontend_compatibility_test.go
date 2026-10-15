package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go-backend/internal/model"
	"go-backend/internal/repository"
	"go-backend/internal/service"
)

func TestFrontendCompatibilityEndpoints(t *testing.T) {
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
	invService := service.NewInventoryService(invRepo, auditRepo)
	shiftService := service.NewShiftService(shiftRepo, auditRepo)
	posService := service.NewPOSService(txRepo, invRepo, catalogRepo, shiftRepo, auditRepo, cfg.DefaultTaxRateBps)
	approvalService := service.NewApprovalService(approvalRepo, auditRepo)

	// Login kasir
	cashierLogin, err := authService.Login(ctx, model.LoginRequest{Username: "kasir1", Password: "password123"})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	cashierID := cashierLogin.User.ID

	// 1. Test GET /api/v1/categories
	categories, err := catalogService.ListCategories(ctx)
	if err != nil || len(categories) == 0 {
		t.Fatalf("ListCategories failed: %v", err)
	}
	if categories[0].Name == "" {
		t.Errorf("Expected category name, got empty")
	}

	// 2. Test GET /api/v1/inventory/stock
	stores, _ := orgRepo.ListStores(ctx)
	storeID := stores[0].ID
	warehouses, _ := orgRepo.ListWarehouses(ctx, storeID)
	warehouseID := warehouses[0].ID

	stockList, err := invService.ListStock(ctx, warehouseID)
	if err != nil || len(stockList) == 0 {
		t.Fatalf("ListStock failed: %v", err)
	}
	if stockList[0].ProductName == "" || stockList[0].Quantity < 0 {
		t.Errorf("Invalid stock item: %+v", stockList[0])
	}

	// 3. Test Dual-Field Open Shift (using opening_cash_minor)
	terminals, _ := orgRepo.ListTerminals(ctx, storeID)
	terminalID := terminals[0].ID

	// Close previous shift if any
	if prev, _ := shiftRepo.GetActiveShift(ctx, storeID, terminalID, cashierID); prev != nil {
		_, _ = shiftService.CloseShift(ctx, prev.ID, model.CloseShiftRequest{
			ActualCashMinor:   prev.StartingCash,
			DiscrepancyReason: "Reset",
		}, cashierID)
	}

	openReq := model.OpenShiftRequest{
		StoreID:          storeID,
		TerminalID:       terminalID,
		OpeningCashMinor: 150000, // Frontend PRD format
	}
	shift, err := shiftService.OpenShift(ctx, openReq, cashierID)
	if err != nil {
		t.Fatalf("OpenShift with opening_cash_minor failed: %v", err)
	}
	if shift.StartingCash != 150000 {
		t.Errorf("Expected starting cash 150000, got %d", shift.StartingCash)
	}

	// 4. Test Cash Movement with Frontend PRD format: type="cash_in", amount_minor=25000
	movementReq := model.CreateCashMovementRequest{
		Type:        "cash_in",
		AmountMinor: 25000,
		Category:    "PETTY_CASH",
		Reason:      "Tambahan kas kecil",
	}
	err = shiftService.CreateCashMovement(ctx, movementReq, shift.ID, cashierID)
	if err != nil {
		t.Fatalf("CreateCashMovement with type/amount_minor failed: %v", err)
	}

	// 5. Test GET /api/v1/shifts/:id (Shift Detail with Expected Cash)
	shiftDetail, err := shiftService.GetShiftDetail(ctx, shift.ID)
	if err != nil {
		t.Fatalf("GetShiftDetail failed: %v", err)
	}
	if shiftDetail.Shift.ID != shift.ID {
		t.Errorf("Expected shift ID %s, got %s", shift.ID, shiftDetail.Shift.ID)
	}
	if shiftDetail.CashIn != 25000 {
		t.Errorf("Expected CashIn 25000, got %d", shiftDetail.CashIn)
	}
	// ExpectedCash = 150000 + 25000 = 175000
	if shiftDetail.ExpectedCash != 175000 {
		t.Errorf("Expected ExpectedCash 175000, got %d", shiftDetail.ExpectedCash)
	}
	if len(shiftDetail.Movements) != 1 {
		t.Errorf("Expected 1 movement, got %d", len(shiftDetail.Movements))
	}

	// 6. Test Checkout with Frontend PRD nested Payment format: payment: { method: "cash", received_minor: 200000 }
	products, _ := catalogService.ListProducts(ctx, storeID, warehouseID, "")
	checkoutReq := model.CheckoutRequest{
		IdempotencyKey: fmt.Sprintf("IDEM-FRONTEND-%d", time.Now().UnixNano()),
		StoreID:        storeID,
		TerminalID:     &terminalID,
		WarehouseID:    warehouseID,
		Payment: &model.PaymentDetail{
			Method:        "cash",
			ReceivedMinor: 200000,
		},
		Items: []model.CheckoutItemRequest{
			{ProductID: products[0].ID, Quantity: 1, DiscountAmount: 0},
		},
	}
	checkoutResp, err := posService.CheckoutOnline(ctx, checkoutReq, cashierID)
	if err != nil {
		t.Fatalf("Checkout with nested payment failed: %v", err)
	}
	if checkoutResp.PaidAmount != 200000 {
		t.Errorf("Expected paid amount 200000, got %d", checkoutResp.PaidAmount)
	}

	// 7. Test GET /api/v1/approvals/:id
	appReq, err := approvalService.CreateRequest(ctx, model.CreateApprovalRequest{
		StoreID:    &storeID,
		ActionType: "VOID_TRANSACTION",
		Payload:    `{"transaction_id": "test"}`,
		Reason:     "Pelanggan membatalkan pesanan",
	}, cashierID)
	if err != nil {
		t.Fatalf("Create approval failed: %v", err)
	}

	retrievedApproval, err := approvalService.GetByID(ctx, appReq.ID)
	if err != nil {
		t.Fatalf("GetByID approval failed: %v", err)
	}
	if retrievedApproval.ID != appReq.ID {
		t.Errorf("Expected approval ID %s, got %s", appReq.ID, retrievedApproval.ID)
	}

	// 8. Test Close Shift with Frontend PRD format (actual_cash_minor, discrepancy_reason)
	closedShift, err := shiftService.CloseShift(ctx, shift.ID, model.CloseShiftRequest{
		ActualCashMinor:   193000,
		DiscrepancyReason: "Sesuai hitungan fisik kasir",
	}, cashierID)
	if err != nil {
		t.Fatalf("CloseShift with actual_cash_minor failed: %v", err)
	}
	if *closedShift.EndingCashActual != 193000 {
		t.Errorf("Expected actual cash 193000, got %d", *closedShift.EndingCashActual)
	}
	if closedShift.Notes != "Sesuai hitungan fisik kasir" {
		t.Errorf("Expected notes to take discrepancy_reason, got '%s'", closedShift.Notes)
	}
}
