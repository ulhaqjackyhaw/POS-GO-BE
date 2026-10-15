# 🛒 POS-GO-BE — Enterprise POS & Inventory Backend (Multi-Cabang)

> **Backend REST API Point of Sale (POS) & Manajemen Inventori Ritel Multi-Cabang** berkinerja tinggi, berorientasi integritas data finansial, dan siap mendukung operasional kasir *offline-first*. Dibangun menggunakan **Go 1.24+ (Fiber v2)** dan **PostgreSQL murni (`pgx/v5`)**.

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Framework](https://img.shields.io/badge/Framework-GoFiber_v2-00ACD7?style=flat)](https://gofiber.io)
[![Database](https://img.shields.io/badge/Database-PostgreSQL_pgx_v5-336791?style=flat&logo=postgresql)](https://github.com/jackc/pgx)
[![Architecture](https://img.shields.io/badge/Architecture-Modular_Layered-blueviolet)](#-arsitektur--struktur-folder)
[![Tests](https://img.shields.io/badge/Tests-Passing_100%25-success)](#-pengujian-testing)

---

## 📌 Tentang Proyek: Aplikasi Apa Ini & Untuk Apa?

Dalam bisnis ritel yang memiliki banyak cabang (multi-outlet), pemilik toko sering menghadapi kendala besar:
1. **Selisih Kas:** Kasir tidak disiplin mencatat kas masuk/keluar sehingga terjadi ketidakcocokan saldo saat tutup toko.
2. **Stok Negatif & Overselling:** Dua kasir menjual sisa 1 unit barang secara bersamaan, menyebabkan saldo stok minus di sistem.
3. **Harga Berbeda Antar-Cabang:** Harga sewa atau biaya logistik antar-kota berbeda, tetapi sistem tidak fleksibel mendukung harga khusus per cabang.
4. **Kecurangan (Fraud) Kasir:** Transaksi dibatalkan (void) atau di-diskon besar-besaran secara sepihak tanpa izin atasan.
5. **Koneksi Internet Kasir Mati:** Transaksi di toko terhenti total saat koneksi internet terputus.

### 🎯 Solusi yang Diberikan `POS-GO-BE`
`POS-GO-BE` hadir sebagai **pusat kebenaran (*single source of truth*) data operasional dan keuangan** ritel modern:
- Mengendalikan seluruh transaksi kasir secara **ACID dan atomik** tanpa celah duplikasi data.
- Menjamin saldo stok **tidak pernah negatif** melalui locking database dan constraint level DB.
- Menyediakan mekanisme **Approval Workflow** untuk mencegah kecurangan kasir.
- Menyediakan antrean **Offline Queue Synchronization** sehingga kasir tetap bisa melayani penjualan tunai meski jaringan toko terputus.
- Menghitung pajak dan uang secara **deterministik tanpa floating-point** (bebas dari *rounding error* sen/rupiah).

---

## ✨ Fitur-Fitur Utama (Key Features)

### 1. 🏢 Multi-Cabang & Hierarki Organisasi
- Pengelolaan multi-toko (**Store**), gudang penyimpanan (**Warehouse**), dan terminal kasir (**Terminal POS**).
- Pembatasan data per cabang: kasir cabang A tidak dapat melihat atau memanipulasi data cabang B.

### 2. 👥 RBAC (Role-Based Access Control) & Autentikasi
- Login aman berbasis token **JWT** dengan password terenkripsi **bcrypt**.
- Multi-role bawaan: `owner` (akses global), `store_manager`, `supervisor`, `cashier`, `inventory_staff`, dan `auditor`.
- Validasi izin (*permission-based guard*) di setiap endpoint API dan pengecekan cakupan cabang (*store-scoping* via header `X-Store-ID`).

### 3. 🏷️ Katalog Produk & Dynamic Branch Pricing
- Pencarian produk instan berdasarkan **Nama, SKU, atau Barcode**.
- **Harga Khusus Cabang (*Price Override*):** Setiap cabang dapat memiliki harga jual berbeda untuk produk yang sama, dengan fallback otomatis ke harga dasar (*base price*).
- Snapshot harga dan pajak saat transaksi agar perubahan harga di masa depan tidak merusak histori nota lama.

### 4. 💰 Siklus Shift Kasir & Rekonsiliasi Kas (Cash Management)
- Kasir wajib membuka shift dengan mencatat **modal awal (*starting cash*)**.
- Pencatatan pergerakan kas kecil non-penjualan (**Cash In / Cash Out**) dengan alasan wajib.
- **Tutup Shift Otomatis:** Sistem menghitung ekspektasi kas fisik (`Modal Awal + Penjualan Tunai + Kas Masuk - Kas Keluar`), membandingkannya dengan uang fisik kasir, dan mencatat selisih (*cash discrepancy*).

### 5. ⚡ POS Checkout Atomik & Anti-Double Submit (Idempotency)
- **Kalkulasi Pajak Tax-Inclusive:** Harga jual sudah termasuk PPN (dihitung presisi: `pajak = harga × r / (1 + r)`).
- **Idempotency Key:** Mencegah transaksi terproses ganda saat kasir melakukan *double-click* atau koneksi *retry*. Request ulang dengan key sama akan mengembalikan nota yang sama tanpa mengurangi stok 2 kali.
- **Deterministic Row-Level Locking (`FOR UPDATE`):** Mengunci baris stok produk dalam urutan deterministik saat checkout, menjamin tidak ada *race condition*, tidak ada deadlock, dan stok tidak pernah minus.

### 6. 📦 Inventori Real-Time & Immutable Ledger
- Saldo stok terpantau real-time per gudang/cabang.
- **Ledger Mutasi Stok Abadi (*Immutable*):** Setiap pergerakan barang (penerimaan, penjualan kasir, penyesuaian manual, void, refund) dicatat sebagai mutasi baru. Data mutasi tidak boleh diedit atau dihapus.
- Penyesuaian stok (*Stock Adjustment*) wajib menyertakan alasan dan identitas pengguna.

### 7. 🛡️ Kontrol Tindakan Kritis & Approval System
- Tindakan berisiko tinggi (void nota, refund uang, diskon khusus di atas batas) dapat mewajibkan **Approval Permintaan**.
- **Prinsip Pemisahan Tugas:** Pemohon dilarang menyetujui permintaannya sendiri (`CANNOT_APPROVE_OWN_REQUEST`). Review harus dilakukan oleh manajer/supervisor yang berbeda.

### 8. 🔄 Offline Queue Synchronization (Mode Kasir Offline)
- Kasir tetap dapat melayani penjualan **tunai (CASH)** saat offline.
- Transaksi disimpan di antrean lokal perangkat (IndexedDB) dan dikirim secara batch saat koneksi pulih (`POST /sync/batch`).
- Server memvalidasi batch secara idempoten dan mengembalikan status eksplisit per transaksi (`SYNCED` vs `SYNC_CONFLICT`) tanpa merusak data valid lainnya.

### 9. 🧾 Struk Penjualan, Void & Refund Kompensasi
- Struk siap cetak untuk printer thermal (58mm / 80mm).
- Pembatalan nota (*Void*) dan pengembalian uang (*Refund*) dilakukan via **transaksi kompensasi** yang mengembalikan stok ke gudang secara transparan tanpa menghapus jejak nota aslinya.

### 10. 📊 Laporan Analitik Bisnis & Health Probe
- Ringkasan omzet bruto, diskon, pajak, refund, dan penjualan bersih.
- Peringkat produk terlaris (*Top Products by Quantity & Sales*).
- Endpoint `/health` probe untuk monitoring status database dan server.

---

## 🛠️ Tech Stack & Keputusan Arsitektur

| Komponen | Teknologi | Alasan Pemilihan |
|---|---|---|
| **Bahasa** | Go 1.24+ | Kinerja tinggi, penggunaan memori sangat rendah, kompilasi cepat. |
| **HTTP Engine** | Fiber v2 (`github.com/gofiber/fiber/v2`) | Framework tercepat berbasis Fasthttp dengan middleware ecosystem yang lengkap. |
| **Database** | PostgreSQL | Standar industri untuk kepatuhan ACID, foreign key constraints, dan locking transactional. |
| **Driver DB** | `pgx/v5` & `pgxpool` | Koneksi PostgreSQL murni tercepat tanpa overhead ORM (menghindari GORM demi kontrol query & locking ketat). |
| **Format Uang** | Integer Minor Units (`int64` Rupiah bulat) | **Zero floating-point**. Menghilangkan bug pembulatan desimal yang fatal dalam akuntansi. |
| **Integritas Stok**| `CHECK (quantity >= 0)` | Saldo negatif dicegah langsung di level kernel database PostgreSQL. |

---

## 📁 Arsitektur & Struktur Folder

Proyek ini menerapkan **Modular Layered Architecture** dengan dependensi yang mengarah ke dalam (*clean architecture*):

```text
.
├── cmd/
│   └── api/
│       └── main.go           # Entry point utama aplikasi
├── docs/
│   ├── AGENT.md              # Aturan teknis dan konvensi AI/Developer
│   ├── prd.md                # PRD Backend & Spesifikasi Sistem
│   └── frontend-prd.md       # PRD Frontend Web POS & Backoffice
├── internal/
│   ├── app/                  # Bootstrap server, middleware CORS, & graceful shutdown
│   ├── config/               # Parsing environment variables (.env)
│   ├── handler/              # HTTP Controller (Request parser & response JSON)
│   ├── middleware/           # JWT Auth, RBAC, Store Scoping (X-Store-ID), Logger, Custom Error Handler
│   ├── model/                # Entity structs, DTOs, & Standar AppError
│   ├── platform/
│   │   └── database/         # Pgxpool connection, auto-migration runner, & seed master data
│   ├── repository/           # Parameterized SQL database queries & transaction handling
│   ├── route/                # Definisi routing API v1
│   └── service/              # Logic bisnis, kalkulasi pajak, locking deterministik, & approval
├── migrations/
│   ├── 000001_init_schema.up.sql   # Skema tabel relasional & DB constraints
│   └── 000001_init_schema.down.sql # Rollback migration
├── tests/
│   ├── api_contract_test.go          # Pengujian status code & format JSON error
│   ├── business_rules_test.go        # Unit test pajak tax-inclusive, idempotency hash, self-approval
│   ├── frontend_compatibility_test.go# Pengujian kesesuaian endpoint dengan Frontend PRD
│   └── integration_test.go           # End-to-end integration test (Auth -> Shift -> POS -> Void -> Offline)
├── .env.example              # Template variabel lingkungan
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

---

## 🚀 Panduan Memulai (Quick Start)

### 1. Prasyarat
- Go 1.24 atau lebih baru
- PostgreSQL 14+ (pastikan service database sudah berjalan)

### 2. Kloning Repository
```bash
git clone https://github.com/ulhaqjackyhaw/POS-GO-BE.git
cd POS-GO-BE
```

### 3. Konfigurasi Lingkungan (`.env`)
Salin file `.env.example` ke `.env`:
```bash
cp .env.example .env
```
Sesuaikan konfigurasi koneksi database Anda pada file `.env`:
```env
APP_NAME="Go Fiber POS Backend"
APP_ENV="development"
APP_PORT="8080"

# Format DSN PostgreSQL
DATABASE_URL="postgres://postgres:password@localhost:5432/pos_db?sslmode=disable"

# JWT Config
JWT_SECRET="pos-super-secret-key-change-in-production-2026"
JWT_EXPIRATION_HOURS="24"

# Tarif Pajak Standar (1100 = 11.00%)
DEFAULT_TAX_RATE_BPS="1100"
```

### 4. Menjalankan Aplikasi
```bash
go run ./cmd/api
```
> **Catatan:** Saat server pertama kali berjalan, sistem akan **otomatis mengeksekusi migrasi database** dan mengisi data awal (*auto-seed*) toko, gudang, terminal kasir, role, permission, serta akun default.

---

## 👤 Akun & Data Master Bawaan (Default Seeded)

| Username | Password | Role | Cakupan Hak Akses |
|---|---|---|---|
| `admin` | `password123` | `owner` | Akses penuh seluruh cabang, master data, konfigurasi, laporan, dan audit trail. |
| `manager1` | `password123` | `store_manager` | Cabang Jakarta (`STORE-JKT-01`), review approval, void nota, penyesuaian stok. |
| `kasir1` | `password123` | `cashier` | Cabang Jakarta (`STORE-JKT-01`), buka/tutup shift, quote cart, dan checkout penjualan. |

### Data Toko Awal
- **Cabang:** `Cabang Utama - Jakarta` (`STORE-JKT-01`)
- **Gudang:** `Gudang Utama Jakarta` (`WH-JKT-01`)
- **Terminal:** `Terminal Kasir 1` (`POS-01`)
- **Produk:** Kopi Susu Gula Aren (Rp 18.000), Roti Bakar (Rp 15.000), Teh Melati Spesial (Base: Rp 8.000, Override Cabang: Rp 9.000).

---

## 📡 Ringkasan Endpoint API (`/api/v1`)

Semua request store-scoped mendukung identifikasi cabang via header `X-Store-ID: <store_id>`.

### 🔑 Autentikasi
- `POST /api/v1/auth/login` — Login pengguna & penerbitan token JWT.
- `GET  /api/v1/auth/me` — Ambil profil user, daftar permission, dan hak cabang.

### 🏢 Organisasi & Cabang
- `GET  /api/v1/stores` — Daftar seluruh cabang.
- `POST /api/v1/stores` — Tambah cabang baru.
- `GET  /api/v1/stores/:store_id` — Detail cabang.
- `GET  /api/v1/stores/:store_id/warehouses` — Daftar gudang cabang.
- `GET  /api/v1/stores/:store_id/terminals` — Daftar terminal POS cabang.

### 📦 Katalog & Harga
- `GET  /api/v1/categories` — Daftar kategori produk.
- `GET  /api/v1/products` — Katalog produk (dengan resolver harga cabang & stok).
- `POST /api/v1/products` — Tambah produk master baru.
- `POST /api/v1/stores/:store_id/products/:id/price` — Override harga khusus cabang.

### 📊 Inventori & Stok
- `GET  /api/v1/inventory/stock` — Saldo stok terkini produk per gudang.
- `POST /api/v1/inventory/adjust` — Penyesuaian stok manual (alasan wajib).
- `GET  /api/v1/inventory/ledgers` — Histori mutasi buku besar inventori (*audit ledger*).

### 💵 Kasir & Shift
- `GET  /api/v1/shifts/active` — Cek shift kasir yang sedang berjalan.
- `GET  /api/v1/shifts/:id` — Detail shift lengkap & rekonsiliasi kas.
- `POST /api/v1/shifts/open` — Buka shift kasir dengan modal awal.
- `POST /api/v1/shifts/:id/movements` — Catat kas masuk/keluar non-penjualan.
- `POST /api/v1/shifts/:id/close` — Tutup shift & rekonsiliasi kas fisik.

### 🛒 POS Penjualan
- `POST /api/v1/pos/quote` — Preview transaksi & kalkulasi rincian pajak *tax-inclusive*.
- `POST /api/v1/pos/checkout` — Checkout online atomik (wajib menyertakan `Idempotency-Key`).
- `GET  /api/v1/pos/transactions` — Riwayat transaksi penjualan cabang.
- `GET  /api/v1/pos/transactions/:id` — Detail satu nota transaksi.
- `POST /api/v1/pos/transactions/:id/void` — Pembatalan transaksi (mutasi stok kompensasi).
- `POST /api/v1/pos/transactions/:id/refund` — Pengembalian dana transaksi.

### 🛡️ Approval Tindakan Kritis
- `POST /api/v1/approvals` — Buat permintaan persetujuan tindakan berisiko.
- `GET  /api/v1/approvals` — Daftar permintaan approval pending.
- `GET  /api/v1/approvals/:id` — Detail satu approval.
- `POST /api/v1/approvals/:id/review` — Eksekusi persetujuan (*Approve* / *Reject*).

### 🔄 Sinkronisasi Offline
- `POST /api/v1/sync/batch` — Sinkronisasi batch transaksi tunai offline kasir.

### 📈 Laporan & Operasional
- `GET  /api/v1/reports/sales` — Laporan penjualan, omzet bruto/netto, dan diskon.
- `GET  /api/v1/reports/top-products` — Laporan produk terlaris.
- `GET  /health` — Liveness and readiness probe.

---

## 🧪 Pengujian (Testing)

Proyek ini dilengkapi dengan cakupan pengujian komprehensif mulai dari unit test, contract test, hingga end-to-end integration test terhadap PostgreSQL:

```bash
# Jalankan seluruh test suite
go test -v ./tests/...

# Jalankan pengujian tanpa cache
go test -v -count=1 ./tests/...
```

Hasil pengujian mencakup:
- ✅ **API Contract Test:** Memastikan seluruh error format mematuhi standar JSON `{ "error": "...", "code": "..." }`.
- ✅ **Business Rules Test:** Uji determinisme formula pajak *tax-inclusive* & pencegahan *self-approval*.
- ✅ **Frontend Compatibility Test:** Uji kecocokan DTO dual-fields dan endpoint pelengkap frontend.
- ✅ **Complete POS Integration Test:** Alur lengkap login, buka shift, quote, checkout atomik, proteksi stok paralel (*concurrency test*), retry idempotency, void, approval, hingga batch offline sync.

---

## 📄 Format Respons Error Standar

Semua kegagalan API mengembalikan pesan yang ramah pengguna dengan kode mesin yang stabil tanpa membocorkan detail internal server atau SQL query:

```json
{
  "error": "Stok tidak mencukupi untuk produk: Kopi Susu Gula Aren (tersedia: 5, diminta: 10)",
  "code": "INSUFFICIENT_STOCK"
}
```

Daftar kode error umum:
- `VALIDATION_ERROR` (400) — Input request tidak lengkap atau tidak valid.
- `INSUFFICIENT_STOCK` (400) — Stok barang di gudang tidak mencukupi.
- `SHIFT_NOT_OPEN` (400) — Kasir belum membuka shift.
- `CANNOT_APPROVE_OWN_REQUEST` (403) — Pemohon tidak boleh menyetujui permintaannya sendiri.
- `UNAUTHENTICATED` (401) — Token tidak valid atau sesi berakhir.
- `FORBIDDEN` (403) — Hak akses / permission / cakupan cabang tidak mencukupi.
- `NOT_FOUND` (404) — Data tidak ditemukan.
- `IDEMPOTENCY_CONFLICT` (409) — Idempotency key yang sama dikirim dengan payload berbeda.
- `SYNC_CONFLICT` (409) — Transaksi offline mengalami konflik stok/harga.
- `INTERNAL_SERVER_ERROR` (500) — Terjadi kesalahan internal pada server.

---

## 📜 Lisensi & Kontributor

Dikembangkan oleh **[ulhaqjackyhaw](https://github.com/ulhaqjackyhaw)** sebagai bagian dari ekosistem **POS & Backoffice ERP Retail Multi-Outlet**.
Didistribusikan di bawah lisensi MIT.