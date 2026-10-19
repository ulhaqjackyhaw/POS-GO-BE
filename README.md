# 🏢 POS-GO-BE — Enterprise POS & Inventory Backend (Multi-Cabang)

> **Backend REST API Point of Sale (POS) & Manajemen Inventori Ritel Multi-Cabang** berkinerja tinggi, berorientasi integritas data finansial, dan siap mendukung operasional kasir *offline-first*. Dibangun menggunakan **Go 1.24+ (Fiber v2)** dan **PostgreSQL murni (`pgx/v5`)**.

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Framework](https://img.shields.io/badge/Framework-GoFiber_v2-00ACD7?style=flat)](https://gofiber.io)
[![Database](https://img.shields.io/badge/Database-PostgreSQL_pgx_v5-336791?style=flat&logo=postgresql)](https://github.com/jackc/pgx)
[![Architecture](https://img.shields.io/badge/Architecture-Modular_Layered-blueviolet)](#-arsitektur--struktur-folder)
[![Tests](https://img.shields.io/badge/Tests-Passing_100%25-success)](#-pengujian-testing)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Frontend repository: **[POS-FE-TSX (React 19 + TypeScript + Vite)](https://github.com/ulhaqjackyhaw/POS-FE-TSX)**

---

## 📌 Tentang Proyek: Aplikasi Apa Ini & Masalah Apa yang Diselesaikan?

Dalam bisnis ritel yang memiliki banyak cabang (multi-outlet), pemilik bisnis dan pengelola toko sering menghadapi kendala serius yang berdampak langsung pada kerugian finansial:
1. **Selisih Kas Fisik:** Kasir tidak disiplin mencatat kas masuk/keluar non-penjualan sehingga terjadi ketidakcocokan saldo saat tutup shift.
2. **Stok Negatif & Overselling:** Dua kasir menjual sisa 1 unit barang pada detik yang sama (*race condition*), menyebabkan stok gudang tercatat minus di sistem.
3. **Harga Berbeda Antar-Cabang:** Biaya logistik dan harga pasar antar-kota berbeda, tetapi sistem kasir lama tidak fleksibel mendukung harga khusus per cabang.
4. **Kecurangan (Fraud) Kasir:** Transaksi dibatalkan (*void*) atau di-refund sepihak tanpa otorisasi supervisor.
5. **Koneksi Internet Toko Terputus:** Transaksi kasir terhenti total ketika koneksi internet terganggu.

### 💡 Solusi yang Diberikan `POS-GO-BE`
`POS-GO-BE` bertindak sebagai **pusat kebenaran (*single source of truth*) data operasional dan keuangan** ritel modern:
- **Transaksi ACID & Atomik:** Seluruh alur checkout mengunci stok produk secara deterministik (`FOR UPDATE`) dalam satu transaksi database PostgreSQL yang atomik.
- **Garansi Stok Anti-Minus:** Dilindungi oleh database constraint `CHECK (quantity >= 0)`. Sistem menolak transaksi secara otomatis jika stok tidak mencukupi tanpa ada celah *race condition*.
- **Anti-Double Submission (Idempotency):** Request checkout wajib menyertakan header `Idempotency-Key` bertipe UUID. Double-click kasir atau koneksi timeout yang di-retry tidak akan pernah menduplikasi transaksi atau memotong stok dua kali.
- **Pemisahan Otorisasi (Approval Workflow):** Aksi berisiko tinggi (void, refund, penyesuaian stok besar) wajib melalui persetujuan manajer/supervisor dengan aturan ketat: *pemohon tidak boleh menyetujui permintaannya sendiri*.
- **Offline Batch Synchronization:** Mendukung endpoint `POST /api/v1/sync/batch` untuk menyinkronkan transaksi tunai kasir yang dilakukan saat toko offline secara aman dan tervalidasi.
- **Zero Floating-Point Error:** Seluruh nominal uang dan kalkulasi diskon/pajak menggunakan integer Rupiah murni (`int64`), menghilangkan selisih pembulatan desimal yang umum merusak pembukuan akuntansi.

---

## ✨ Fitur-Fitur Utama (Key Features)

### 1. 🏬 Multi-Cabang & Hierarki Organisasi
- Pengelolaan multi-toko (**Store**), gudang penyimpanan (**Warehouse**), dan terminal kasir (**Terminal POS**).
- Pembatasan data per cabang: kasir cabang A tidak dapat melihat atau memanipulasi data cabang B (dikontrol via header `X-Store-ID`).

### 2. 🔐 RBAC (Role-Based Access Control) & Autentikasi
- Login aman berbasis token **JWT** dengan password terenkripsi **bcrypt**.
- Multi-role bawaan: `owner` (akses global), `store_manager`, `supervisor`, `cashier`, `inventory_staff`, dan `auditor`.
- Validasi izin berbasis permission (*permission guard*) di setiap endpoint API dan verifikasi cakupan cabang (*store-scoping*).

### 3. 🏷️ Manajemen Katalog & Dynamic Branch Pricing
- Pencarian produk cepat berdasarkan **Nama, SKU, atau Barcode**.
- **Harga Khusus Cabang (*Price Override*):** Setiap cabang dapat memiliki harga jual berbeda untuk produk yang sama, dengan fallback otomatis ke harga dasar nasional (*base price*).
- Snapshot harga dan tarif pajak tersimpan permanen di baris transaksi sehingga perubahan harga di masa depan tidak merusak histori nota lama.

### 4. 💵 Siklus Shift Kasir & Rekonsiliasi Kas (Cash Management)
- Kasir wajib membuka shift dengan mencatat **modal awal kas (*starting cash*)**.
- Pencatatan pergerakan kas kecil non-penjualan (**Cash In / Cash Out**) dengan input alasan audit wajib.
- **Tutup Shift & Rekonsiliasi Otomatis:** Sistem menghitung ekspektasi kas fisik laci (`Modal Awal + Total Penjualan Tunai + Kas Masuk - Kas Keluar`), membandingkannya dengan uang fisik hitungan kasir, dan mencatat selisih (*cash discrepancy*).

### 5. ⚡ POS Checkout Atomik & Idempotent
- **Kalkulasi Pajak Tax-Inclusive:** Harga jual sudah termasuk PPN (dihitung presisi: `pajak = harga × r / (1 + r)`).
- **Idempotency Key:** Mencegah nota terproses ganda saat kasir melakukan *double-click* atau koneksi *retry*. Request ulang dengan key sama akan mengembalikan data nota yang sudah berhasil tanpa mengurangi stok 2 kali.
- **Deterministic Row-Level Locking (`FOR UPDATE`):** Mengunci baris stok produk dalam urutan deterministik saat checkout, menjamin tidak ada *race condition*, tidak ada deadlock, dan stok tidak pernah minus.

### 6. 📦 Inventori Real-Time & Immutable Ledger
- Saldo stok terpantau real-time per gudang dan per cabang.
- **Buku Besar Mutasi Stok Abadi (*Immutable Ledger*):** Setiap pergerakan barang (penerimaan, penjualan kasir, penyesuaian manual, void, refund) dicatat sebagai mutasi baru. Data mutasi tidak boleh diedit atau dihapus demi integritas audit trail.
- Penyesuaian stok (*Stock Adjustment*) wajib menyertakan alasan dan identitas pengguna yang melakukan mutasi.

### 7. 🛡️ Kontrol Tindakan Kritis & Approval System
- Tindakan berisiko tinggi (void nota, refund uang, penyesuaian stok) dapat mewajibkan **Approval Permintaan**.
- **Prinsip Pemisahan Tugas (Separation of Duties):** Pemohon dilarang menyetujui permintaannya sendiri (`CANNOT_APPROVE_OWN_REQUEST`). Review harus dilakukan oleh akun dengan role Store Manager atau Supervisor yang berbeda.

### 8. 📶 Offline Queue Synchronization (Mode Kasir Offline)
- Kasir tetap dapat melayani penjualan **tunai (CASH)** saat offline.
- Transaksi disimpan di antrean lokal perangkat (IndexedDB) dan dikirim secara batch saat koneksi pulih (`POST /api/v1/sync/batch`).
- Server memvalidasi batch secara idempoten dan mengembalikan status eksplisit per transaksi (`SYNCED` vs `SYNC_CONFLICT`) tanpa merusak data transaksi valid lainnya.

### 9. 🧾 Struk Penjualan, Void & Refund Kompensasi
- Format payload struk siap cetak untuk printer thermal (58mm / 80mm).
- Pembatalan nota (*Void*) dan pengembalian uang (*Refund*) dilakukan via **transaksi kompensasi** yang mengembalikan stok ke gudang secara transparan tanpa memanipulasi atau menghapus jejak nota aslinya.

### 10. 📊 Laporan Analitik Bisnis & Health Probe
- Ringkasan omzet bruto, total diskon, pajak, refund, dan penjualan bersih (*net sales*).
- Peringkat produk terlaris (*Top Products by Quantity & Sales*).
- Endpoint `/health` probe untuk monitoring kesiapan dan kesehatan database serta web server.

---

## 🛠️ Tech Stack & Keputusan Arsitektur

| Komponen | Teknologi | Alasan Pemilihan |
|---|---|---|
| **Bahasa** | Go 1.24+ | Performa mendekati C/C++, efisiensi memori tinggi, konkurensi native (Goroutines), kompilasi ke binary mandiri (*single static binary*). |
| **HTTP Engine** | Fiber v2 (`github.com/gofiber/fiber/v2`) | Web framework berkinerja sangat tinggi berbasis Fasthttp dengan alokasi memori nol (*zero memory allocation*) dan ekosistem middleware lengkap. |
| **Database** | PostgreSQL 14+ | Standar industri untuk kepatuhan ACID, foreign key constraints, JSONB, dan mekanisme transactional row locking yang matang. |
| **Driver Database** | `pgx/v5` & `pgxpool` | Driver PostgreSQL murni tercepat di Go tanpa overhead ORM (menghindari GORM demi kontrol mutlak atas query SQL, locking `FOR UPDATE`, dan transaksi bertingkat). |
| **Autentikasi** | `golang-jwt/jwt/v5` | Token JWT stateless standar industri dengan masa berlaku terkelola dan validasi claims yang aman. |
| **Keamanan Password**| `golang.org/x/crypto/bcrypt` | Hashing password standar industri dengan salt otomatis untuk keamanan maksimal terhadap serangan brute-force. |
| **Format Uang** | Integer Minor Units (`int64` Rupiah bulat) | **Zero floating-point**. Menghilangkan bug pembulatan desimal yang fatal dalam akuntansi dan audit finansial. |
| **Integritas Stok** | `CHECK (quantity >= 0)` | Garansi matematis bahwa stok tidak pernah minus langsung di level kernel PostgreSQL. |

---

## 🏗️ Arsitektur & Struktur Folder

Proyek ini menerapkan **Modular Layered Architecture** dengan alur dependensi satu arah yang bersih:

```text
go-backend/
├── cmd/
│   └── api/
│       └── main.go                 # Entry point utama aplikasi
├── docs/
│   ├── AGENT.md                    # Aturan teknis dan konvensi AI/Developer
│   ├── prd.md                      # PRD Backend & Spesifikasi Sistem
│   └── frontend-prd.md             # PRD Frontend Web POS & Backoffice
├── internal/
│   ├── app/                        # Bootstrap server, middleware CORS, & graceful shutdown
│   ├── config/                     # Parsing environment variables (.env)
│   ├── handler/                    # HTTP Controller (Request parser & response JSON)
│   ├── middleware/                 # JWT Auth, RBAC, Store Scoping (X-Store-ID), Logger, Custom Error
│   ├── model/                      # Entity structs, DTOs, & Standar AppError
│   ├── platform/
│   │   └── database/               # Pgxpool connection, auto-migration runner, & seed master data
│   ├── repository/                 # Parameterized SQL database queries & transaction handling
│   ├── route/                      # Definisi routing API v1
│   └── service/                    # Logic bisnis, kalkulasi pajak, locking deterministik, & approval
├── migrations/
│   ├── 000001_init_schema.up.sql   # Skema tabel relasional & DB constraints
│   └── 000001_init_schema.down.sql # Rollback migration
├── tests/
│   ├── api_contract_test.go        # Pengujian status code & format JSON error
│   ├── business_rules_test.go      # Unit test pajak tax-inclusive, idempotency hash, self-approval
│   ├── frontend_compatibility_test.go # Pengujian kesesuaian endpoint dengan Frontend PRD
│   └── integration_test.go         # End-to-end integration test (Auth -> Shift -> POS -> Void -> Offline)
├── .env.example                    # Template variabel lingkungan
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

---

## 🚀 Panduan Memulai (Quick Start)

### 1. Prasyarat Sistem
- **Go:** Versi `1.24` atau lebih baru.
- **PostgreSQL:** Versi `14+` (pastikan service database sudah berjalan).

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

# Konfigurasi Token JWT
JWT_SECRET="pos-super-secret-key-change-in-production-2026"
JWT_EXPIRATION_HOURS="24"

# Tarif Pajak Standar (1100 = 11.00%)
DEFAULT_TAX_RATE_BPS="1100"
```

### 4. Menjalankan Server Backend
```bash
go run ./cmd/api
```
> **Catatan:** Saat server pertama kali berjalan, sistem akan **otomatis mengeksekusi migrasi skema database** dan mengisikan data awal (*auto-seed*) toko, gudang, terminal kasir, role, permission, serta akun default.

---

## 👥 Akun & Data Master Bawaan (Default Seeded)

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

Semua request yang berkaitan dengan cabang mendukung identifikasi toko via header:
```http
X-Store-ID: <store_id>
```

### 🔑 Autentikasi & Akun
- `POST /api/v1/auth/login` — Login pengguna & penerbitan token JWT.
- `GET  /api/v1/auth/me` — Ambil profil user, daftar permission, dan hak akses cabang.
- `GET  /api/v1/roles` — Daftar role jabatan sistem (Owner, Store Manager, Cashier, Supervisor, dll.).
- `GET  /api/v1/users` — Daftar akun staf & pengguna (filter `?role=`, `?store_id=`, `?search=`).
- `POST /api/v1/users` — Tambah akun staf baru beserta penempatan cabang kustom (`store_ids`).
- `GET  /api/v1/users/:id` — Detail akun staf.
- `PUT  /api/v1/users/:id` — Perbarui nama, role, cabang penempatan, atau reset password.
- `PATCH /api/v1/users/:id/status` — Aktifkan / nonaktifkan akun pengguna.

### 🏬 Organisasi, Cabang & Gudang
- `GET  /api/v1/stores` — Daftar seluruh cabang toko.
- `POST /api/v1/stores` — Tambah cabang toko baru.
- `GET  /api/v1/stores/:store_id` — Detail satu cabang.
- `GET  /api/v1/stores/:store_id/warehouses` — Daftar gudang cabang.
- `POST /api/v1/stores/:store_id/warehouses` — Tambah gudang baru di cabang.
- `GET  /api/v1/stores/:store_id/terminals` — Daftar terminal POS cabang.
- `GET  /api/v1/warehouses` — Daftar seluruh master gudang (opsional filter `?store_id=`).
- `POST /api/v1/warehouses` — Tambah master data gudang baru.
- `GET  /api/v1/warehouses/:id` — Detail master data gudang.
- `PUT  /api/v1/warehouses/:id` — Perbarui data gudang.
- `DELETE /api/v1/warehouses/:id` — Hapus / nonaktifkan gudang.

### 🏷️ Katalog & Harga
- `GET  /api/v1/categories` — Daftar kategori produk.
- `GET  /api/v1/products` — Katalog produk (dengan resolver harga cabang & stok gudang).
- `POST /api/v1/products` — Tambah produk master baru.
- `POST /api/v1/stores/:store_id/products/:id/price` — Override harga khusus cabang.

### 📦 Inventori & Stok
- `GET  /api/v1/inventory/stock` — Saldo stok terkini produk per gudang.
- `POST /api/v1/inventory/adjust` — Penyesuaian stok manual (wajib alasan audit).
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

### 📶 Sinkronisasi Offline
- `POST /api/v1/sync/batch` — Sinkronisasi batch transaksi tunai offline kasir.

### 📈 Laporan & Operasional
- `GET  /api/v1/reports/sales` — Laporan omzet bruto/netto, diskon, dan pajak.
- `GET  /api/v1/reports/top-products` — Laporan produk terlaris.
- `GET  /health` — Liveness & readiness probe.

---

## 🧪 Pengujian (Testing)

Proyek ini dilengkapi dengan cakupan pengujian komprehensif mulai dari unit test, contract test, hingga end-to-end integration test terhadap PostgreSQL:

```bash
# Jalankan seluruh test suite
go test -v ./tests/...

# Jalankan pengujian tanpa cache
go test -v -count=1 ./tests/...
```

Cakupan pengujian:
- ✅ **API Contract Test:** Memastikan seluruh format error mematuhi standar JSON `{ "error": "...", "code": "..." }`.
- ✅ **Business Rules Test:** Uji determinisme formula pajak *tax-inclusive* & pencegahan *self-approval*.
- ✅ **Frontend Compatibility Test:** Uji kecocokan DTO dual-fields dan kelengkapan endpoint frontend.
- ✅ **Complete POS Integration Test:** Alur lengkap login, buka shift, quote, checkout atomik, proteksi stok paralel (*concurrency test*), retry idempotency, void, approval, hingga batch offline sync.

---

## ⚠️ Format Respons Error Standar

Semua kegagalan API mengembalikan pesan yang informatif dan ramah pengguna dengan kode mesin yang stabil tanpa membocorkan detail internal server atau SQL query:

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
- `CANNOT_APPROVE_OWN_REQUEST` (403) — Pemohon dilarang menyetujui permintaannya sendiri.
- `UNAUTHENTICATED` (401) — Token tidak valid atau sesi login berakhir.
- `FORBIDDEN` (403) — Hak akses / permission / cakupan cabang tidak mencukupi.
- `NOT_FOUND` (404) — Data tidak ditemukan.
- `IDEMPOTENCY_CONFLICT` (409) — Idempotency key yang sama dikirim dengan payload berbeda.
- `SYNC_CONFLICT` (409) — Transaksi offline mengalami konflik stok/harga.
- `INTERNAL_SERVER_ERROR` (500) — Terjadi kesalahan internal pada server.

---

## 🤝 Lisensi & Kontributor

Dikembangkan oleh **[ulhaqjackyhaw](https://github.com/ulhaqjackyhaw)** sebagai bagian dari ekosistem **Enterprise POS & Backoffice ERP Retail Multi-Outlet**.

Didistribusikan di bawah lisensi **MIT**.
