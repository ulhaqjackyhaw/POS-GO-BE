package repository

import (
	"context"
	"errors"

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
	query := `SELECT id, code, name, COALESCE(address, ''), is_active, created_at, updated_at FROM stores WHERE id = $1`
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
	query := `
		INSERT INTO stores (code, name, address, is_active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query, s.Code, s.Name, s.Address, s.IsActive).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

// Warehouses
func (r *OrgRepository) ListWarehouses(ctx context.Context, storeID string) ([]model.Warehouse, error) {
	query := `SELECT id, store_id, code, name, is_active, created_at, updated_at FROM warehouses WHERE store_id = $1 ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var warehouses []model.Warehouse
	for rows.Next() {
		var w model.Warehouse
		if err := rows.Scan(&w.ID, &w.StoreID, &w.Code, &w.Name, &w.IsActive, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		warehouses = append(warehouses, w)
	}
	return warehouses, nil
}

func (r *OrgRepository) GetWarehouseByID(ctx context.Context, id string) (*model.Warehouse, error) {
	query := `SELECT id, store_id, code, name, is_active, created_at, updated_at FROM warehouses WHERE id = $1`
	var w model.Warehouse
	err := r.pool.QueryRow(ctx, query, id).Scan(&w.ID, &w.StoreID, &w.Code, &w.Name, &w.IsActive, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &w, nil
}

// Terminals
func (r *OrgRepository) ListTerminals(ctx context.Context, storeID string) ([]model.Terminal, error) {
	query := `SELECT id, store_id, code, name, is_active, created_at, updated_at FROM terminals WHERE store_id = $1 ORDER BY code ASC`
	rows, err := r.pool.Query(ctx, query, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var terminals []model.Terminal
	for rows.Next() {
		var t model.Terminal
		if err := rows.Scan(&t.ID, &t.StoreID, &t.Code, &t.Name, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		terminals = append(terminals, t)
	}
	return terminals, nil
}

func (r *OrgRepository) GetTerminalByID(ctx context.Context, id string) (*model.Terminal, error) {
	query := `SELECT id, store_id, code, name, is_active, created_at, updated_at FROM terminals WHERE id = $1`
	var t model.Terminal
	err := r.pool.QueryRow(ctx, query, id).Scan(&t.ID, &t.StoreID, &t.Code, &t.Name, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}
