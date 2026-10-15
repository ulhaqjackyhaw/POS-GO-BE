package repository

import (
	"context"
	"time"

	"go-backend/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportRepository struct {
	pool *pgxpool.Pool
}

func NewReportRepository(pool *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{pool: pool}
}

func (r *ReportRepository) GetSalesSummary(ctx context.Context, storeID string, from, to time.Time) (*model.SalesSummaryReport, error) {
	query := `
		SELECT 
			COUNT(id),
			COALESCE(SUM(subtotal), 0),
			COALESCE(SUM(discount_amount), 0),
			COALESCE(SUM(tax_amount), 0),
			COALESCE(SUM(total_amount), 0)
		FROM sales_transactions
		WHERE status = 'COMPLETED'
		  AND ($1 = '' OR store_id = $1)
		  AND created_at >= $2 AND created_at <= $3
	`
	var s model.SalesSummaryReport
	err := r.pool.QueryRow(ctx, query, storeID, from, to).Scan(
		&s.TotalTransactions, &s.GrossSales, &s.TotalDiscount, &s.TotalTax, &s.NetSales,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ReportRepository) GetTopProducts(ctx context.Context, storeID string, from, to time.Time, limit int) ([]model.TopProductItem, error) {
	query := `
		SELECT 
			sti.product_id,
			sti.product_name,
			SUM(sti.quantity) AS total_qty,
			SUM(sti.total) AS total_val
		FROM sales_transaction_items sti
		JOIN sales_transactions st ON sti.transaction_id = st.id
		WHERE st.status = 'COMPLETED'
		  AND ($1 = '' OR st.store_id = $1)
		  AND st.created_at >= $2 AND st.created_at <= $3
		GROUP BY sti.product_id, sti.product_name
		ORDER BY total_qty DESC
		LIMIT $4
	`
	rows, err := r.pool.Query(ctx, query, storeID, from, to, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.TopProductItem
	for rows.Next() {
		var item model.TopProductItem
		if err := rows.Scan(&item.ProductID, &item.ProductName, &item.Quantity, &item.TotalSales); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, nil
}
