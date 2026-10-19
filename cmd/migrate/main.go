package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"go-backend/internal/config"
	"go-backend/internal/platform/database"
)

func main() {
	resetFlag := flag.Bool("reset", true, "Reset entire database schema and re-run all migrations & seeds")
	flag.Parse()

	log.Println("🔄 Memulai proses migrasi database...")

	// 1. Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Gagal membaca config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 2. Connect
	pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ Gagal terhubung ke PostgreSQL: %v", err)
	}
	defer pool.Close()
	log.Println("✅ Terhubung ke PostgreSQL.")

	// 3. Reset if requested
	if *resetFlag {
		log.Println("⚠️  Mereset seluruh tabel database (DROP & RECREATE public schema)...")

		resetQuery := `
			DROP SCHEMA public CASCADE;
			CREATE SCHEMA public;
			GRANT ALL ON SCHEMA public TO postgres;
			GRANT ALL ON SCHEMA public TO public;
		`
		if _, err := pool.Exec(ctx, resetQuery); err != nil {
			log.Fatalf("❌ Gagal mereset skema database: %v", err)
		}
		log.Println("✅ Skema database berhasil dibersihkan.")
	}

	// 4. Run up migrations
	migrationsDir := "migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		migrationsDir = filepath.Join("..", "..", "migrations")
	}

	log.Printf("📦 Menjalankan migrasi SQL dari folder: %s...", migrationsDir)
	if err := database.RunMigrations(ctx, pool, migrationsDir); err != nil {
		log.Fatalf("❌ Migrasi gagal: %v", err)
	}
	log.Println("✅ Semua file migrasi (*.up.sql) berhasil diaplikasikan.")

	// 5. Seed initial data
	log.Println("🌱 Mengisi data awal (seeding master data)...")
	if err := database.SeedInitialData(ctx, pool); err != nil {
		log.Fatalf("❌ Seeding gagal: %v", err)
	}
	log.Println("✅ Master data (Roles, Permissions, Cabang, Gudang, Terminal, Kasir, Produk, Stok) berhasil dibuat.")

	fmt.Println("\n=======================================================")
	fmt.Println("🎉 Database BERHASIL di-migrate ulang dari nol!")
	fmt.Println("=======================================================")
	fmt.Println("Akun Default Siap Pakai:")
	fmt.Println("  1. Owner / Super Admin : admin    | password123")
	fmt.Println("  2. Store Manager       : manager1 | password123")
	fmt.Println("  3. Kasir               : kasir1   | password123")
	fmt.Println("=======================================================")
}
