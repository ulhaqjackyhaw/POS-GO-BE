package repository

import (
	"context"

	"go-backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

func (r *AuditRepository) Create(ctx context.Context, l *model.AuditLog) error {
	query := `
		INSERT INTO audit_logs (store_id, actor_id, action, entity, entity_id, details, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, NOW())
		RETURNING id, created_at
	`
	detailsJSON := l.Details
	if detailsJSON == "" {
		detailsJSON = "{}"
	}
	return r.pool.QueryRow(ctx, query, l.StoreID, l.ActorID, l.Action, l.Entity, l.EntityID, detailsJSON, l.IPAddress).
		Scan(&l.ID, &l.CreatedAt)
}

func (r *AuditRepository) List(ctx context.Context, storeID string, limit, offset int) ([]model.AuditLog, error) {
	query := `
		SELECT id, store_id, actor_id, action, entity, entity_id, details::text, COALESCE(ip_address, ''), created_at
		FROM audit_logs
		WHERE ($1 = '' OR store_id::text = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, storeID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.AuditLog
	for rows.Next() {
		var l model.AuditLog
		err := rows.Scan(
			&l.ID, &l.StoreID, &l.ActorID, &l.Action, &l.Entity, &l.EntityID,
			&l.Details, &l.IPAddress, &l.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}
