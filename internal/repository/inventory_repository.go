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

// GetBalance returns stock for a single product in store
func (r *InventoryRepository) GetBalance(ctx context.Context, storeID, productID string) (int64, error) {
	if storeID == "" || productID == "" {
		return 0, nil
	}
	query := `SELECT COALESCE(quantity, 0) FROM inventory_balances WHERE store_id::text = $1 AND product_id::text = $2`
	var qty int64
	err := r.pool.QueryRow(ctx, query, storeID, productID).Scan(&qty)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return qty, nil
}

// LockBalancesForUpdate locks multiple stock balance rows in deterministic order to prevent deadlocks
func (r *InventoryRepository) LockBalancesForUpdate(ctx context.Context, tx pgx.Tx, storeID string, productIDs []string) (map[string]int64, error) {
	// Deterministic sorting of product IDs as mandated by AGENT.md
	sortedIDs := make([]string, len(productIDs))
	copy(sortedIDs, productIDs)
	sort.Strings(sortedIDs)

	balances := make(map[string]int64)

	// Lock each row in deterministic sorted order
	for _, pid := range sortedIDs {
		// Ensure row exists or create with 0 if missing
		_, err := tx.Exec(ctx, `
			INSERT INTO inventory_balances (store_id, product_id, quantity)
			VALUES ($1, $2, 0)
			ON CONFLICT (store_id, product_id) DO NOTHING
		`, storeID, pid)
		if err != nil {
			return nil, err
		}

		var qty int64
		err = tx.QueryRow(ctx, `
			SELECT quantity
			FROM inventory_balances
			WHERE store_id = $1 AND product_id = $2
			FOR UPDATE
		`, storeID, pid).Scan(&qty)
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
	storeID, productID string,
	qtyDelta int64,
	mutationType, refType, refID, reason string,
	actorID *string,
) (int64, error) {
	var newBalance int64
	// Atomic update enforced by DB constraint chk_stock_non_negative
	err := tx.QueryRow(ctx, `
		UPDATE inventory_balances
		SET quantity = quantity + $1, updated_at = NOW()
		WHERE store_id = $2 AND product_id = $3
		RETURNING quantity
	`, qtyDelta, storeID, productID).Scan(&newBalance)
	if err != nil {
		return 0, err
	}

	// Insert immutable ledger
	_, err = tx.Exec(ctx, `
		INSERT INTO inventory_ledgers (
			store_id, product_id, mutation_type, reference_type, reference_id,
			quantity_change, balance_after, reason, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`, storeID, productID, mutationType, refType, refID, qtyDelta, newBalance, reason, actorID)
	if err != nil {
		return 0, err
	}

	return newBalance, nil
}

func (r *InventoryRepository) ListLedgers(ctx context.Context, storeID, productID string, limit, offset int) ([]model.InventoryLedger, error) {
	query := `
		SELECT id, store_id, product_id, mutation_type, COALESCE(reference_type, ''), COALESCE(reference_id, ''),
		       quantity_change, balance_after, COALESCE(reason, ''), created_by, created_at
		FROM inventory_ledgers
		WHERE ($1 = '' OR store_id::text = $1)
		  AND ($2 = '' OR product_id::text = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	rows, err := r.pool.Query(ctx, query, storeID, productID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ledgers []model.InventoryLedger
	for rows.Next() {
		var l model.InventoryLedger
		err := rows.Scan(
			&l.ID, &l.StoreID, &l.ProductID, &l.MutationType,
			&l.ReferenceType, &l.ReferenceID, &l.QuantityChange,
			&l.BalanceAfter, &l.Reason, &l.CreatedBy, &l.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		// Also populate legacy WarehouseID field for backward compat
		l.WarehouseID = l.StoreID
		ledgers = append(ledgers, l)
	}
	return ledgers, nil
}

func (r *InventoryRepository) ListStock(ctx context.Context, storeID string) ([]model.CurrentStockItem, error) {
	var query string
	var rows pgx.Rows
	var err error

	if storeID != "" {
		query = `
			SELECT 
				$1::text AS store_id,
				COALESCE(s.name, '') AS store_name,
				p.id::text AS product_id,
				p.name AS product_name,
				p.sku,
				COALESCE(p.unit, 'pcs') AS unit,
				COALESCE(ib.quantity, 0) AS quantity,
				COALESCE(ib.updated_at, p.updated_at) AS updated_at
			FROM products p
			LEFT JOIN stores s ON s.id::text = $1
			LEFT JOIN inventory_balances ib ON ib.store_id::text = $1 AND ib.product_id = p.id
			WHERE p.is_active = true
			ORDER BY p.name ASC
		`
		rows, err = r.pool.Query(ctx, query, storeID)
	} else {
		query = `
			SELECT 
				ib.store_id::text,
				COALESCE(s.name, '') AS store_name,
				ib.product_id::text,
				p.name AS product_name,
				p.sku,
				COALESCE(p.unit, 'pcs') AS unit,
				ib.quantity,
				ib.updated_at
			FROM inventory_balances ib
			JOIN stores s ON ib.store_id = s.id
			JOIN products p ON ib.product_id = p.id
			ORDER BY p.name ASC
		`
		rows, err = r.pool.Query(ctx, query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.CurrentStockItem
	for rows.Next() {
		var it model.CurrentStockItem
		err := rows.Scan(
			&it.StoreID, &it.StoreName, &it.ProductID,
			&it.ProductName, &it.SKU, &it.Unit, &it.Quantity, &it.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		// Alias legacy fields
		it.WarehouseID = it.StoreID
		it.WarehouseName = it.StoreName
		list = append(list, it)
	}
	return list, nil
}
