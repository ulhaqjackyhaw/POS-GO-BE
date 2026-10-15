package repository

import (
	"context"
	"errors"

	"go-backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransferRepository struct {
	pool *pgxpool.Pool
}

func NewTransferRepository(pool *pgxpool.Pool) *TransferRepository {
	return &TransferRepository{pool: pool}
}

func (r *TransferRepository) CreateTransfer(ctx context.Context, tx pgx.Tx, t *model.StockTransfer) error {
	query := `
		INSERT INTO stock_transfers (transfer_number, from_warehouse_id, to_warehouse_id, status, requested_by, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	err := tx.QueryRow(ctx, query, t.TransferNumber, t.FromWarehouseID, t.ToWarehouseID, t.Status, t.RequestedBy, t.Notes).
		Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return err
	}

	for i := range t.Items {
		item := &t.Items[i]
		itemQuery := `
			INSERT INTO stock_transfer_items (transfer_id, product_id, quantity, received_quantity)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`
		err = tx.QueryRow(ctx, itemQuery, t.ID, item.ProductID, item.Quantity, item.ReceivedQuantity).Scan(&item.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *TransferRepository) GetTransferByID(ctx context.Context, id string) (*model.StockTransfer, error) {
	query := `
		SELECT id, transfer_number, from_warehouse_id, to_warehouse_id, status, requested_by, COALESCE(notes, ''), created_at, updated_at
		FROM stock_transfers
		WHERE id = $1
	`
	var t model.StockTransfer
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.TransferNumber, &t.FromWarehouseID, &t.ToWarehouseID,
		&t.Status, &t.RequestedBy, &t.Notes, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	// Fetch items
	rows, err := r.pool.Query(ctx, `SELECT id, transfer_id, product_id, quantity, received_quantity FROM stock_transfer_items WHERE transfer_id = $1`, t.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var it model.StockTransferItem
		if err := rows.Scan(&it.ID, &it.TransferID, &it.ProductID, &it.Quantity, &it.ReceivedQuantity); err != nil {
			return nil, err
		}
		t.Items = append(t.Items, it)
	}

	return &t, nil
}

func (r *TransferRepository) UpdateStatus(ctx context.Context, tx pgx.Tx, id, status string) error {
	query := `UPDATE stock_transfers SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := tx.Exec(ctx, query, status, id)
	return err
}

func (r *TransferRepository) ListTransfers(ctx context.Context) ([]model.StockTransfer, error) {
	query := `
		SELECT id, transfer_number, from_warehouse_id, to_warehouse_id, status, requested_by, COALESCE(notes, ''), created_at, updated_at
		FROM stock_transfers
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.StockTransfer
	for rows.Next() {
		var t model.StockTransfer
		if err := rows.Scan(&t.ID, &t.TransferNumber, &t.FromWarehouseID, &t.ToWarehouseID, &t.Status, &t.RequestedBy, &t.Notes, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, nil
}
