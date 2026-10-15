package route

import (
	"go-backend/internal/handler"
	"go-backend/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

type RouterConfig struct {
	App       *fiber.App
	JWTSecret string
	AuthH     *handler.AuthHandler
	OrgH      *handler.OrgHandler
	CatalogH  *handler.CatalogHandler
	InvH      *handler.InventoryHandler
	ShiftH    *handler.ShiftHandler
	POSH      *handler.POSHandler
	ApprovalH *handler.ApprovalHandler
	OfflineH  *handler.OfflineSyncHandler
	ReportH   *handler.ReportHandler
	HealthH   *handler.HealthHandler
}

func SetupRoutes(cfg RouterConfig) {
	app := cfg.App

	// Health check (Public)
	app.Get("/health", cfg.HealthH.Health)

	// API Group v1
	v1 := app.Group("/api/v1")

	// Auth routes (Public)
	v1.Post("/auth/login", cfg.AuthH.Login)

	// Protected routes
	protected := v1.Group("", middleware.AuthRequired(cfg.JWTSecret))

	// User info
	protected.Get("/auth/me", cfg.AuthH.Me)

	// Organizations & Stores
	stores := protected.Group("/stores")
	stores.Get("", cfg.OrgH.ListStores)
	stores.Post("", middleware.RequirePermission("org:manage"), cfg.OrgH.CreateStore)
	stores.Get("/:store_id", middleware.RequireStoreAccess(), cfg.OrgH.GetStore)
	stores.Get("/:store_id/warehouses", middleware.RequireStoreAccess(), cfg.OrgH.ListWarehouses)
	stores.Get("/:store_id/terminals", middleware.RequireStoreAccess(), cfg.OrgH.ListTerminals)

	// Categories
	protected.Get("/categories", middleware.RequirePermission("catalog:read"), cfg.CatalogH.ListCategories)

	// Product Catalog & Pricing
	products := protected.Group("/products")
	products.Get("", middleware.RequirePermission("catalog:read"), cfg.CatalogH.ListProducts)
	products.Post("", middleware.RequirePermission("catalog:write"), cfg.CatalogH.CreateProduct)
	products.Get("/:id", middleware.RequirePermission("catalog:read"), cfg.CatalogH.GetProduct)

	// Store price override
	protected.Post("/stores/:store_id/products/:id/price",
		middleware.RequirePermission("price:override"),
		middleware.RequireStoreAccess(),
		cfg.CatalogH.SetBranchPrice,
	)

	// Inventory
	inv := protected.Group("/inventory")
	inv.Get("/stock", middleware.RequirePermission("inventory:read"), cfg.InvH.ListStock)
	inv.Post("/adjust", middleware.RequirePermission("inventory:adjust"), cfg.InvH.AdjustStock)
	inv.Get("/ledgers", middleware.RequirePermission("inventory:read"), cfg.InvH.ListLedgers)

	// Cashier Shifts & Cash Movements
	shifts := protected.Group("/shifts")
	shifts.Get("/active", cfg.ShiftH.GetActiveShift)
	shifts.Get("/:id", cfg.ShiftH.GetShiftDetail)
	shifts.Post("/open", middleware.RequirePermission("shift:open"), cfg.ShiftH.OpenShift)
	shifts.Post("/:id/close", middleware.RequirePermission("shift:close"), cfg.ShiftH.CloseShift)
	shifts.Post("/:id/movements", middleware.RequirePermission("shift:movement"), cfg.ShiftH.CreateCashMovement)

	// POS Transactions
	pos := protected.Group("/pos")
	pos.Post("/quote", middleware.RequirePermission("pos:quote"), cfg.POSH.GenerateQuote)
	pos.Post("/checkout",
		middleware.RequirePermission("pos:checkout"),
		middleware.RequireStoreAccess(),
		cfg.POSH.Checkout,
	)
	pos.Get("/transactions", cfg.POSH.ListTransactions)
	pos.Get("/transactions/:id", cfg.POSH.GetTransaction)
	pos.Post("/transactions/:id/void", middleware.RequirePermission("pos:void"), cfg.POSH.VoidTransaction)
	pos.Post("/transactions/:id/refund", middleware.RequirePermission("pos:refund"), cfg.POSH.RefundTransaction)

	// Critical Actions Approvals
	approvals := protected.Group("/approvals")
	approvals.Post("", middleware.RequirePermission("approval:request"), cfg.ApprovalH.CreateRequest)
	approvals.Get("", cfg.ApprovalH.ListRequests)
	approvals.Get("/:id", cfg.ApprovalH.GetApproval)
	approvals.Post("/:id/review", middleware.RequirePermission("approval:review"), cfg.ApprovalH.ReviewRequest)

	// Offline Sync
	syncGroup := protected.Group("/sync")
	syncGroup.Post("/batch", middleware.RequirePermission("pos:checkout"), cfg.OfflineH.SyncBatch)

	// Operational Reports
	reports := protected.Group("/reports", middleware.RequirePermission("report:read"))
	reports.Get("/sales", cfg.ReportH.GetSalesReport)
	reports.Get("/top-products", cfg.ReportH.GetTopProducts)
}
