package repository

import (
	"context"
	"errors"

	"go-backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ApprovalRepository struct {
	pool *pgxpool.Pool
}

func NewApprovalRepository(pool *pgxpool.Pool) *ApprovalRepository {
	return &ApprovalRepository{pool: pool}
}

func (r *ApprovalRepository) Create(ctx context.Context, a *model.ApprovalRequest) error {
	query := `
		INSERT INTO approval_requests (store_id, action_type, status, requested_by, payload, reason, created_at, updated_at)
		VALUES ($1, $2, 'PENDING', $3, $4, $5, NOW(), NOW())
		RETURNING id, status, created_at, updated_at
	`
	return r.pool.QueryRow(ctx, query, a.StoreID, a.ActionType, a.RequestedBy, a.Payload, a.Reason).
		Scan(&a.ID, &a.Status, &a.CreatedAt, &a.UpdatedAt)
}

func (r *ApprovalRepository) GetByID(ctx context.Context, id string) (*model.ApprovalRequest, error) {
	query := `
		SELECT id, store_id, action_type, status, requested_by, approved_by, payload::text, reason, rejection_reason, created_at, updated_at
		FROM approval_requests
		WHERE id = $1
	`
	var a model.ApprovalRequest
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.StoreID, &a.ActionType, &a.Status, &a.RequestedBy, &a.ApprovedBy,
		&a.Payload, &a.Reason, &a.RejectionReason, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *ApprovalRepository) Review(ctx context.Context, id, approverID, status string, rejectionReason *string) error {
	query := `
		UPDATE approval_requests
		SET status = $1, approved_by = $2, rejection_reason = $3, updated_at = NOW()
		WHERE id = $4 AND status = 'PENDING'
	`
	ct, err := r.pool.Exec(ctx, query, status, approverID, rejectionReason, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return errors.New("permintaan approval tidak ditemukan atau sudah diproses")
	}
	return nil
}

func (r *ApprovalRepository) List(ctx context.Context, storeID, status string) ([]model.ApprovalRequest, error) {
	query := `
		SELECT id, store_id, action_type, status, requested_by, approved_by, payload::text, reason, rejection_reason, created_at, updated_at
		FROM approval_requests
		WHERE ($1 = '' OR store_id = $1)
		  AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, storeID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.ApprovalRequest
	for rows.Next() {
		var a model.ApprovalRequest
		err := rows.Scan(
			&a.ID, &a.StoreID, &a.ActionType, &a.Status, &a.RequestedBy, &a.ApprovedBy,
			&a.Payload, &a.Reason, &a.RejectionReason, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}
