package model

import "time"

// Organization
type Store struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Warehouse struct {
	ID        string    `json:"id"`
	StoreID   string    `json:"store_id"`
	StoreName string    `json:"store_name,omitempty"`
	StoreCode string    `json:"store_code,omitempty"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Terminal struct {
	ID                string     `json:"id"`
	StoreID           string     `json:"store_id"`
	StoreName         string     `json:"store_name,omitempty"`
	StoreCode         string     `json:"store_code,omitempty"`
	Code              string     `json:"code"`
	Name              string     `json:"name"`
	IsActive          bool       `json:"is_active"`
	AssignedCashierID *string    `json:"assigned_cashier_id,omitempty"`
	CashierName       string     `json:"cashier_name,omitempty"`
	CashierUsername   string     `json:"cashier_username,omitempty"`
	HasActiveShift    bool       `json:"has_active_shift"`
	ActiveShiftID     string     `json:"active_shift_id,omitempty"`
	ActiveCashierID   string     `json:"active_cashier_id,omitempty"`
	ActiveCashierName string     `json:"active_cashier_name,omitempty"`
	ActiveCashierUser string     `json:"active_cashier_username,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// RBAC & User
type Role struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Permission struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Module      string `json:"module"`
}

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	RoleID       string    `json:"role_id"`
	RoleCode     string    `json:"role_code,omitempty"`
	RoleName     string    `json:"role_name,omitempty"`
	StoreIDs     []string  `json:"store_ids,omitempty"`
	Stores       []Store   `json:"stores,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Catalog
type Category struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Product struct {
	ID         string    `json:"id"`
	SKU        string    `json:"sku"`
	Barcode    string    `json:"barcode"`
	Name       string    `json:"name"`
	CategoryID *string   `json:"category_id"`
	Unit       string    `json:"unit"`
	BasePrice  int64     `json:"base_price"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ProductWithPrice struct {
	Product
	EffectivePrice int64  `json:"effective_price"`
	IsOverride     bool   `json:"is_override"`
	CurrentStock   int64  `json:"current_stock"`
	WarehouseID    string `json:"warehouse_id"`
}

