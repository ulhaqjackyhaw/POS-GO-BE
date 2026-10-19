package service

import (
	"context"
	"errors"
	"strings"

	"go-backend/internal/model"
	"go-backend/internal/repository"

	"github.com/jackc/pgx/v5"
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

func (s *OrgService) GetWarehouse(ctx context.Context, id string) (*model.Warehouse, error) {
	if id == "" {
		return nil, model.ErrValidation("ID gudang tidak boleh kosong")
	}
	w, err := s.orgRepo.GetWarehouseByID(ctx, id)
	if err != nil {
		return nil, model.ErrInternal("Gagal mengambil data gudang")
	}
	if w == nil {
		return nil, model.ErrNotFound("Gudang tidak ditemukan")
	}
	return w, nil
}

func (s *OrgService) CreateWarehouse(ctx context.Context, w *model.Warehouse) error {
	if w.StoreID == "" || w.Code == "" || w.Name == "" {
		return model.ErrValidation("Cabang, kode gudang, dan nama gudang wajib diisi")
	}
	// Verify store exists
	store, err := s.orgRepo.GetStoreByID(ctx, w.StoreID)
	if err != nil {
		return model.ErrInternal("Gagal memvalidasi cabang")
	}
	if store == nil {
		return model.ErrNotFound("Cabang yang dipilih tidak ditemukan")
	}
	w.IsActive = true
	return s.orgRepo.CreateWarehouse(ctx, w)
}

func (s *OrgService) UpdateWarehouse(ctx context.Context, id string, w *model.Warehouse) error {
	if id == "" {
		return model.ErrValidation("ID gudang tidak valid")
	}
	if w.Code == "" || w.Name == "" {
		return model.ErrValidation("Kode dan nama gudang wajib diisi")
	}
	existing, err := s.orgRepo.GetWarehouseByID(ctx, id)
	if err != nil {
		return model.ErrInternal("Gagal mengambil data gudang")
	}
	if existing == nil {
		return model.ErrNotFound("Gudang tidak ditemukan")
	}

	w.ID = id
	if w.StoreID == "" {
		w.StoreID = existing.StoreID
	} else if w.StoreID != existing.StoreID {
		// verify store exists
		store, err := s.orgRepo.GetStoreByID(ctx, w.StoreID)
		if err != nil || store == nil {
			return model.ErrNotFound("Cabang yang dipilih tidak ditemukan")
		}
	}

	return s.orgRepo.UpdateWarehouse(ctx, w)
}

func (s *OrgService) DeleteWarehouse(ctx context.Context, id string) error {
	if id == "" {
		return model.ErrValidation("ID gudang tidak valid")
	}
	err := s.orgRepo.DeleteWarehouse(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrNotFound("Gudang tidak ditemukan")
		}
		return model.ErrValidation(err.Error())
	}
	return nil
}

func (s *OrgService) ListTerminals(ctx context.Context, storeID string) ([]model.Terminal, error) {
	return s.orgRepo.ListTerminals(ctx, storeID)
}

func (s *OrgService) GetTerminal(ctx context.Context, id string) (*model.Terminal, error) {
	t, err := s.orgRepo.GetTerminalByID(ctx, id)
	if err != nil {
		return nil, model.ErrInternal("Gagal memuat terminal POS")
	}
	if t == nil {
		return nil, model.ErrNotFound("Terminal POS tidak ditemukan")
	}
	return t, nil
}

func (s *OrgService) CreateTerminal(ctx context.Context, t *model.Terminal) error {
	if t.StoreID == "" {
		return model.ErrValidation("Cabang toko (store_id) wajib ditentukan")
	}
	if t.Code == "" {
		return model.ErrValidation("Kode terminal wajib diisi")
	}
	if t.Name == "" {
		return model.ErrValidation("Nama terminal wajib diisi")
	}

	store, err := s.orgRepo.GetStoreByID(ctx, t.StoreID)
	if err != nil || store == nil {
		return model.ErrValidation("Cabang toko yang dipilih tidak valid")
	}

	t.IsActive = true
	if err := s.orgRepo.CreateTerminal(ctx, t); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			return model.ErrValidation("Kode terminal sudah digunakan. Gunakan kode lain.")
		}
		return model.ErrInternal("Gagal menambahkan terminal POS")
	}
	return nil
}

func (s *OrgService) UpdateTerminal(ctx context.Context, id string, req model.UpdateTerminalRequest) (*model.Terminal, error) {
	t, err := s.orgRepo.UpdateTerminal(ctx, id, req.Name, req.IsActive, req.AssignedCashierID)
	if err != nil {
		return nil, model.ErrInternal("Gagal memperbarui terminal POS")
	}
	if t == nil {
		return nil, model.ErrNotFound("Terminal POS tidak ditemukan")
	}
	return t, nil
}

func (s *OrgService) DeleteTerminal(ctx context.Context, id string) error {
	t, err := s.orgRepo.GetTerminalByID(ctx, id)
	if err != nil || t == nil {
		return model.ErrNotFound("Terminal POS tidak ditemukan")
	}
	if err := s.orgRepo.DeleteTerminal(ctx, id); err != nil {
		return model.ErrInternal("Gagal menghapus terminal POS")
	}
	return nil
}
