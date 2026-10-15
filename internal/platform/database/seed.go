package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func SeedInitialData(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Roles
	roles := []struct {
		code, name, desc string
	}{
		{"owner", "Owner / Business Admin", "Akses penuh seluruh cabang dan konfigurasi"},
		{"store_manager", "Store Manager", "Operasional cabang, persetujuan dan stok"},
		{"supervisor", "Supervisor", "Supervisi kasir dan persetujuan operasional"},
		{"cashier", "Cashier", "Kasir penjualan dan penutupan shift"},
		{"inventory_staff", "Inventory Staff", "Penerimaan, penyesuaian, dan transfer stok"},
		{"auditor", "Auditor / Read-only", "Laporan dan audit tanpa hak ubah"},
	}

	for _, r := range roles {
		_, err := tx.Exec(ctx, `
			INSERT INTO roles (code, name, description)
			VALUES ($1, $2, $3)
			ON CONFLICT (code) DO NOTHING
		`, r.code, r.name, r.desc)
		if err != nil {
			return fmt.Errorf("seeding role %s failed: %w", r.code, err)
		}
	}

	// 2. Permissions
	permissions := []struct {
		code, desc, module string
	}{
		{"org:manage", "Kelola cabang, gudang, terminal", "organization"},
		{"catalog:read", "Melihat katalog produk", "catalog"},
		{"catalog:write", "Membuat dan mengubah katalog produk", "catalog"},
		{"price:override", "Mengatur harga khusus cabang", "catalog"},
		{"inventory:read", "Melihat saldo dan mutasi stok", "inventory"},
		{"inventory:adjust", "Melakukan penyesuaian stok", "inventory"},
		{"inventory:transfer", "Melakukan transfer stok antarcabang", "inventory"},
		{"pos:quote", "Membuat quote transaksi", "pos"},
		{"pos:checkout", "Melakukan checkout penjualan", "pos"},
		{"pos:void", "Melakukan pembatalan transaksi", "pos"},
		{"pos:refund", "Melakukan pengembalian dana transaksi", "pos"},
		{"shift:open", "Membuka shift kasir", "shift"},
		{"shift:close", "Menutup shift kasir", "shift"},
		{"shift:movement", "Mencatat kas masuk/keluar", "shift"},
		{"approval:request", "Mengajukan approval tindakan kritis", "approval"},
		{"approval:review", "Menyetujui atau menolak approval", "approval"},
		{"report:read", "Melihat laporan penjualan dan inventori", "report"},
		{"audit:read", "Melihat audit log aktivitas sistem", "audit"},
	}

	for _, p := range permissions {
		_, err := tx.Exec(ctx, `
			INSERT INTO permissions (code, description, module)
			VALUES ($1, $2, $3)
			ON CONFLICT (code) DO NOTHING
		`, p.code, p.desc, p.module)
		if err != nil {
			return fmt.Errorf("seeding permission %s failed: %w", p.code, err)
		}
	}

	// 3. Assign all permissions to owner role
	_, err = tx.Exec(ctx, `
		INSERT INTO role_permissions (role_id, permission_id)
		SELECT r.id, p.id
		FROM roles r, permissions p
		WHERE r.code = 'owner'
		ON CONFLICT DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("assigning owner permissions failed: %w", err)
	}

	// Assign cashier permissions
	cashierPerms := []string{"catalog:read", "pos:quote", "pos:checkout", "shift:open", "shift:close", "shift:movement", "approval:request"}
	for _, cp := range cashierPerms {
		_, err = tx.Exec(ctx, `
			INSERT INTO role_permissions (role_id, permission_id)
			SELECT r.id, p.id
			FROM roles r, permissions p
			WHERE r.code = 'cashier' AND p.code = $1
			ON CONFLICT DO NOTHING
		`, cp)
		if err != nil {
			return fmt.Errorf("assigning cashier permission %s failed: %w", cp, err)
		}
	}

	// Assign store_manager permissions
	managerPerms := []string{
		"catalog:read", "catalog:write", "price:override", "inventory:read",
		"inventory:adjust", "inventory:transfer", "pos:quote", "pos:checkout",
		"pos:void", "pos:refund", "shift:open", "shift:close", "shift:movement",
		"approval:request", "approval:review", "report:read", "audit:read",
	}
	for _, mp := range managerPerms {
		_, err = tx.Exec(ctx, `
			INSERT INTO role_permissions (role_id, permission_id)
			SELECT r.id, p.id
			FROM roles r, permissions p
			WHERE r.code = 'store_manager' AND p.code = $1
			ON CONFLICT DO NOTHING
		`, mp)
		if err != nil {
			return fmt.Errorf("assigning manager permission %s failed: %w", mp, err)
		}
	}

	// 4. Default Store, Warehouse, Terminal
	var storeID string
	err = tx.QueryRow(ctx, `
		INSERT INTO stores (code, name, address, is_active)
		VALUES ('STORE-JKT-01', 'Cabang Utama - Jakarta', 'Jl. Sudirman Kav. 1, Jakarta Pusat', true)
		ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`).Scan(&storeID)
	if err != nil {
		return fmt.Errorf("seeding store failed: %w", err)
	}

	var warehouseID string
	err = tx.QueryRow(ctx, `
		INSERT INTO warehouses (store_id, code, name, is_active)
		VALUES ($1, 'WH-JKT-01', 'Gudang Utama Jakarta', true)
		ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, storeID).Scan(&warehouseID)
	if err != nil {
		return fmt.Errorf("seeding warehouse failed: %w", err)
	}

	var terminalID string
	err = tx.QueryRow(ctx, `
		INSERT INTO terminals (store_id, code, name, is_active)
		VALUES ($1, 'POS-01', 'Terminal Kasir 1', true)
		ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, storeID).Scan(&terminalID)
	if err != nil {
		return fmt.Errorf("seeding terminal failed: %w", err)
	}

	// 5. Default Users
	defaultPassword := "password123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("password hashing failed: %w", err)
	}

	// Admin (Owner)
	var adminID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, full_name, role_id, is_active)
		SELECT 'admin', 'admin@pos.local', $1, 'Super Administrator', r.id, true
		FROM roles r WHERE r.code = 'owner'
		ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash
		RETURNING id
	`, string(hashedPassword)).Scan(&adminID)
	if err != nil {
		return fmt.Errorf("seeding admin user failed: %w", err)
	}

	// Cashier User
	var cashierID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, full_name, role_id, is_active)
		SELECT 'kasir1', 'kasir1@pos.local', $1, 'Budi Kasir', r.id, true
		FROM roles r WHERE r.code = 'cashier'
		ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash
		RETURNING id
	`, string(hashedPassword)).Scan(&cashierID)
	if err != nil {
		return fmt.Errorf("seeding cashier user failed: %w", err)
	}

	// Store Manager User
	var managerID string
	err = tx.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, full_name, role_id, is_active)
		SELECT 'manager1', 'manager1@pos.local', $1, 'Siti Manager', r.id, true
		FROM roles r WHERE r.code = 'store_manager'
		ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash
		RETURNING id
	`, string(hashedPassword)).Scan(&managerID)
	if err != nil {
		return fmt.Errorf("seeding manager user failed: %w", err)
	}

	// Assign cashier and manager to store
	_, _ = tx.Exec(ctx, `INSERT INTO user_stores (user_id, store_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, cashierID, storeID)
	_, _ = tx.Exec(ctx, `INSERT INTO user_stores (user_id, store_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, managerID, storeID)

	// 6. Categories & Products
	var catID string
	err = tx.QueryRow(ctx, `
		INSERT INTO categories (name)
		VALUES ('Minuman & Makanan')
		RETURNING id
	`).Scan(&catID)
	if err != nil {
		// Category might already exist
		_ = tx.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&catID)
	}

	sampleProducts := []struct {
		sku, barcode, name, unit string
		basePrice                int64
		branchPrice              int64
		initialStock             int64
	}{
		{"SKU-KOPI-01", "89912345001", "Kopi Susu Gula Aren", "cup", 18000, 18000, 100},
		{"SKU-ROTI-01", "89912345002", "Roti Bakar Coklat Keju", "porsi", 15000, 15000, 50},
		{"SKU-TEH-01", "89912345003", "Teh Melati Spesial", "cup", 8000, 9000, 80}, // override to 9000 at store
	}

	for _, sp := range sampleProducts {
		var productID string
		err := tx.QueryRow(ctx, `
			INSERT INTO products (sku, barcode, name, category_id, unit, base_price, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, true)
			ON CONFLICT (sku) DO UPDATE SET name = EXCLUDED.name, base_price = EXCLUDED.base_price
			RETURNING id
		`, sp.sku, sp.barcode, sp.name, catID, sp.unit, sp.basePrice).Scan(&productID)
		if err != nil {
			return fmt.Errorf("seeding product %s failed: %w", sp.sku, err)
		}

		// Store price override
		if sp.branchPrice != sp.basePrice {
			_, err = tx.Exec(ctx, `
				INSERT INTO store_product_prices (store_id, product_id, price, is_active)
				VALUES ($1, $2, $3, true)
				ON CONFLICT (store_id, product_id) DO UPDATE SET price = EXCLUDED.price
			`, storeID, productID, sp.branchPrice)
			if err != nil {
				return fmt.Errorf("seeding store product price failed: %w", err)
			}
		}

		// Inventory balance
		var balanceExists bool
		_ = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM inventory_balances WHERE warehouse_id = $1 AND product_id = $2)`, warehouseID, productID).Scan(&balanceExists)
		if !balanceExists {
			_, err = tx.Exec(ctx, `
				INSERT INTO inventory_balances (warehouse_id, product_id, quantity)
				VALUES ($1, $2, $3)
			`, warehouseID, productID, sp.initialStock)
			if err != nil {
				return fmt.Errorf("seeding stock balance failed: %w", err)
			}

			// Ledger entry for initial stock
			_, err = tx.Exec(ctx, `
				INSERT INTO inventory_ledgers (
					warehouse_id, product_id, mutation_type, reference_type, reference_id,
					quantity_change, balance_after, reason, created_by
				) VALUES (
					$1, $2, 'RECEIVE', 'INITIAL_STOCK', 'INIT-001',
					$3, $3, 'Initial inventory setup', $4
				)
			`, warehouseID, productID, sp.initialStock, adminID)
			if err != nil {
				return fmt.Errorf("seeding stock ledger failed: %w", err)
			}
		}
	}

	return tx.Commit(ctx)
}
