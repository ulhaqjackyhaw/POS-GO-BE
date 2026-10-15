package model

import (
	"strings"
	"time"
)

// Auth DTOs
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken string   `json:"access_token"`
	TokenType   string   `json:"token_type"`
	ExpiresIn   int      `json:"expires_in"`
	User        UserInfo `json:"user"`
}

type UserSummary struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

type UserInfo struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	FullName    string   `json:"full_name"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	StoreIDs    []string `json:"store_ids"`
}

// Catalog DTOs
type CreateProductRequest struct {
	SKU        string  `json:"sku"`
	Barcode    string  `json:"barcode"`
	Name       string  `json:"name"`
	CategoryID *string `json:"category_id"`
	Unit       string  `json:"unit"`
	BasePrice  int64   `json:"base_price"`
}

type SetBranchPriceRequest struct {
	Price      int64 `json:"price"`
	PriceMinor int64 `json:"price_minor,omitempty"`
}

func (r *SetBranchPriceRequest) GetEffectivePrice() int64 {
	if r.Price <= 0 && r.PriceMinor > 0 {
		return r.PriceMinor
	}
	return r.Price
}

// Inventory DTOs
type StockAdjustmentRequest struct {
	WarehouseID   string `json:"warehouse_id"`
	ProductID     string `json:"product_id"`
	Quantity      int64  `json:"quantity"` // delta change (+ or -)
	QuantityDelta int64  `json:"quantity_delta,omitempty"`
	Reason        string `json:"reason"`
}

func (r *StockAdjustmentRequest) GetEffectiveQuantity() int64 {
	if r.Quantity == 0 && r.QuantityDelta != 0 {
		return r.QuantityDelta
	}
	return r.Quantity
}

type CurrentStockItem struct {
	WarehouseID   string    `json:"warehouse_id"`
	WarehouseName string    `json:"warehouse_name"`
	ProductID     string    `json:"product_id"`
	ProductName   string    `json:"product_name"`
	SKU           string    `json:"sku"`
	Quantity      int64     `json:"quantity"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Shift DTOs
type OpenShiftRequest struct {
	StoreID          string `json:"store_id"`
	TerminalID       string `json:"terminal_id"`
	StartingCash     int64  `json:"starting_cash"`
	OpeningCashMinor int64  `json:"opening_cash_minor,omitempty"`
	Note             string `json:"note,omitempty"`
}

func (r *OpenShiftRequest) GetStartingCash() int64 {
	if r.StartingCash <= 0 && r.OpeningCashMinor > 0 {
		return r.OpeningCashMinor
	}
	return r.StartingCash
}

type CloseShiftRequest struct {
	EndingCashActual  int64  `json:"ending_cash_actual"`
	ActualCashMinor   int64  `json:"actual_cash_minor,omitempty"`
	DiscrepancyReason string `json:"discrepancy_reason,omitempty"`
	Notes             string `json:"notes"`
}

func (r *CloseShiftRequest) GetActualCash() int64 {
	if r.EndingCashActual <= 0 && r.ActualCashMinor > 0 {
		return r.ActualCashMinor
	}
	return r.EndingCashActual
}

func (r *CloseShiftRequest) GetNotes() string {
	if r.Notes == "" && r.DiscrepancyReason != "" {
		return r.DiscrepancyReason
	}
	return r.Notes
}

type CreateCashMovementRequest struct {
	MovementType string `json:"movement_type"`  // IN, OUT
	Type         string `json:"type,omitempty"` // cash_in, cash_out
	Amount       int64  `json:"amount"`
	AmountMinor  int64  `json:"amount_minor,omitempty"`
	Category     string `json:"category"`
	Reason       string `json:"reason"`
}

func (r *CreateCashMovementRequest) Normalize() {
	if r.MovementType == "" && r.Type != "" {
		lower := strings.ToLower(r.Type)
		if lower == "cash_in" || lower == "in" {
			r.MovementType = "IN"
		} else if lower == "cash_out" || lower == "out" {
			r.MovementType = "OUT"
		}
	} else {
		r.MovementType = strings.ToUpper(r.MovementType)
	}
	if r.Amount <= 0 && r.AmountMinor > 0 {
		r.Amount = r.AmountMinor
	}
}

type ShiftDetailResponse struct {
	Shift        CashierShift   `json:"shift"`
	ExpectedCash int64          `json:"expected_cash"`
	CashSales    int64          `json:"cash_sales"`
	CashIn       int64          `json:"cash_in"`
	CashOut      int64          `json:"cash_out"`
	Movements    []CashMovement `json:"movements"`
}

// POS Quote & Checkout DTOs
type QuoteItemRequest struct {
	ProductID      string `json:"product_id"`
	Quantity       int64  `json:"quantity"`
	DiscountAmount int64  `json:"discount_amount"` // per line
}

type QuoteRequest struct {
	StoreID string             `json:"store_id"`
	Items   []QuoteItemRequest `json:"items"`
}

type QuoteItemResponse struct {
	ProductID      string `json:"product_id"`
	ProductName    string `json:"product_name"`
	UnitPrice      int64  `json:"unit_price"` // Tax-inclusive unit price
	UnitPriceMinor int64  `json:"unit_price_minor"`
	Quantity       int64  `json:"quantity"`
	Subtotal       int64  `json:"subtotal"`
	DiscountAmount int64  `json:"discount_amount"`
	DiscountMinor  int64  `json:"discount_minor"`
	TaxAmount      int64  `json:"tax_amount"` // Tax component extracted: total * r / (1 + r)
	TaxMinor       int64  `json:"tax_minor"`
	Total          int64  `json:"total"`
	LineTotalMinor int64  `json:"line_total_minor"`
	AvailableStock int64  `json:"available_stock"`
}

