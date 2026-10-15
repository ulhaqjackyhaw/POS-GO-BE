package repository

import (
	"context"
	"errors"

	"go-backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepository struct {
	pool *pgxpool.Pool
}

func NewTransactionRepository(pool *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{pool: pool}
}

func (r *TransactionRepository) GetPool() *pgxpool.Pool {
	return r.pool
}

func (r *TransactionRepository) GetByIdempotencyKey(ctx context.Context, key string) (*model.SalesTransaction, error) {
	query := `
		SELECT id, invoice_number, idempotency_key, store_id, terminal_id, shift_id, cashier_id,
		       COALESCE(customer_name, ''), subtotal, discount_amount, tax_amount, total_amount,
		       payment_method, paid_amount, change_amount, status, payload_hash, offline_created_at,
		       created_at, updated_at
		FROM sales_transactions
		WHERE idempotency_key = $1
	`
	var t model.SalesTransaction
	err := r.pool.QueryRow(ctx, query, key).Scan(
		&t.ID, &t.InvoiceNumber, &t.IdempotencyKey, &t.StoreID, &t.TerminalID, &t.ShiftID, &t.CashierID,
		&t.CustomerName, &t.Subtotal, &t.DiscountAmount, &t.TaxAmount, &t.TotalAmount,
		&t.PaymentMethod, &t.PaidAmount, &t.ChangeAmount, &t.Status, &t.PayloadHash, &t.OfflineCreatedAt,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Fetch items
	items, err := r.GetTransactionItems(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	t.Items = items

	return &t, nil
}

func (r *TransactionRepository) GetByID(ctx context.Context, id string) (*model.SalesTransaction, error) {
	if id == "" {
		return nil, nil
	}
	query := `
		SELECT id, invoice_number, idempotency_key, store_id, terminal_id, shift_id, cashier_id,
		       COALESCE(customer_name, ''), subtotal, discount_amount, tax_amount, total_amount,
		       payment_method, paid_amount, change_amount, status, payload_hash, offline_created_at,
		       created_at, updated_at
		FROM sales_transactions
		WHERE id::text = $1
	`
	var t model.SalesTransaction
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.InvoiceNumber, &t.IdempotencyKey, &t.StoreID, &t.TerminalID, &t.ShiftID, &t.CashierID,
		&t.CustomerName, &t.Subtotal, &t.DiscountAmount, &t.TaxAmount, &t.TotalAmount,
		&t.PaymentMethod, &t.PaidAmount, &t.ChangeAmount, &t.Status, &t.PayloadHash, &t.OfflineCreatedAt,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	items, err := r.GetTransactionItems(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	t.Items = items

	return &t, nil
}

func (r *TransactionRepository) GetTransactionItems(ctx context.Context, txID string) ([]model.SalesTransactionItem, error) {
	if txID == "" {
		return nil, nil
	}
	query := `
		SELECT id, transaction_id, product_id, product_name, unit_price, quantity, subtotal, discount_amount, tax_amount, total, created_at
		FROM sales_transaction_items
		WHERE transaction_id::text = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, txID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.SalesTransactionItem
	for rows.Next() {
		var it model.SalesTransactionItem
		err := rows.Scan(
			&it.ID, &it.TransactionID, &it.ProductID, &it.ProductName, &it.UnitPrice,
			&it.Quantity, &it.Subtotal, &it.DiscountAmount, &it.TaxAmount, &it.Total, &it.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *TransactionRepository) CreateTransaction(ctx context.Context, tx pgx.Tx, t *model.SalesTransaction) error {
	query := `
		INSERT INTO sales_transactions (
			invoice_number, idempotency_key, store_id, terminal_id, shift_id, cashier_id,
			customer_name, subtotal, discount_amount, tax_amount, total_amount, payment_method,
			paid_amount, change_amount, status, payload_hash, offline_created_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	err := tx.QueryRow(ctx, query,
		t.InvoiceNumber, t.IdempotencyKey, t.StoreID, t.TerminalID, t.ShiftID, t.CashierID,
		t.CustomerName, t.Subtotal, t.DiscountAmount, t.TaxAmount, t.TotalAmount, t.PaymentMethod,
		t.PaidAmount, t.ChangeAmount, t.Status, t.PayloadHash, t.OfflineCreatedAt,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return err
	}

	for i := range t.Items {
		item := &t.Items[i]
		itemQuery := `
			INSERT INTO sales_transaction_items (
				transaction_id, product_id, product_name, unit_price, quantity,
				subtotal, discount_amount, tax_amount, total, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
			RETURNING id, created_at
		`
		err = tx.QueryRow(ctx, itemQuery,
			t.ID, item.ProductID, item.ProductName, item.UnitPrice, item.Quantity,
			item.Subtotal, item.DiscountAmount, item.TaxAmount, item.Total,
		).Scan(&item.ID, &item.CreatedAt)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *TransactionRepository) UpdateStatus(ctx context.Context, tx pgx.Tx, id, status string) error {
	query := `UPDATE sales_transactions SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := tx.Exec(ctx, query, status, id)
	return err
}

func (r *TransactionRepository) List(ctx context.Context, storeID string, limit, offset int) ([]model.SalesTransaction, error) {
	query := `
		SELECT id, invoice_number, idempotency_key, store_id, terminal_id, shift_id, cashier_id,
		       COALESCE(customer_name, ''), subtotal, discount_amount, tax_amount, total_amount,
		       payment_method, paid_amount, change_amount, status, payload_hash, offline_created_at,
		       created_at, updated_at
		FROM sales_transactions
		WHERE ($1 = '' OR store_id::text = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, storeID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.SalesTransaction
	for rows.Next() {
		var t model.SalesTransaction
		err := rows.Scan(
			&t.ID, &t.InvoiceNumber, &t.IdempotencyKey, &t.StoreID, &t.TerminalID, &t.ShiftID, &t.CashierID,
			&t.CustomerName, &t.Subtotal, &t.DiscountAmount, &t.TaxAmount, &t.TotalAmount,
			&t.PaymentMethod, &t.PaidAmount, &t.ChangeAmount, &t.Status, &t.PayloadHash, &t.OfflineCreatedAt,
			&t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, nil
}
