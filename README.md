# Go Fiber POS & Inventory Backend (Multi-Cabang)

Sistem Point of Sale (POS) dan manajemen inventori multi-cabang berkinerja tinggi berbasis Go Fiber dan PostgreSQL, dirancang sesuai spesifikasi pada `docs/prd.md` dan aturan teknis pada `docs/AGENT.md`.

---

## 1. Arsitektur & Tech Stack

- **Bahasa:** Go 1.24+ (teruji pada Go 1.27)
- **HTTP Framework:** `github.com/gofiber/fiber/v2`
- **Database:** PostgreSQL via `github.com/jackc/pgx/v5` & `pgxpool` (tanpa ORM / GORM)
- **Autentikasi & Otorisasi:** JWT (`golang-jwt/jwt/v5`), Password hashing (`golang.org/x/crypto/bcrypt`), Role-Based Access Control (RBAC) & Store Scoping
- **Kalkulasi Uang & Stok:** Tipe integer fixed-precision (`int64` dalam satuan Rupiah penuh), perhitungan pajak deterministik *tax-inclusive*, tanpa floating point
- **Integritas Data:** Migrasi database berversi, transaksi atomik ACID (`pgx.Tx`), deterministic row-level locking (`SELECT ... FOR UPDATE` dengan SKU terurut) untuk mencegah race condition dan deadlock, serta database check constraint `CHECK (quantity >= 0)`

---

## 2. Struktur Direktori Proyek

```text
.
├── cmd/
│   └── api/
│       └── main.go           # Entry point aplikasi utama
├── docs/
│   ├── AGENT.md              # Aturan teknis dan konvensi AI/Developer
│   └── prd.md                # Product Requirements Document
├── internal/
│   ├── app/                  # Bootstrap dan lifecycle aplikasi
│   ├── config/               # Loader konfigurasi environment variables
│   ├── handler/              # HTTP request/response handler
│   ├── middleware/           # Auth JWT, RBAC Guard, Store Scope, Request ID, Error Handler, Structured Logger
│   ├── model/                # Domain entities, DTOs, & AppError
│   ├── platform/
│   │   └── database/         # Pgxpool connection, migration runner, & initial seeder
│   ├── repository/           # Parameterized SQL database access layer
│   ├── route/                # Deklarasi router dan middleware chain
│   └── service/              # Aturan bisnis, kalkulasi harga, locking, idempotency, & approval
├── migrations/
│   ├── 000001_init_schema.up.sql
│   └── 000001_init_schema.down.sql
├── tests/
│   ├── api_contract_test.go   # Tes format error API dan kode HTTP
│   ├── business_rules_test.go # Unit test kalkulasi pajak, idempotency hash, self-approval
│   └── integration_test.go    # End-to-end integration test database PostgreSQL
├── .env.example
├── go.mod
├── go.sum
├── main.go                   # Root entry point
└── README.md
```

---

## 3. Akun Default & Master Data (Seeded)

Saat aplikasi pertama kali dijalankan, migrasi dan seeder otomatis mengisi data berikut:

| Username | Password | Role | Hak Akses |
|---|---|---|---|
| `admin` | `password123` | `owner` | Akses global seluruh cabang, konfigurasi, laporan, audit |
| `manager1` | `password123` | `store_manager` | Akses cabang Jakarta (`STORE-JKT-01`), review approval, void/refund, stok |
| `kasir1` | `password123` | `cashier` | Akses kasir cabang Jakarta (`STORE-JKT-01`), quote, checkout, shift |

### Data Cabang & Gudang Awal
- **Cabang:** `Cabang Utama - Jakarta` (Kode: `STORE-JKT-01`)
- **Gudang:** `Gudang Utama Jakarta` (Kode: `WH-JKT-01`)
- **Terminal:** `Terminal Kasir 1` (Kode: `POS-01`)

### Produk & Harga Khusus Cabang
- `SKU-KOPI-01` (Kopi Susu Gula Aren) — Base Price: Rp 18.000 — Stok: 100
- `SKU-ROTI-01` (Roti Bakar Coklat Keju) — Base Price: Rp 15.000 — Stok: 50
- `SKU-TEH-01` (Teh Melati Spesial) — Base Price: Rp 8.000, **Override Cabang: Rp 9.000** — Stok: 80

---

## 4. Cara Menjalankan Aplikasi

### A. Konfigurasi Database
Pastikan PostgreSQL berjalan di port 5432 dan database `pos_db` telah dibuat:
```bash
# Salin konfigurasi environment
cp .env.example .env
```
Isi konfigurasi pada file `.env`:
```env
APP_NAME="Go Fiber POS Backend"
APP_ENV="development"
APP_PORT="8080"
DATABASE_URL="postgres://postgres:1234@localhost:5432/pos_db?sslmode=disable"
JWT_SECRET="pos-super-secret-key-change-in-production-2026"
JWT_EXPIRATION_HOURS="24"
DEFAULT_TAX_RATE_BPS="1100" # 11% tax rate (basis points)
```

