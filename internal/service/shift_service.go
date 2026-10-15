package service

import (
	"context"

	"go-backend/internal/model"
	"go-backend/internal/repository"
)

type ShiftService struct {
	shiftRepo *repository.ShiftRepository
	auditRepo *repository.AuditRepository
}

func NewShiftService(shiftRepo *repository.ShiftRepository, auditRepo *repository.AuditRepository) *ShiftService {
	return &ShiftService{shiftRepo: shiftRepo, auditRepo: auditRepo}
}

func (s *ShiftService) GetActiveShift(ctx context.Context, storeID, terminalID, userID string) (*model.CashierShift, error) {
	return s.shiftRepo.GetActiveShift(ctx, storeID, terminalID, userID)
}

func (s *ShiftService) GetShiftDetail(ctx context.Context, shiftID string) (*model.ShiftDetailResponse, error) {
	shift, err := s.shiftRepo.GetShiftByID(ctx, shiftID)
	if err != nil || shift == nil {
		return nil, model.ErrNotFound("Shift tidak ditemukan")
	}

	cashSales, cashIn, cashOut, err := s.shiftRepo.GetShiftCashTotals(ctx, shiftID)
	if err != nil {
		return nil, model.ErrInternal("Gagal menghitung total kas shift")
	}

	expectedCash := shift.StartingCash + cashSales + cashIn - cashOut
	movements, _ := s.shiftRepo.GetShiftMovements(ctx, shiftID)

	return &model.ShiftDetailResponse{
		Shift:        *shift,
		ExpectedCash: expectedCash,
		CashSales:    cashSales,
		CashIn:       cashIn,
		CashOut:      cashOut,
		Movements:    movements,
	}, nil
}

func (s *ShiftService) OpenShift(ctx context.Context, req model.OpenShiftRequest, userID string) (*model.CashierShift, error) {
	startingCash := req.GetStartingCash()
	if req.StoreID == "" || req.TerminalID == "" || startingCash < 0 {
		return nil, model.ErrValidation("StoreID, TerminalID, dan modal awal (>= 0) wajib diisi")
	}

	// Check if already has open shift
	active, err := s.shiftRepo.GetActiveShift(ctx, req.StoreID, req.TerminalID, userID)
	if err != nil {
		return nil, model.ErrInternal("Gagal memeriksa status shift aktif")
	}
	if active != nil {
		return nil, model.ErrValidation("Anda masih memiliki shift yang sedang aktif pada cabang ini")
	}

	note := req.Note
	shift := &model.CashierShift{
		StoreID:      req.StoreID,
		TerminalID:   req.TerminalID,
		UserID:       userID,
		StartingCash: startingCash,
		Status:       "OPEN",
		Notes:        note,
	}

	if err := s.shiftRepo.OpenShift(ctx, shift); err != nil {
		return nil, model.ErrInternal("Gagal membuka shift")
	}

	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		StoreID:  &req.StoreID,
		ActorID:  &userID,
		Action:   "SHIFT_OPEN",
		Entity:   "cashier_shifts",
		EntityID: shift.ID,
	})

	return shift, nil
}

func (s *ShiftService) CreateCashMovement(ctx context.Context, req model.CreateCashMovementRequest, shiftID, userID string) error {
	req.Normalize()
	if req.MovementType != "IN" && req.MovementType != "OUT" {
		return model.ErrValidation("Jenis pergerakan kas harus 'IN' (cash_in) atau 'OUT' (cash_out)")
	}
	if req.Amount <= 0 {
		return model.ErrValidation("Nominal pergerakan kas harus lebih besar dari 0")
	}
	if req.Category == "" {
		return model.ErrValidation("Kategori pergerakan kas wajib diisi")
	}

	shift, err := s.shiftRepo.GetShiftByID(ctx, shiftID)
	if err != nil || shift == nil {
		return model.ErrNotFound("Shift tidak ditemukan")
	}
	if shift.Status != "OPEN" {
		return model.ErrValidation("Shift telah ditutup, tidak dapat mencatat pergerakan kas")
	}

	m := &model.CashMovement{
		ShiftID:      shiftID,
		MovementType: req.MovementType,
		Amount:       req.Amount,
		Category:     req.Category,
		Reason:       req.Reason,
		CreatedBy:    &userID,
	}

	return s.shiftRepo.CreateCashMovement(ctx, m)
}

func (s *ShiftService) CloseShift(ctx context.Context, shiftID string, req model.CloseShiftRequest, userID string) (*model.CashierShift, error) {
	actualCash := req.GetActualCash()
	notes := req.GetNotes()
	if actualCash < 0 {
		return nil, model.ErrValidation("Jumlah kas fisik aktual (>= 0) wajib diisi")
	}

	shift, err := s.shiftRepo.GetShiftByID(ctx, shiftID)
	if err != nil || shift == nil {
		return nil, model.ErrNotFound("Shift tidak ditemukan")
	}
	if shift.Status != "OPEN" {
		return nil, model.ErrValidation("Shift sudah ditutup sebelumnya")
	}

	// Calculate expected cash = starting_cash + cash_sales + cash_in - cash_out
	cashSales, cashIn, cashOut, err := s.shiftRepo.GetShiftCashTotals(ctx, shiftID)
	if err != nil {
		return nil, model.ErrInternal("Gagal menghitung rekonsiliasi kas shift")
	}

	expectedCash := shift.StartingCash + cashSales + cashIn - cashOut
	cashDiff := actualCash - expectedCash

	if err := s.shiftRepo.CloseShift(ctx, shiftID, actualCash, expectedCash, cashDiff, notes); err != nil {
		return nil, model.ErrInternal("Gagal menutup shift")
	}

	shift.EndingCashActual = &actualCash
	shift.EndingCashExpected = &expectedCash
	shift.CashDifference = &cashDiff
	shift.Status = "CLOSED"
	shift.Notes = notes

	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		StoreID:  &shift.StoreID,
		ActorID:  &userID,
		Action:   "SHIFT_CLOSE",
		Entity:   "cashier_shifts",
		EntityID: shiftID,
	})

	return shift, nil
}
