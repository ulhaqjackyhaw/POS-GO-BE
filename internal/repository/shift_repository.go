package repository

import (
	"context"
	"errors"

	"go-backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ShiftRepository struct {
	pool *pgxpool.Pool
}

func NewShiftRepository(pool *pgxpool.Pool) *ShiftRepository {
	return &ShiftRepository{pool: pool}
}

func (r *ShiftRepository) GetActiveShift(ctx context.Context, storeID, terminalID, userID string) (*model.CashierShift, error) {
	query := `
		SELECT id, store_id, terminal_id, user_id, starting_cash, ending_cash_actual, ending_cash_expected,
		       cash_difference, status, opened_at, closed_at, COALESCE(notes, '')
		FROM cashier_shifts
		WHERE store_id = $1 AND user_id = $2 AND status = 'OPEN'
		ORDER BY opened_at DESC
		LIMIT 1
	`
	var s model.CashierShift
	err := r.pool.QueryRow(ctx, query, storeID, userID).Scan(
		&s.ID, &s.StoreID, &s.TerminalID, &s.UserID, &s.StartingCash,
		&s.EndingCashActual, &s.EndingCashExpected, &s.CashDifference,
		&s.Status, &s.OpenedAt, &s.ClosedAt, &s.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *ShiftRepository) GetShiftByID(ctx context.Context, id string) (*model.CashierShift, error) {
	query := `
		SELECT id, store_id, terminal_id, user_id, starting_cash, ending_cash_actual, ending_cash_expected,
		       cash_difference, status, opened_at, closed_at, COALESCE(notes, '')
		FROM cashier_shifts
		WHERE id = $1
	`
	var s model.CashierShift
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.StoreID, &s.TerminalID, &s.UserID, &s.StartingCash,
		&s.EndingCashActual, &s.EndingCashExpected, &s.CashDifference,
		&s.Status, &s.OpenedAt, &s.ClosedAt, &s.Notes,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *ShiftRepository) OpenShift(ctx context.Context, s *model.CashierShift) error {
	query := `
		INSERT INTO cashier_shifts (store_id, terminal_id, user_id, starting_cash, status, opened_at)
		VALUES ($1, $2, $3, $4, 'OPEN', NOW())
		RETURNING id, opened_at
	`
	return r.pool.QueryRow(ctx, query, s.StoreID, s.TerminalID, s.UserID, s.StartingCash).Scan(&s.ID, &s.OpenedAt)
}

func (r *ShiftRepository) CloseShift(ctx context.Context, shiftID string, actual, expected, diff int64, notes string) error {
	query := `
		UPDATE cashier_shifts
		SET ending_cash_actual = $1,
		    ending_cash_expected = $2,
		    cash_difference = $3,
		    status = 'CLOSED',
		    closed_at = NOW(),
		    notes = $4
		WHERE id = $5 AND status = 'OPEN'
	`
	ct, err := r.pool.Exec(ctx, query, actual, expected, diff, notes, shiftID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return errors.New("shift tidak ditemukan atau sudah ditutup")
	}
	return nil
}

func (r *ShiftRepository) CreateCashMovement(ctx context.Context, m *model.CashMovement) error {
	query := `
		INSERT INTO cash_movements (shift_id, movement_type, amount, category, reason, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, created_at
	`
	return r.pool.QueryRow(ctx, query, m.ShiftID, m.MovementType, m.Amount, m.Category, m.Reason, m.CreatedBy).Scan(&m.ID, &m.CreatedAt)
}

func (r *ShiftRepository) GetShiftCashTotals(ctx context.Context, shiftID string) (cashSales, cashIn, cashOut int64, err error) {
	// Cash sales from transactions completed in this shift
	salesQuery := `
		SELECT COALESCE(SUM(total_amount), 0)
		FROM sales_transactions
		WHERE shift_id = $1 AND status = 'COMPLETED' AND payment_method = 'CASH'
	`
	err = r.pool.QueryRow(ctx, salesQuery, shiftID).Scan(&cashSales)
	if err != nil {
		return 0, 0, 0, err
	}

	// Movements
	inQuery := `SELECT COALESCE(SUM(amount), 0) FROM cash_movements WHERE shift_id = $1 AND movement_type = 'IN'`
	err = r.pool.QueryRow(ctx, inQuery, shiftID).Scan(&cashIn)
	if err != nil {
		return 0, 0, 0, err
	}

	outQuery := `SELECT COALESCE(SUM(amount), 0) FROM cash_movements WHERE shift_id = $1 AND movement_type = 'OUT'`
	err = r.pool.QueryRow(ctx, outQuery, shiftID).Scan(&cashOut)
	if err != nil {
		return 0, 0, 0, err
	}

	return cashSales, cashIn, cashOut, nil
}

func (r *ShiftRepository) GetShiftMovements(ctx context.Context, shiftID string) ([]model.CashMovement, error) {
	query := `
		SELECT id, shift_id, movement_type, amount, category, COALESCE(reason, ''), created_by, created_at
		FROM cash_movements
		WHERE shift_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, shiftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []model.CashMovement
	for rows.Next() {
		var m model.CashMovement
		if err := rows.Scan(&m.ID, &m.ShiftID, &m.MovementType, &m.Amount, &m.Category, &m.Reason, &m.CreatedBy, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, nil
}