### B. Menjalankan Server
```bash
go run ./cmd/api
# atau
go run main.go
```
Aplikasi akan otomatis menerapkan migrasi schema `migrations/000001_init_schema.up.sql` dan melakukan seeding data master.

### C. Menjalankan Pengujian (Testing)
```bash
go test -v ./tests/...
```

---

## 5. Ringkasan Endpoint API

### Autentikasi & Profil
- `POST /api/v1/auth/login` — Login kasir / staf / admin
- `GET /api/v1/auth/me` — Info user aktif dan permission

### Cabang, Gudang, & Terminal
- `GET /api/v1/stores` — Daftar cabang
- `POST /api/v1/stores` — Buat cabang baru (`org:manage`)
- `GET /api/v1/stores/:store_id` — Detail cabang
- `GET /api/v1/stores/:store_id/warehouses` — Daftar gudang cabang
- `GET /api/v1/stores/:store_id/terminals` — Daftar terminal cabang

### Katalog & Harga Cabang
- `GET /api/v1/categories` — Daftar kategori produk
- `GET /api/v1/products?store_id=...&warehouse_id=...&search=...` — Katalog produk dengan resolver harga efektif cabang dan saldo stok
- `POST /api/v1/products` — Tambah produk baru (`catalog:write`)
- `POST /api/v1/stores/:store_id/products/:id/price` — Override harga khusus cabang (`price:override`)

### Inventori & Stok
- `GET /api/v1/inventory/stock` — Saldo stok produk terkini per gudang
- `POST /api/v1/inventory/adjust` — Penyesuaian stok dengan alasan wajib (`inventory:adjust`)
- `GET /api/v1/inventory/ledgers` — Histori mutasi stok (immutable audit trail)

### Kasir & Shift
- `GET /api/v1/shifts/active?store_id=...` — Cek shift kasir yang sedang aktif
- `GET /api/v1/shifts/:id` — Detail shift lengkap & rekonsiliasi kas
- `POST /api/v1/shifts/open` — Buka shift kasir dengan modal awal (`starting_cash` atau `opening_cash_minor`)
- `POST /api/v1/shifts/:id/movements` — Catat kas masuk / keluar (`IN`/`OUT` atau `cash_in`/`cash_out`)
- `POST /api/v1/shifts/:id/close` — Tutup shift dengan rekonsiliasi kas aktual vs ekspektasi

### POS Penjualan
- `POST /api/v1/pos/quote` — Kalkulasi subtotal, diskon, dan pajak *tax-inclusive*
- `POST /api/v1/pos/checkout` — Checkout online atomik dengan idempotency key, row lock, & snapshot harga (mendukung payload flat atau nested `payment: { method, received_minor }`)
- `GET /api/v1/pos/transactions` — Histori transaksi penjualan
- `GET /api/v1/pos/transactions/:id` — Detail transaksi penjualan
- `POST /api/v1/pos/transactions/:id/void` — Pembatalan transaksi (kompensasi stok di ledger)
- `POST /api/v1/pos/transactions/:id/refund` — Pengembalian dana transaksi

### Approval Tindakan Kritis
- `POST /api/v1/approvals` — Ajukan approval (diskon khusus, penyesuaian stok, void)
- `GET /api/v1/approvals` — Daftar permintaan approval
- `GET /api/v1/approvals/:id` — Detail satu permintaan approval
- `POST /api/v1/approvals/:id/review` — Approve/Reject (Dicegah *self-approval*)

### Sinkronisasi Offline
- `POST /api/v1/sync/batch` — Sinkronisasi batch transaksi tunai offline dengan pelaporan konflik eksplisit

### Laporan & Health
- `GET /api/v1/reports/sales` — Ringkasan omzet bruto, diskon, pajak, dan penjualan bersih
- `GET /api/v1/reports/top-products` — Peringkat produk terlaris
- `GET /health` — Liveness & readiness probe (CORS enabled)

---

## 6. Format Respons Error Standar

Semua error API mematuhi kontrak JSON seragam pada `docs/AGENT.md`:
```json
{
  "error": "Stok tidak mencukupi untuk produk: Kopi Susu Gula Aren",
  "code": "INSUFFICIENT_STOCK"
}
```
Kode error umum: `VALIDATION_ERROR` (400), `INSUFFICIENT_STOCK` (400), `SHIFT_NOT_OPEN` (400), `CANNOT_APPROVE_OWN_REQUEST` (403), `UNAUTHENTICATED` (401), `FORBIDDEN` (403), `NOT_FOUND` (404), `IDEMPOTENCY_CONFLICT` (409), `SYNC_CONFLICT` (409), `INTERNAL_SERVER_ERROR` (500).
#   P O S - G O - B E  
 