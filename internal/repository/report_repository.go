package repository

import (
	"context"
	"strings"
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
		  AND ($1 = '' OR store_id::text = $1)
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
		  AND ($1 = '' OR st.store_id::text = $1)
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

func indonesianDayName(dayStr string) string {
	switch strings.ToLower(strings.TrimSpace(dayStr)) {
	case "sunday":
		return "Minggu"
	case "monday":
		return "Senin"
	case "tuesday":
		return "Selasa"
	case "wednesday":
		return "Rabu"
	case "thursday":
		return "Kamis"
	case "friday":
		return "Jumat"
	case "saturday":
		return "Sabtu"
	default:
		return strings.TrimSpace(dayStr)
	}
}

func (r *ReportRepository) GetTerminalSalesReport(ctx context.Context, storeID string, from, to time.Time) (*model.TerminalReportResponse, error) {
	// 1. Summary Query per Terminal
	summaryQuery := `
		SELECT 
			COALESCE(st.terminal_id::text, '') AS term_id,
			COALESCE(t.code, 'POS-KASIR') AS term_code,
			COALESCE(t.name, 'Terminal Kasir') AS term_name,
			COUNT(st.id) FILTER (WHERE st.status = 'COMPLETED') AS completed_tx,
			COALESCE(SUM(st.subtotal) FILTER (WHERE st.status = 'COMPLETED'), 0) AS gross_sales,
			COALESCE(SUM(st.discount_amount) FILTER (WHERE st.status = 'COMPLETED'), 0) AS total_discount,
			COALESCE(SUM(st.tax_amount) FILTER (WHERE st.status = 'COMPLETED'), 0) AS total_tax,
			COALESCE(SUM(st.total_amount) FILTER (WHERE st.status = 'COMPLETED'), 0) AS net_sales,
			COUNT(st.id) FILTER (WHERE st.status = 'VOIDED' OR st.status = 'REFUNDED') AS void_count,
			COALESCE(SUM(st.total_amount) FILTER (WHERE st.status = 'VOIDED' OR st.status = 'REFUNDED'), 0) AS void_amount
		FROM sales_transactions st
		LEFT JOIN terminals t ON st.terminal_id = t.id
		WHERE ($1 = '' OR st.store_id::text = $1)
		  AND st.created_at >= $2 AND st.created_at <= $3
		GROUP BY term_id, term_code, term_name
		ORDER BY net_sales DESC, term_code ASC
	`
	sRows, err := r.pool.Query(ctx, summaryQuery, storeID, from, to)
	if err != nil {
		return nil, err
	}
	defer sRows.Close()

	var summaries []model.TerminalSummaryItem
	for sRows.Next() {
		var it model.TerminalSummaryItem
		if err := sRows.Scan(
			&it.TerminalID, &it.TerminalCode, &it.TerminalName,
			&it.TransactionCount, &it.GrossSales, &it.TotalDiscount, &it.TotalTax, &it.NetSales,
			&it.VoidCount, &it.VoidAmount,
		); err != nil {
			return nil, err
		}
		summaries = append(summaries, it)
	}

	// 2. Daily breakdown per Terminal
	dailyQuery := `
		SELECT 
			TO_CHAR(st.created_at, 'YYYY-MM-DD') AS tx_date,
			TO_CHAR(st.created_at, 'Day') AS day_raw,
			COALESCE(st.terminal_id::text, '') AS term_id,
			COALESCE(t.code, 'POS-KASIR') AS term_code,
			COALESCE(t.name, 'Terminal Kasir') AS term_name,
			COUNT(st.id) FILTER (WHERE st.status = 'COMPLETED') AS completed_tx,
			COALESCE(SUM(st.subtotal) FILTER (WHERE st.status = 'COMPLETED'), 0) AS gross_sales,
			COALESCE(SUM(st.discount_amount) FILTER (WHERE st.status = 'COMPLETED'), 0) AS total_discount,
			COALESCE(SUM(st.tax_amount) FILTER (WHERE st.status = 'COMPLETED'), 0) AS total_tax,
			COALESCE(SUM(st.total_amount) FILTER (WHERE st.status = 'COMPLETED'), 0) AS net_sales,
			COUNT(st.id) FILTER (WHERE st.status = 'VOIDED' OR st.status = 'REFUNDED') AS void_count,
			COALESCE(SUM(st.total_amount) FILTER (WHERE st.status = 'VOIDED' OR st.status = 'REFUNDED'), 0) AS void_amount
		FROM sales_transactions st
		LEFT JOIN terminals t ON st.terminal_id = t.id
		WHERE ($1 = '' OR st.store_id::text = $1)
		  AND st.created_at >= $2 AND st.created_at <= $3
		GROUP BY tx_date, day_raw, term_id, term_code, term_name
		ORDER BY tx_date DESC, term_code ASC
	`
	dRows, err := r.pool.Query(ctx, dailyQuery, storeID, from, to)
	if err != nil {
		return nil, err
	}
	defer dRows.Close()

	var dailies []model.TerminalDailyItem
	for dRows.Next() {
		var it model.TerminalDailyItem
		var dayRaw string
		if err := dRows.Scan(
			&it.Date, &dayRaw, &it.TerminalID, &it.TerminalCode, &it.TerminalName,
			&it.TransactionCount, &it.GrossSales, &it.TotalDiscount, &it.TotalTax, &it.NetSales,
			&it.VoidCount, &it.VoidAmount,
		); err != nil {
			return nil, err
		}
		it.DayName = indonesianDayName(dayRaw)
		dailies = append(dailies, it)
	}

	return &model.TerminalReportResponse{
		Summary:        summaries,
		DailyBreakdown: dailies,
	}, nil
}
