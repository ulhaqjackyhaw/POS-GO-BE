package handler

import (
	"strconv"
	"time"

	"go-backend/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ReportHandler struct {
	reportService *service.ReportService
}

func NewReportHandler(reportService *service.ReportService) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

func (h *ReportHandler) GetSalesReport(c *fiber.Ctx) error {
	storeID := c.Query("store_id")

	var fromTime, toTime time.Time
	if fromStr := c.Query("from"); fromStr != "" {
		fromTime, _ = time.Parse("2006-01-02", fromStr)
	}
	if toStr := c.Query("to"); toStr != "" {
		toTime, _ = time.Parse("2006-01-02", toStr)
		toTime = toTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	report, err := h.reportService.GetSalesReport(c.Context(), storeID, fromTime, toTime)
	if err != nil {
		return err
	}
	return c.JSON(report)
}

func (h *ReportHandler) GetTopProducts(c *fiber.Ctx) error {
	storeID := c.Query("store_id")
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	var fromTime, toTime time.Time
	if fromStr := c.Query("from"); fromStr != "" {
		fromTime, _ = time.Parse("2006-01-02", fromStr)
	}
	if toStr := c.Query("to"); toStr != "" {
		toTime, _ = time.Parse("2006-01-02", toStr)
		toTime = toTime.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	top, err := h.reportService.GetTopProducts(c.Context(), storeID, fromTime, toTime, limit)
	if err != nil {
		return err
	}
	return c.JSON(top)
}
