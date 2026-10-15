package repository

import (
	"context"
	"errors"

	"go-backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CatalogRepository struct {
	pool *pgxpool.Pool
}

func NewCatalogRepository(pool *pgxpool.Pool) *CatalogRepository {
	return &CatalogRepository{pool: pool}
}

// ListProducts returns products with effective prices for a specific store and stock in warehouse
func (r *CatalogRepository) ListProducts(ctx context.Context, storeID, warehouseID, search string) ([]model.ProductWithPrice, error) {
	query := `
		SELECT 
			p.id, p.sku, COALESCE(p.barcode, ''), p.name, p.category_id, p.unit, p.base_price, p.is_active, p.created_at, p.updated_at,
			COALESCE(spp.price, p.base_price) AS effective_price,
			(spp.price IS NOT NULL) AS is_override,
			COALESCE(ib.quantity, 0) AS current_stock,
			$2::text AS warehouse_id
		FROM products p
		LEFT JOIN store_product_prices spp ON p.id = spp.product_id AND spp.store_id::text = $1 AND spp.is_active = true
		LEFT JOIN inventory_balances ib ON p.id = ib.product_id AND ib.warehouse_id::text = $2
		WHERE p.is_active = true
		  AND ($3 = '' OR p.name ILIKE '%' || $3 || '%' OR p.sku ILIKE '%' || $3 || '%' OR p.barcode ILIKE '%' || $3 || '%')
		ORDER BY p.name ASC
	`
	rows, err := r.pool.Query(ctx, query, storeID, warehouseID, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.ProductWithPrice
	for rows.Next() {
		var item model.ProductWithPrice
		err := rows.Scan(
			&item.ID, &item.SKU, &item.Barcode, &item.Name, &item.CategoryID,
			&item.Unit, &item.BasePrice, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
			&item.EffectivePrice, &item.IsOverride, &item.CurrentStock, &item.WarehouseID,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *CatalogRepository) GetProductByID(ctx context.Context, productID string) (*model.Product, error) {
	if productID == "" {
		return nil, nil
	}
	query := `
		SELECT id, sku, COALESCE(barcode, ''), name, category_id, unit, base_price, is_active, created_at, updated_at
		FROM products
		WHERE id::text = $1
	`
	var p model.Product
	err := r.pool.QueryRow(ctx, query, productID).Scan(
		&p.ID, &p.SKU, &p.Barcode, &p.Name, &p.CategoryID, &p.Unit,
		&p.BasePrice, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *CatalogRepository) GetProductWithEffectivePrice(ctx context.Context, productID, storeID, warehouseID string) (*model.ProductWithPrice, error) {
	if productID == "" {
		return nil, nil
	}
	query := `
		SELECT 
			p.id, p.sku, COALESCE(p.barcode, ''), p.name, p.category_id, p.unit, p.base_price, p.is_active, p.created_at, p.updated_at,
			COALESCE(spp.price, p.base_price) AS effective_price,
			(spp.price IS NOT NULL) AS is_override,
			COALESCE(ib.quantity, 0) AS current_stock,
			$3::text AS warehouse_id
		FROM products p
		LEFT JOIN store_product_prices spp ON p.id = spp.product_id AND spp.store_id::text = $2 AND spp.is_active = true
		LEFT JOIN inventory_balances ib ON p.id = ib.product_id AND ib.warehouse_id::text = $3
		WHERE p.id::text = $1
	`
	var item model.ProductWithPrice
	err := r.pool.QueryRow(ctx, query, productID, storeID, warehouseID).Scan(
		&item.ID, &item.SKU, &item.Barcode, &item.Name, &item.CategoryID,
		&item.Unit, &item.BasePrice, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
		&item.EffectivePrice, &item.IsOverride, &item.CurrentStock, &item.WarehouseID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *CatalogRepository) CreateProduct(ctx context.Context, p *model.Product) error {
	query := `
		INSERT INTO products (sku, barcode, name, category_id, unit, base_price, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query, p.SKU, p.Barcode, p.Name, p.CategoryID, p.Unit, p.BasePrice, p.IsActive).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *CatalogRepository) SetStorePrice(ctx context.Context, storeID, productID string, price int64) error {
	query := `
		INSERT INTO store_product_prices (store_id, product_id, price, is_active, updated_at)
		VALUES ($1, $2, $3, true, NOW())
		ON CONFLICT (store_id, product_id)
		DO UPDATE SET price = EXCLUDED.price, is_active = true, updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, storeID, productID, price)
	return err
}

func (r *CatalogRepository) ListCategories(ctx context.Context) ([]model.Category, error) {
	query := `SELECT id, name, created_at FROM categories ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []model.Category
	for rows.Next() {
		var c model.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.CreatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}
