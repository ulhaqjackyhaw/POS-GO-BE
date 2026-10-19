-- Extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Drop legacy unversioned tables if they exist
DROP TABLE IF EXISTS transaction_items CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;

-- 1. Organizations & Branches
CREATE TABLE IF NOT EXISTS stores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    address TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS terminals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. RBAC & Users
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    module VARCHAR(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_stores (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, store_id)
);

-- 3. Product Catalog & Pricing per Branch
CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- If old products table exists, drop it to ensure clean schema
DROP TABLE IF EXISTS products CASCADE;

CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku VARCHAR(100) UNIQUE NOT NULL,
    barcode VARCHAR(100),
    name VARCHAR(255) NOT NULL,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    unit VARCHAR(50) NOT NULL DEFAULT 'pcs',
    base_price BIGINT NOT NULL CHECK (base_price >= 0),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS store_product_prices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    price BIGINT NOT NULL CHECK (price >= 0),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(store_id, product_id)
);

-- 4. Inventory Balances & Ledgers (Direct Store-Level Stock Management)
CREATE TABLE IF NOT EXISTS inventory_balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(store_id, product_id),
    CONSTRAINT chk_stock_non_negative CHECK (quantity >= 0)
);

CREATE TABLE IF NOT EXISTS inventory_ledgers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    mutation_type VARCHAR(50) NOT NULL,
    reference_type VARCHAR(50),
    reference_id VARCHAR(100),
    quantity_change BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    reason TEXT,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5. Cashier Shifts & Cash Movements
CREATE TABLE IF NOT EXISTS cashier_shifts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    terminal_id UUID NOT NULL REFERENCES terminals(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    starting_cash BIGINT NOT NULL DEFAULT 0,
    ending_cash_actual BIGINT,
    ending_cash_expected BIGINT,
    cash_difference BIGINT,
    status VARCHAR(50) NOT NULL DEFAULT 'OPEN',
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    notes TEXT
);

CREATE TABLE IF NOT EXISTS cash_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shift_id UUID NOT NULL REFERENCES cashier_shifts(id) ON DELETE CASCADE,
    movement_type VARCHAR(50) NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    category VARCHAR(100) NOT NULL,
    reason TEXT,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 6. Sales Transactions & Items
CREATE TABLE IF NOT EXISTS sales_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_number VARCHAR(100) UNIQUE NOT NULL,
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    terminal_id UUID REFERENCES terminals(id) ON DELETE SET NULL,
    shift_id UUID REFERENCES cashier_shifts(id) ON DELETE SET NULL,
    cashier_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    customer_name VARCHAR(255),
    subtotal BIGINT NOT NULL DEFAULT 0,
    discount_amount BIGINT NOT NULL DEFAULT 0,
    tax_amount BIGINT NOT NULL DEFAULT 0,
    total_amount BIGINT NOT NULL DEFAULT 0,
    payment_method VARCHAR(50) NOT NULL DEFAULT 'CASH',
    paid_amount BIGINT NOT NULL DEFAULT 0,
    change_amount BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'COMPLETED',
    payload_hash VARCHAR(64),
    offline_created_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sales_transaction_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES sales_transactions(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    product_name VARCHAR(255) NOT NULL,
    unit_price BIGINT NOT NULL,
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    subtotal BIGINT NOT NULL,
    discount_amount BIGINT NOT NULL DEFAULT 0,
    tax_amount BIGINT NOT NULL DEFAULT 0,
    total BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 7. Approval Requests
CREATE TABLE IF NOT EXISTS approval_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID REFERENCES stores(id) ON DELETE CASCADE,
    action_type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    requested_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    approved_by UUID REFERENCES users(id) ON DELETE RESTRICT,
    payload JSONB NOT NULL DEFAULT '{}',
    reason TEXT NOT NULL,
    rejection_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 8. Stock Transfers (Inter-Store / Antarcabang)
CREATE TABLE IF NOT EXISTS stock_transfers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transfer_number VARCHAR(100) UNIQUE NOT NULL,
    from_store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    to_store_id UUID NOT NULL REFERENCES stores(id) ON DELETE RESTRICT,
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT',
    requested_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS stock_transfer_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transfer_id UUID NOT NULL REFERENCES stock_transfers(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    received_quantity BIGINT NOT NULL DEFAULT 0
);

-- 9. Audit Logs
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID REFERENCES stores(id) ON DELETE SET NULL,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    entity VARCHAR(100) NOT NULL,
    entity_id VARCHAR(100),
    details JSONB NOT NULL DEFAULT '{}',
    ip_address VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_products_sku ON products(sku);
CREATE INDEX IF NOT EXISTS idx_products_barcode ON products(barcode);
CREATE INDEX IF NOT EXISTS idx_inventory_balances_store ON inventory_balances(store_id);
CREATE INDEX IF NOT EXISTS idx_inventory_ledgers_store ON inventory_ledgers(store_id);
CREATE INDEX IF NOT EXISTS idx_inventory_ledgers_product ON inventory_ledgers(product_id);
CREATE INDEX IF NOT EXISTS idx_sales_transactions_store ON sales_transactions(store_id);
CREATE INDEX IF NOT EXISTS idx_sales_transactions_created_at ON sales_transactions(created_at);
CREATE INDEX IF NOT EXISTS idx_sales_transactions_idempotency ON sales_transactions(idempotency_key);
CREATE INDEX IF NOT EXISTS idx_cashier_shifts_user ON cashier_shifts(user_id, status);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
