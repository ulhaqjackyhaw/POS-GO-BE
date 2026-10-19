package app

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-backend/internal/config"
	"go-backend/internal/handler"
	"go-backend/internal/middleware"
	"go-backend/internal/platform/database"
	"go-backend/internal/repository"
	"go-backend/internal/route"
	"go-backend/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func Run() {
	log.Println("Starting Go Fiber POS Backend...")

	// 1. Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Connect to PostgreSQL via pgxpool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to PostgreSQL successfully.")

	// 3. Run Migrations
	migrationCtx, migCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer migCancel()

	if err := database.RunMigrations(migrationCtx, pool, "migrations"); err != nil {
		log.Fatalf("Database migrations failed: %v", err)
	}
	log.Println("Database migrations applied successfully.")

	// 4. Seed Initial Master Data
	seedCtx, seedCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer seedCancel()

	if err := database.SeedInitialData(seedCtx, pool); err != nil {
		log.Fatalf("Database seeding failed: %v", err)
	}
	log.Println("Initial master data verified/seeded successfully.")

	// 5. Initialize Repositories
	userRepo := repository.NewUserRepository(pool)
	orgRepo := repository.NewOrgRepository(pool)
	catalogRepo := repository.NewCatalogRepository(pool)
	invRepo := repository.NewInventoryRepository(pool)
	shiftRepo := repository.NewShiftRepository(pool)
	txRepo := repository.NewTransactionRepository(pool)
	approvalRepo := repository.NewApprovalRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)
	reportRepo := repository.NewReportRepository(pool)

	// 6. Initialize Services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpirationHours)
	orgService := service.NewOrgService(orgRepo)
	catalogService := service.NewCatalogService(catalogRepo, orgRepo)
	invService := service.NewInventoryService(invRepo, auditRepo)
	shiftService := service.NewShiftService(shiftRepo, auditRepo)
	posService := service.NewPOSService(txRepo, invRepo, catalogRepo, shiftRepo, auditRepo, cfg.DefaultTaxRateBps)
	approvalService := service.NewApprovalService(approvalRepo, auditRepo)
	offlineSyncService := service.NewOfflineSyncService(posService, txRepo, auditRepo)
	reportService := service.NewReportService(reportRepo)
	userService := service.NewUserService(userRepo, orgRepo)

	// 7. Initialize Handlers
	authH := handler.NewAuthHandler(authService)
	orgH := handler.NewOrgHandler(orgService)
	catalogH := handler.NewCatalogHandler(catalogService)
	invH := handler.NewInventoryHandler(invService)
	shiftH := handler.NewShiftHandler(shiftService)
	posH := handler.NewPOSHandler(posService)
	approvalH := handler.NewApprovalHandler(approvalService)
	offlineH := handler.NewOfflineSyncHandler(offlineSyncService)
	reportH := handler.NewReportHandler(reportService)
	healthH := handler.NewHealthHandler(pool)
	userH := handler.NewUserHandler(userService)

	// 8. Initialize Fiber App with strict AGENT.md Error Handler
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ErrorHandler: middleware.CustomErrorHandler,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Store-ID, Idempotency-Key, X-Request-ID",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	}))
	app.Use(recover.New())
	app.Use(middleware.RequestID())
	app.Use(middleware.StructuredLogger())

	// 9. Register Routes
	route.SetupRoutes(route.RouterConfig{
		App:       app,
		JWTSecret: cfg.JWTSecret,
		AuthH:     authH,
		OrgH:      orgH,
		CatalogH:  catalogH,
		InvH:      invH,
		ShiftH:    shiftH,
		POSH:      posH,
		ApprovalH: approvalH,
		OfflineH:  offlineH,
		ReportH:   reportH,
		HealthH:   healthH,
		UserH:     userH,
	})

	// 10. Start Server with Graceful Shutdown
	go func() {
		addr := ":" + cfg.AppPort
		log.Printf("Server listening on http://localhost%s", addr)
		if err := app.Listen(addr); err != nil {
			log.Printf("Fiber server stopped: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("Server forced shutdown: %v", err)
	}
	log.Println("Server exiting.")
}
