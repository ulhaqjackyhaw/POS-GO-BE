package service

import (
	"context"
	"time"

	"go-backend/internal/model"
	"go-backend/internal/repository"
)

type ReportService struct {
	reportRepo *repository.ReportRepository
}

func NewReportService(reportRepo *repository.ReportRepository) *ReportService {
	return &ReportService{reportRepo: reportRepo}
}

func (s *ReportService) GetSalesReport(ctx context.Context, storeID string, from, to time.Time) (*model.SalesSummaryReport, error) {
	if to.IsZero() {
		to = time.Now()
	}
	if from.IsZero() {
		from = to.AddDate(0, 0, -30) // default 30 days
	}
	return s.reportRepo.GetSalesSummary(ctx, storeID, from, to)
}

func (s *ReportService) GetTopProducts(ctx context.Context, storeID string, from, to time.Time, limit int) ([]model.TopProductItem, error) {
	if limit <= 0 {
		limit = 10
	}
	if to.IsZero() {
		to = time.Now()
	}
	if from.IsZero() {
		from = to.AddDate(0, 0, -30)
	}
	return s.reportRepo.GetTopProducts(ctx, storeID, from, to, limit)
}
