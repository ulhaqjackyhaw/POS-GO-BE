package model

import "fmt"

type AppError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"error"`
	Code       string `json:"code"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func NewAppError(statusCode int, code, message string) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
	}
}

// Common Standard Errors matching PRD and AGENT.md
func ErrValidation(msg string) *AppError {
	return NewAppError(400, "VALIDATION_ERROR", msg)
}

func ErrUnauthenticated(msg string) *AppError {
	if msg == "" {
		msg = "Autentikasi diperlukan"
	}
	return NewAppError(401, "UNAUTHENTICATED", msg)
}

func ErrForbidden(msg string) *AppError {
	if msg == "" {
		msg = "Akses ditolak"
	}
	return NewAppError(403, "FORBIDDEN", msg)
}

func ErrNotFound(msg string) *AppError {
	if msg == "" {
		msg = "Data tidak ditemukan"
	}
	return NewAppError(404, "NOT_FOUND", msg)
}

func ErrInsufficientStock(msg string) *AppError {
	if msg == "" {
		msg = "Stok tidak mencukupi"
	}
	return NewAppError(400, "INSUFFICIENT_STOCK", msg)
}

func ErrPriceChanged(msg string) *AppError {
	if msg == "" {
		msg = "Harga telah berubah sejak quote"
	}
	return NewAppError(409, "PRICE_CHANGED", msg)
}

func ErrShiftNotOpen(msg string) *AppError {
	if msg == "" {
		msg = "Tidak ada shift aktif yang memenuhi syarat"
	}
	return NewAppError(400, "SHIFT_NOT_OPEN", msg)
}

func ErrIdempotencyConflict(msg string) *AppError {
	if msg == "" {
		msg = "Idempotency key yang sama digunakan dengan payload berbeda"
	}
	return NewAppError(409, "IDEMPOTENCY_CONFLICT", msg)
}

func ErrApprovalRequired(msg string) *AppError {
	if msg == "" {
		msg = "Tindakan ini memerlukan persetujuan"
	}
	return NewAppError(400, "APPROVAL_REQUIRED", msg)
}

func ErrCannotApproveOwnRequest() *AppError {
	return NewAppError(403, "CANNOT_APPROVE_OWN_REQUEST", "Pemohon tidak dapat menyetujui permintaannya sendiri")
}

func ErrSyncConflict(msg string) *AppError {
	if msg == "" {
		msg = "Transaksi offline mengalami konflik"
	}
	return NewAppError(409, "SYNC_CONFLICT", msg)
}

func ErrInternal(msg string) *AppError {
	if msg == "" {
		msg = "Terjadi kesalahan internal pada server"
	}
	return NewAppError(500, "INTERNAL_SERVER_ERROR", msg)
}
