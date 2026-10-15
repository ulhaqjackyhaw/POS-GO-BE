package service

import (
	"context"

	"go-backend/internal/model"
	"go-backend/internal/repository"
)

type CatalogService struct {
	catalogRepo *repository.CatalogRepository
	orgRepo     *repository.OrgRepository
}

func NewCatalogService(catalogRepo *repository.CatalogRepository, orgRepo *repository.OrgRepository) *CatalogService {
	return &CatalogService{catalogRepo: catalogRepo, orgRepo: orgRepo}
}

func (s *CatalogService) ListProducts(ctx context.Context, storeID, warehouseID, search string) ([]model.ProductWithPrice, error) {
	// If warehouseID is empty, try to get default warehouse of store
	if warehouseID == "" && storeID != "" {
		whs, err := s.orgRepo.ListWarehouses(ctx, storeID)
		if err == nil && len(whs) > 0 {
			warehouseID = whs[0].ID
		}
	}
	return s.catalogRepo.ListProducts(ctx, storeID, warehouseID, search)
}

func (s *CatalogService) GetProduct(ctx context.Context, productID, storeID, warehouseID string) (*model.ProductWithPrice, error) {
	p, err := s.catalogRepo.GetProductWithEffectivePrice(ctx, productID, storeID, warehouseID)
	if err != nil {
		return nil, model.ErrInternal("Gagal memuat produk")
	}
	if p == nil {
		return nil, model.ErrNotFound("Produk tidak ditemukan")
	}
	return p, nil
}

func (s *CatalogService) CreateProduct(ctx context.Context, req model.CreateProductRequest) (*model.Product, error) {
	if req.SKU == "" || req.Name == "" || req.BasePrice < 0 {
		return nil, model.ErrValidation("SKU, nama produk, dan harga dasar (>= 0) wajib diisi")
	}

	p := &model.Product{
		SKU:        req.SKU,
		Barcode:    req.Barcode,
		Name:       req.Name,
		CategoryID: req.CategoryID,
		Unit:       req.Unit,
		BasePrice:  req.BasePrice,
		IsActive:   true,
	}
	if p.Unit == "" {
		p.Unit = "pcs"
	}

	if err := s.catalogRepo.CreateProduct(ctx, p); err != nil {
		return nil, model.ErrInternal("Gagal membuat produk (pastikan SKU unik)")
	}
	return p, nil
}

func (s *CatalogService) SetStorePrice(ctx context.Context, storeID, productID string, price int64) error {
	if price < 0 {
		return model.ErrValidation("Harga tidak boleh negatif")
	}
	// Verify product exists
	p, err := s.catalogRepo.GetProductByID(ctx, productID)
	if err != nil || p == nil {
		return model.ErrNotFound("Produk tidak ditemukan")
	}

	return s.catalogRepo.SetStorePrice(ctx, storeID, productID, price)
}

func (s *CatalogService) ListCategories(ctx context.Context) ([]model.Category, error) {
	return s.catalogRepo.ListCategories(ctx)
}
