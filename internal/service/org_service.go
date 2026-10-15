package service

import (
	"context"

	"go-backend/internal/model"
	"go-backend/internal/repository"
)

type OrgService struct {
	orgRepo *repository.OrgRepository
}

func NewOrgService(orgRepo *repository.OrgRepository) *OrgService {
	return &OrgService{orgRepo: orgRepo}
}

func (s *OrgService) ListStores(ctx context.Context) ([]model.Store, error) {
	return s.orgRepo.ListStores(ctx)
}

func (s *OrgService) GetStore(ctx context.Context, id string) (*model.Store, error) {
	store, err := s.orgRepo.GetStoreByID(ctx, id)
	if err != nil {
		return nil, model.ErrInternal("Gagal mengambil data cabang")
	}
	if store == nil {
		return nil, model.ErrNotFound("Cabang tidak ditemukan")
	}
	return store, nil
}

func (s *OrgService) CreateStore(ctx context.Context, store *model.Store) error {
	if store.Code == "" || store.Name == "" {
		return model.ErrValidation("Kode dan nama cabang wajib diisi")
	}
	store.IsActive = true
	return s.orgRepo.CreateStore(ctx, store)
}

func (s *OrgService) ListWarehouses(ctx context.Context, storeID string) ([]model.Warehouse, error) {
	return s.orgRepo.ListWarehouses(ctx, storeID)
}

func (s *OrgService) ListTerminals(ctx context.Context, storeID string) ([]model.Terminal, error) {
	return s.orgRepo.ListTerminals(ctx, storeID)
}
