package repository

import (
	"context"
	"errors"
	"strings"

	"go-backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrgRepository struct {
	pool *pgxpool.Pool
}

func NewOrgRepository(pool *pgxpool.Pool) *OrgRepository {
	return &OrgRepository{pool: pool}
}

// Stores
func (r *OrgRepository) ListStores(ctx context.Context) ([]model.Store, error) {
	query := `SELECT id, code, name, COALESCE(address, ''), is_active, created_at, updated_at FROM stores ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stores []model.Store
	for rows.Next() {
		var s model.Store
		if err := rows.Scan(&s.ID, &s.Code, &s.Name, &s.Address, &s.IsActive, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		stores = append(stores, s)
	}
	return stores, nil
}

func (r *OrgRepository) GetStoreByID(ctx context.Context, id string) (*model.Store, error) {
	if id == "" {
		return nil, nil
	}
	query := `SELECT id, code, name, COALESCE(address, ''), is_active, created_at, updated_at FROM stores WHERE id::text = $1`
	var s model.Store
	err := r.pool.QueryRow(ctx, query, id).Scan(&s.ID, &s.Code, &s.Name, &s.Address, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *OrgRepository) CreateStore(ctx context.Context, s *model.Store) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	queryStore := `
		INSERT INTO stores (code, name, address, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	if err := tx.QueryRow(ctx, queryStore, s.Code, s.Name, s.Address, s.IsActive).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt); err != nil {
		return err
	}

	// Otomatis buat Terminal Kasir 1 default untuk cabang ini
	termCode := "TERM-01-" + s.Code
	if strings.HasPrefix(s.Code, "STORE-") {
		termCode = "TERM-01-" + strings.TrimPrefix(s.Code, "STORE-")
	}
	termName := "Kasir 1 " + s.Name

	queryTerminal := `
		INSERT INTO terminals (store_id, code, name, is_active)
		VALUES ($1, $2, $3, true)
		ON CONFLICT (code) DO NOTHING
	`
	if _, err := tx.Exec(ctx, queryTerminal, s.ID, termCode, termName); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Warehouses (Deprecated - stock is now directly managed per Store)
func (r *OrgRepository) ListWarehouses(ctx context.Context, storeID string) ([]model.Warehouse, error) {
	return []model.Warehouse{}, nil
}

func (r *OrgRepository) GetWarehouseByID(ctx context.Context, id string) (*model.Warehouse, error) {
	return nil, nil
}

func (r *OrgRepository) CreateWarehouse(ctx context.Context, w *model.Warehouse) error {
	w.ID = "wh-deprecated"
	return nil
}

func (r *OrgRepository) UpdateWarehouse(ctx context.Context, w *model.Warehouse) error {
	return nil
}

func (r *OrgRepository) DeleteWarehouse(ctx context.Context, id string) error {
	return nil
}


// Terminals
func (r *OrgRepository) ListTerminals(ctx context.Context, storeID string) ([]model.Terminal, error) {
	query := `
		SELECT 
			t.id, t.store_id, COALESCE(s.name, ''), COALESCE(s.code, ''),
			t.code, t.name, t.is_active, t.assigned_cashier_id,
			COALESCE(u.full_name, ''), COALESCE(u.username, ''),
			CASE WHEN cs.id IS NOT NULL THEN true ELSE false END AS has_active_shift,
			COALESCE(cs.id::text, '') AS active_shift_id,
			COALESCE(cs.user_id::text, '') AS active_cashier_id,
			COALESCE(cu.full_name, '') AS active_cashier_name,
			COALESCE(cu.username, '') AS active_cashier_username,
			t.created_at, t.updated_at 
		FROM terminals t
		LEFT JOIN stores s ON t.store_id = s.id
		LEFT JOIN users u ON t.assigned_cashier_id = u.id
		LEFT JOIN cashier_shifts cs ON cs.terminal_id = t.id AND cs.status = 'OPEN'
		LEFT JOIN users cu ON cs.user_id = cu.id
		WHERE ($1 = '' OR t.store_id::text = $1) 
		ORDER BY t.code ASC`
	rows, err := r.pool.Query(ctx, query, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var terminals []model.Terminal
	for rows.Next() {
		var t model.Terminal
		if err := rows.Scan(
			&t.ID, &t.StoreID, &t.StoreName, &t.StoreCode,
			&t.Code, &t.Name, &t.IsActive, &t.AssignedCashierID,
			&t.CashierName, &t.CashierUsername,
			&t.HasActiveShift, &t.ActiveShiftID, &t.ActiveCashierID,
			&t.ActiveCashierName, &t.ActiveCashierUser,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		terminals = append(terminals, t)
	}
	return terminals, nil
}

func (r *OrgRepository) GetTerminalByID(ctx context.Context, id string) (*model.Terminal, error) {
	if id == "" {
		return nil, nil
	}
	query := `
		SELECT 
			t.id, t.store_id, COALESCE(s.name, ''), COALESCE(s.code, ''),
			t.code, t.name, t.is_active, t.assigned_cashier_id,
			COALESCE(u.full_name, ''), COALESCE(u.username, ''),
			CASE WHEN cs.id IS NOT NULL THEN true ELSE false END AS has_active_shift,
			COALESCE(cs.id::text, '') AS active_shift_id,
			COALESCE(cs.user_id::text, '') AS active_cashier_id,
			COALESCE(cu.full_name, '') AS active_cashier_name,
			COALESCE(cu.username, '') AS active_cashier_username,
			t.created_at, t.updated_at 
		FROM terminals t
		LEFT JOIN stores s ON t.store_id = s.id
		LEFT JOIN users u ON t.assigned_cashier_id = u.id
		LEFT JOIN cashier_shifts cs ON cs.terminal_id = t.id AND cs.status = 'OPEN'
		LEFT JOIN users cu ON cs.user_id = cu.id
		WHERE t.id::text = $1`
	var t model.Terminal
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.StoreID, &t.StoreName, &t.StoreCode,
		&t.Code, &t.Name, &t.IsActive, &t.AssignedCashierID,
		&t.CashierName, &t.CashierUsername,
		&t.HasActiveShift, &t.ActiveShiftID, &t.ActiveCashierID,
		&t.ActiveCashierName, &t.ActiveCashierUser,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *OrgRepository) CreateTerminal(ctx context.Context, t *model.Terminal) error {
	query := `
		INSERT INTO terminals (store_id, code, name, is_active, assigned_cashier_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query, t.StoreID, t.Code, t.Name, t.IsActive, t.AssignedCashierID).
		Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *OrgRepository) UpdateTerminal(ctx context.Context, id string, name string, isActive *bool, assignedCashierID *string) (*model.Terminal, error) {
	current, err := r.GetTerminalByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, nil
	}

	newName := current.Name
	if name != "" {
		newName = name
	}
	newActive := current.IsActive
	if isActive != nil {
		newActive = *isActive
	}
	newCashierID := current.AssignedCashierID
	if assignedCashierID != nil {
		if *assignedCashierID == "" {
			newCashierID = nil
		} else {
			newCashierID = assignedCashierID
		}
	}

	query := `
		UPDATE terminals
		SET name = $1, is_active = $2, assigned_cashier_id = $3, updated_at = NOW()
		WHERE id::text = $4
	`
	_, err = r.pool.Exec(ctx, query, newName, newActive, newCashierID, id)
	if err != nil {
		return nil, err
	}

	return r.GetTerminalByID(ctx, id)
}

func (r *OrgRepository) DeleteTerminal(ctx context.Context, id string) error {
	// First try soft delete (deactivate) or delete if unused
	query := `DELETE FROM terminals WHERE id::text = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		// If foreign key constraint, fallback to deactivate
		_, fallbackErr := r.pool.Exec(ctx, `UPDATE terminals SET is_active = false, updated_at = NOW() WHERE id::text = $1`, id)
		if fallbackErr != nil {
			return err
		}
	}
	return nil
}

