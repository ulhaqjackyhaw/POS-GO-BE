package service

import (
	"context"

	"go-backend/internal/model"
	"go-backend/internal/repository"
)

type ApprovalService struct {
	approvalRepo *repository.ApprovalRepository
	auditRepo    *repository.AuditRepository
}

func NewApprovalService(approvalRepo *repository.ApprovalRepository, auditRepo *repository.AuditRepository) *ApprovalService {
	return &ApprovalService{approvalRepo: approvalRepo, auditRepo: auditRepo}
}

func (s *ApprovalService) CreateRequest(ctx context.Context, req model.CreateApprovalRequest, requesterID string) (*model.ApprovalRequest, error) {
	if req.ActionType == "" || req.Reason == "" {
		return nil, model.ErrValidation("Action type dan alasan wajib diisi")
	}

	ar := &model.ApprovalRequest{
		StoreID:     req.StoreID,
		ActionType:  req.ActionType,
		RequestedBy: requesterID,
		Payload:     req.Payload,
		Reason:      req.Reason,
	}

	if err := s.approvalRepo.Create(ctx, ar); err != nil {
		return nil, model.ErrInternal("Gagal membuat permintaan approval")
	}

	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		StoreID:  req.StoreID,
		ActorID:  &requesterID,
		Action:   "APPROVAL_REQUEST_CREATED",
		Entity:   "approval_requests",
		EntityID: ar.ID,
		Details:  req.Reason,
	})

	return ar, nil
}

func (s *ApprovalService) ReviewRequest(ctx context.Context, approvalID string, req model.ReviewApprovalRequest, approverID string) error {
	if req.Status != "APPROVED" && req.Status != "REJECTED" {
		return model.ErrValidation("Status review harus 'APPROVED' atau 'REJECTED'")
	}

	ar, err := s.approvalRepo.GetByID(ctx, approvalID)
	if err != nil || ar == nil {
		return model.ErrNotFound("Permintaan approval tidak ditemukan")
	}

	if ar.Status != "PENDING" {
		return model.ErrValidation("Permintaan approval sudah diproses sebelumnya")
	}

	// Rule in PRD 6.1 & AGENT.md Section 5: "Pemohon tidak boleh menyetujui permintaannya sendiri"
	if ar.RequestedBy == approverID {
		return model.ErrCannotApproveOwnRequest()
	}

	if err := s.approvalRepo.Review(ctx, approvalID, approverID, req.Status, req.RejectionReason); err != nil {
		return model.ErrInternal("Gagal memproses review approval")
	}

	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		StoreID:  ar.StoreID,
		ActorID:  &approverID,
		Action:   "APPROVAL_" + req.Status,
		Entity:   "approval_requests",
		EntityID: approvalID,
	})

	return nil
}

func (s *ApprovalService) ListRequests(ctx context.Context, storeID, status string) ([]model.ApprovalRequest, error) {
	return s.approvalRepo.List(ctx, storeID, status)
}

func (s *ApprovalService) GetByID(ctx context.Context, id string) (*model.ApprovalRequest, error) {
	ar, err := s.approvalRepo.GetByID(ctx, id)
	if err != nil || ar == nil {
		return nil, model.ErrNotFound("Permintaan approval tidak ditemukan")
	}
	return ar, nil
}
