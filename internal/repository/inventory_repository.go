package repository

import (
	"context"
	"errors"
	"sort"

	"go-backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InventoryRepository struct {
	pool *pgxpool.Pool
}

func NewInventoryRepository(pool *pgxpool.Pool) *InventoryRepository {
	return &InventoryRepository{pool: pool}
}

func (r *InventoryRepository) GetPool() *pgxpool.Pool {
	return r.pool
}

// GetBalance returns stock for a single product in warehouse
func (r *InventoryRepository) GetBalance(ctx context.Context, warehouseID, productID string) (int64, error) {
	if warehouseID == "" || productID == "" {
		return 0, nil
	}
	query := `SELECT COALESCE(quantity, 0) FROM inventory_balances WHERE warehouse_id::text = $1 AND product_id::text = $2`
	var qty int64
	err := r.pool.QueryRow(ctx, query, warehouseID, productID).Scan(&qty)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return qty, nil
}

// LockBalancesForUpdate locks multiple stock balance rows in deterministic order to prevent deadlocks
func (r *InventoryRepository) LockBalancesForUpdate(ctx context.Context, tx pgx.Tx, warehouseID string, productIDs []string) (map[string]int64, error) {
	// Deterministic sorting of product IDs as mandated by AGENT.md
	sortedIDs := make([]string, len(productIDs))
	copy(sortedIDs, productIDs)
	sort.Strings(sortedIDs)

	balances := make(map[string]int64)

	// Lock each row in deterministic sorted order
	for _, pid := range sortedIDs {
		// Ensure row exists or create with 0 if missing
		_, err := tx.Exec(ctx, `
			INSERT INTO inventory_balances (warehouse_id, product_id, quantity)
			VALUES ($1, $2, 0)
			ON CONFLICT (warehouse_id, product_id) DO NOTHING
		`, warehouseID, pid)
		if err != nil {
			return nil, err
		}

		var qty int64
		err = tx.QueryRow(ctx, `
			SELECT quantity
			FROM inventory_balances
			WHERE warehouse_id = $1 AND product_id = $2
			FOR UPDATE
		`, warehouseID, pid).Scan(&qty)
		if err != nil {
			return nil, err
		}
		balances[pid] = qty
	}

	return balances, nil
}

// MutateStock applies atomic change to inventory_balances and appends an immutable inventory_ledger entry
func (r *InventoryRepository) MutateStock(
	ctx context.Context,
	tx pgx.Tx,
	warehouseID, productID string,
	qtyDelta int64,
	mutationType, refType, refID, reason string,
	actorID *string,
) (int64, error) {
	var newBalance int64
	// Atomic update enforced by DB constraint chk_stock_non_negative
	err := tx.QueryRow(ctx, `
		UPDATE inventory_balances
		SET quantity = quantity + $1, updated_at = NOW()
		WHERE warehouse_id = $2 AND product_id = $3
		RETURNING quantity
	`, qtyDelta, warehouseID, productID).Scan(&newBalance)
	if err != nil {
		return 0, err
	}

	// Insert immutable ledger
	_, err = tx.Exec(ctx, `
		INSERT INTO inventory_ledgers (
			warehouse_id, product_id, mutation_type, reference_type, reference_id,
			quantity_change, balance_after, reason, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`, warehouseID, productID, mutationType, refType, refID, qtyDelta, newBalance, reason, actorID)
	if err != nil {
		return 0, err
	}

	return newBalance, nil
}

func (r *InventoryRepository) ListLedgers(ctx context.Context, warehouseID, productID string, limit, offset int) ([]model.InventoryLedger, error) {
	query := `
		SELECT id, warehouse_id, product_id, mutation_type, COALESCE(reference_type, ''), COALESCE(reference_id, ''),
		       quantity_change, balance_after, COALESCE(reason, ''), created_by, created_at
		FROM inventory_ledgers
		WHERE ($1 = '' OR warehouse_id::text = $1)
		  AND ($2 = '' OR product_id::text = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.pool.Query(ctx, query, warehouseID, productID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ledgers []model.InventoryLedger
	for rows.Next() {
		var l model.InventoryLedger
		err := rows.Scan(
			&l.ID, &l.WarehouseID, &l.ProductID, &l.MutationType,
			&l.ReferenceType, &l.ReferenceID, &l.QuantityChange,
			&l.BalanceAfter, &l.Reason, &l.CreatedBy, &l.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		ledgers = append(ledgers, l)
	}
	return ledgers, nil
}

func (r *InventoryRepository) ListStock(ctx context.Context, warehouseID string) ([]model.CurrentStockItem, error) {
	query := `
		SELECT ib.warehouse_id, w.name, ib.product_id, p.name, p.sku, ib.quantity, ib.updated_at
		FROM inventory_balances ib
		JOIN warehouses w ON ib.warehouse_id = w.id
		JOIN products p ON ib.product_id = p.id
		WHERE ($1 = '' OR ib.warehouse_id::text = $1)
		ORDER BY p.name ASC
	`
	rows, err := r.pool.Query(ctx, query, warehouseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.CurrentStockItem
	for rows.Next() {
		var it model.CurrentStockItem
		err := rows.Scan(
			&it.WarehouseID, &it.WarehouseName, &it.ProductID,
			&it.ProductName, &it.SKU, &it.Quantity, &it.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, it)
	}
	return list, nil
}