type QuoteResponse struct {
	StoreID        string              `json:"store_id"`
	Subtotal       int64               `json:"subtotal"`
	SubtotalMinor  int64               `json:"subtotal_minor"`
	DiscountAmount int64               `json:"discount_amount"`
	DiscountMinor  int64               `json:"discount_minor"`
	TaxAmount      int64               `json:"tax_amount"`
	TaxMinor       int64               `json:"tax_minor"`
	TotalAmount    int64               `json:"total_amount"`
	TotalMinor     int64               `json:"total_minor"`
	Items          []QuoteItemResponse `json:"items"`
	Lines          []QuoteItemResponse `json:"lines"`
}

type CheckoutItemRequest struct {
	ProductID      string `json:"product_id"`
	Quantity       int64  `json:"quantity"`
	DiscountAmount int64  `json:"discount_amount"`
}

type PaymentDetail struct {
	Method        string `json:"method"`
	ReceivedMinor int64  `json:"received_minor"`
}

type CheckoutRequest struct {
	IdempotencyKey string                `json:"idempotency_key"`
	StoreID        string                `json:"store_id"`
	TerminalID     *string               `json:"terminal_id"`
	WarehouseID    string                `json:"warehouse_id"`
	CustomerName   string                `json:"customer_name"`
	PaymentMethod  string                `json:"payment_method"` // CASH
	PaidAmount     int64                 `json:"paid_amount"`
	Payment        *PaymentDetail        `json:"payment,omitempty"`
	Items          []CheckoutItemRequest `json:"items"`
}

func (r *CheckoutRequest) Normalize() {
	if r.Payment != nil {
		if r.PaymentMethod == "" && r.Payment.Method != "" {
			r.PaymentMethod = strings.ToUpper(r.Payment.Method)
		}
		if r.PaidAmount <= 0 && r.Payment.ReceivedMinor > 0 {
			r.PaidAmount = r.Payment.ReceivedMinor
		}
	}
	if r.PaymentMethod == "" {
		r.PaymentMethod = "CASH"
	} else {
		r.PaymentMethod = strings.ToUpper(r.PaymentMethod)
	}
}

type CheckoutResponse struct {
	TransactionID string            `json:"transaction_id,omitempty"`
	InvoiceNumber string            `json:"invoice_number"`
	ReceiptNumber string            `json:"receipt_number,omitempty"`
	Status        string            `json:"status"`
	Subtotal      int64             `json:"subtotal"`
	DiscountTotal int64             `json:"discount_total"`
	TaxAmount     int64             `json:"tax_amount"`
	TotalAmount   int64             `json:"total_amount"`
	TotalMinor    int64             `json:"total_minor,omitempty"`
	PaidAmount    int64             `json:"paid_amount"`
	ChangeAmount  int64             `json:"change_amount"`
	ChangeMinor   int64             `json:"change_minor,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	Transaction   *SalesTransaction `json:"transaction,omitempty"`
}

type VoidRefundRequest struct {
	Reason string `json:"reason"`
}

// Approval DTOs
type CreateApprovalRequest struct {
	StoreID    *string `json:"store_id"`
	ActionType string  `json:"action_type"` // STOCK_ADJUSTMENT, VOID_TRANSACTION, REFUND, SPECIAL_DISCOUNT
	Payload    string  `json:"payload"`     // JSON string of parameters
	Reason     string  `json:"reason"`
}

type ReviewApprovalRequest struct {
	Status          string  `json:"status"` // APPROVED, REJECTED
	RejectionReason *string `json:"rejection_reason,omitempty"`
}

// Offline Sync DTOs
type SyncTransactionItem struct {
	OfflineTxID    string                `json:"offline_tx_id"`
	IdempotencyKey string                `json:"idempotency_key"`
	StoreID        string                `json:"store_id"`
	TerminalID     *string               `json:"terminal_id"`
	WarehouseID    string                `json:"warehouse_id"`
	CustomerName   string                `json:"customer_name"`
	PaymentMethod  string                `json:"payment_method"` // CASH only
	PaidAmount     int64                 `json:"paid_amount"`
	OfflineTime    time.Time             `json:"offline_time"`
	Items          []CheckoutItemRequest `json:"items"`
}

type SyncBatchRequest struct {
	TerminalID   string                `json:"terminal_id"`
	Transactions []SyncTransactionItem `json:"transactions"`
}

type SyncResultItem struct {
	OfflineTxID   string `json:"offline_tx_id"`
	InvoiceNumber string `json:"invoice_number,omitempty"`
	Status        string `json:"status"` // SYNCED, CONFLICT, DUPLICATE
	ErrorCode     string `json:"error_code,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

type SyncBatchResponse struct {
	TotalProcessed int              `json:"total_processed"`
	SyncedCount    int              `json:"synced_count"`
	ConflictCount  int              `json:"conflict_count"`
	Results        []SyncResultItem `json:"results"`
}

// Report DTOs
type SalesSummaryReport struct {
	TotalTransactions int64 `json:"total_transactions"`
	GrossSales        int64 `json:"gross_sales"`
	TotalDiscount     int64 `json:"total_discount"`
	TotalTax          int64 `json:"total_tax"`
	NetSales          int64 `json:"net_sales"`
}

type TopProductItem struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int64  `json:"quantity"`
	TotalSales  int64  `json:"total_sales"`
}