type StoreProductPrice struct {
	ID        string    `json:"id"`
	StoreID   string    `json:"store_id"`
	ProductID string    `json:"product_id"`
	Price     int64     `json:"price"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Inventory
type InventoryBalance struct {
	ID          string    `json:"id"`
	WarehouseID string    `json:"warehouse_id"`
	ProductID   string    `json:"product_id"`
	Quantity    int64     `json:"quantity"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type InventoryLedger struct {
	ID             string    `json:"id"`
	WarehouseID    string    `json:"warehouse_id"`
	ProductID      string    `json:"product_id"`
	MutationType   string    `json:"mutation_type"` // RECEIVE, SALE, ADJUSTMENT, TRANSFER_OUT, TRANSFER_IN, VOID, REFUND
	ReferenceType  string    `json:"reference_type"`
	ReferenceID    string    `json:"reference_id"`
	QuantityChange int64     `json:"quantity_change"`
	BalanceAfter   int64     `json:"balance_after"`
	Reason         string    `json:"reason"`
	CreatedBy      *string   `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
}

// Cashier Shift & Cash Movement
type CashierShift struct {
	ID                 string     `json:"id"`
	StoreID            string     `json:"store_id"`
	TerminalID         string     `json:"terminal_id"`
	UserID             string     `json:"user_id"`
	StartingCash       int64      `json:"starting_cash"`
	EndingCashActual   *int64     `json:"ending_cash_actual"`
	EndingCashExpected *int64     `json:"ending_cash_expected"`
	CashDifference     *int64     `json:"cash_difference"`
	Status             string     `json:"status"` // OPEN, CLOSED
	OpenedAt           time.Time  `json:"opened_at"`
	ClosedAt           *time.Time `json:"closed_at"`
	Notes              string     `json:"notes"`
}

type CashMovement struct {
	ID           string    `json:"id"`
	ShiftID      string    `json:"shift_id"`
	MovementType string    `json:"movement_type"` // IN, OUT
	Amount       int64     `json:"amount"`
	Category     string    `json:"category"`
	Reason       string    `json:"reason"`
	CreatedBy    *string   `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
}

// Sales Transactions
type SalesTransaction struct {
	ID               string                 `json:"id"`
	InvoiceNumber    string                 `json:"invoice_number"`
	IdempotencyKey   string                 `json:"idempotency_key"`
	StoreID          string                 `json:"store_id"`
	TerminalID       *string                `json:"terminal_id"`
	ShiftID          *string                `json:"shift_id"`
	CashierID        string                 `json:"cashier_id"`
	CustomerName     string                 `json:"customer_name"`
	Subtotal         int64                  `json:"subtotal"`
	DiscountAmount   int64                  `json:"discount_amount"`
	TaxAmount        int64                  `json:"tax_amount"`
	TotalAmount      int64                  `json:"total_amount"`
	PaymentMethod    string                 `json:"payment_method"`
	PaidAmount       int64                  `json:"paid_amount"`
	ChangeAmount     int64                  `json:"change_amount"`
	Status           string                 `json:"status"` // COMPLETED, VOIDED, REFUNDED
	PayloadHash      string                 `json:"payload_hash"`
	OfflineCreatedAt *time.Time             `json:"offline_created_at,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	Items            []SalesTransactionItem `json:"items,omitempty"`
}

type SalesTransactionItem struct {
	ID             string    `json:"id"`
	TransactionID  string    `json:"transaction_id"`
	ProductID      string    `json:"product_id"`
	ProductName    string    `json:"product_name"`
	UnitPrice      int64     `json:"unit_price"`
	Quantity       int64     `json:"quantity"`
	Subtotal       int64     `json:"subtotal"`
	DiscountAmount int64     `json:"discount_amount"`
	TaxAmount      int64     `json:"tax_amount"`
	Total          int64     `json:"total"`
	CreatedAt      time.Time `json:"created_at"`
}

// Approval Request
type ApprovalRequest struct {
	ID              string    `json:"id"`
	StoreID         *string   `json:"store_id"`
	ActionType      string    `json:"action_type"` // STOCK_ADJUSTMENT, VOID_TRANSACTION, REFUND, SPECIAL_DISCOUNT
	Status          string    `json:"status"`      // PENDING, APPROVED, REJECTED
	RequestedBy     string    `json:"requested_by"`
	ApprovedBy      *string   `json:"approved_by"`
	Payload         string    `json:"payload"`
	Reason          string    `json:"reason"`
	RejectionReason *string   `json:"rejection_reason"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Stock Transfer
type StockTransfer struct {
	ID              string              `json:"id"`
	TransferNumber  string              `json:"transfer_number"`
	FromWarehouseID string              `json:"from_warehouse_id"`
	ToWarehouseID   string              `json:"to_warehouse_id"`
	Status          string              `json:"status"` // DRAFT, SHIPPED, RECEIVED, CANCELLED
	RequestedBy     string              `json:"requested_by"`
	Notes           string              `json:"notes"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	Items           []StockTransferItem `json:"items,omitempty"`
}

type StockTransferItem struct {
	ID               string `json:"id"`
	TransferID       string `json:"transfer_id"`
	ProductID        string `json:"product_id"`
	Quantity         int64  `json:"quantity"`
	ReceivedQuantity int64  `json:"received_quantity"`
}

// Audit Log
type AuditLog struct {
	ID        string    `json:"id"`
	StoreID   *string   `json:"store_id"`
	ActorID   *string   `json:"actor_id"`
	Action    string    `json:"action"`
	Entity    string    `json:"entity"`
	EntityID  string    `json:"entity_id"`
	Details   string    `json:"details"`
	IPAddress string    `json:"ip_address"`
	CreatedAt time.Time `json:"created_at"`
}
