# Frontend Product Requirement Document (PRD)
## Web POS & Backoffice ERP Retail Multi-Outlet

**Status:** Draft siap implementasi  
**Versi:** 1.0  
**Bahasa:** Indonesia  
**Target pembaca:** Frontend Engineer, UI/UX Engineer, QA, Backend Engineer, dan AI coding agent  
**Backend:** REST API Go Fiber + PostgreSQL, prefix `/api/v1`  
**Frontend stack rekomendasi:** React + Vite + TypeScript + Tailwind CSS + shadcn/ui

---

## 1. Ringkasan Produk

Aplikasi ini merupakan frontend web untuk sistem Point of Sale (POS) dan backoffice ERP retail multi-outlet. Aplikasi mengutamakan kecepatan transaksi kasir, ketepatan perhitungan uang dan stok, kontrol akses berbasis peran (RBAC), auditabilitas tindakan kritis, serta ketahanan saat koneksi internet terputus.

Frontend adalah klien dari backend Go Fiber. Backend tetap menjadi sumber kebenaran untuk harga final, pajak, diskon yang diizinkan, ketersediaan stok, otorisasi, status shift, nomor transaksi, dan hasil checkout. Frontend boleh menghitung estimasi untuk pengalaman pengguna, tetapi harus mengonfirmasi hasil akhir melalui endpoint quote/checkout.

### 1.1 Tujuan utama

1. Memungkinkan kasir menyelesaikan transaksi dengan sedikit interaksi mouse.
2. Mengurangi kesalahan pemilihan cabang, shift, produk, harga, diskon, pembayaran, dan rekonsiliasi.
3. Mendukung transaksi tunai saat offline melalui antrean lokal persisten, dengan status sinkronisasi yang jelas.
4. Menyediakan backoffice untuk pengelolaan produk, harga per cabang, inventori, laporan, dan approval.
5. Memisahkan UI presentasional dari logika bisnis agar mudah diuji dan dipelihara.
6. Menjaga seluruh perubahan stok dan transaksi konsisten dengan aturan backend.

### 1.2 Prinsip produk

- **Backend authoritative:** hasil quote, checkout, stok, pajak, diskon, dan izin final ditentukan backend.
- **Fail safe:** ketika hasil transaksi tidak pasti, jangan mengirim checkout ulang tanpa idempotency key.
- **Cash-only offline:** hanya transaksi tunai yang boleh masuk antrean offline; metode pembayaran lain dinonaktifkan saat offline.
- **Tidak ada stok negatif:** frontend menampilkan stok yang diketahui, tetapi backend tetap melakukan validasi atomik.
- **Branch-scoped:** semua query/mutasi relevan menggunakan `X-Store-ID` dan pemeriksaan izin.
- **Audit-first:** tindakan kritis harus menyertakan alasan dan melalui approval bila aturan backend mewajibkannya.
- **Accessible and fast:** fokus keyboard konsisten, status dapat dibaca screen reader, dan layout responsif.

### 1.3 Batasan dan asumsi penting

1. Endpoint yang disebutkan adalah kontrak target. Tim backend harus menetapkan schema respons, kode error, pagination, dan payload final sebelum integrasi produksi.
2. `X-Store-ID` wajib untuk endpoint store-scoped. Login dan pemilihan cabang dapat menjadi pengecualian sesuai kebijakan backend.
3. Untuk web, utamakan sesi melalui cookie `HttpOnly`, `Secure`, `SameSite` bila backend mendukung. Jika Bearer token harus dibaca JavaScript, minimalkan masa hidup token, lindungi dari XSS, dan evaluasi risiko penyimpanan token.
4. Transaksi offline harus memiliki ID unik buatan klien dan `idempotency_key`. Backend wajib mendukung idempotensi dan kontrak batch sync eksplisit. Tanpa dukungan ini, offline checkout belum aman untuk produksi.
5. `window.print()` adalah fallback cetak. Thermal printer otomatis dapat memerlukan bridge/agent lokal, WebUSB/WebSerial yang kompatibel, atau aplikasi pendamping.
6. Tidak diasumsikan seluruh API sudah mendukung pagination, idempotensi, quote expiry, atau resolusi konflik.

---

## 2. Persona, Peran, dan Hak Akses

### 2.1 Persona

- **Kasir:** memilih produk, membuat cart, menerima pembayaran tunai, membuka/menutup shift, mencetak struk.
- **Supervisor:** mengawasi shift, menyetujui tindakan kritis, melakukan void/refund bila berwenang.
- **Store Manager:** mengelola harga cabang, penyesuaian stok, laporan cabang, dan pengguna sesuai izin.
- **Inventory Staff:** melihat ledger dan membuat permintaan penyesuaian/mutasi stok.
- **Owner/Admin:** mengelola cabang, produk, konfigurasi, dan laporan lintas cabang.
- **Auditor:** akses baca laporan dan audit/approval sesuai ruang lingkup.

### 2.2 RBAC di frontend

Frontend menggunakan izin dari `GET /auth/me` untuk menyembunyikan atau menonaktifkan aksi. Backend wajib mengotorisasi ulang setiap request.

Permission yang disarankan:
- `pos.sell`, `pos.discount`, `pos.checkout`
- `shift.open`, `shift.close`, `shift.movement`
- `product.read`, `product.create`, `product.update`
- `pricing.read`, `pricing.override`
- `inventory.read`, `inventory.adjust`
- `transaction.read`, `transaction.void`, `transaction.refund`
- `approval.create`, `approval.review`
- `report.sales.read`, `report.products.read`
- `store.read`, `store.create`

Jika izin tidak diketahui, aksi sensitif tidak boleh diaktifkan. Respons `401` mengakhiri sesi; `403` menampilkan akses ditolak tanpa retry otomatis.

---

## 3. Rekomendasi Stack Frontend

### 3.1 Stack utama

| Area | Pilihan | Alasan |
|---|---|---|
| Framework | React + Vite + TypeScript | Cocok untuk aplikasi internal yang interaktif dan dapat di-deploy sebagai SPA/PWA |
| Routing | React Router | Routing dan route guard sederhana untuk SPA |
| Styling | Tailwind CSS | Konsistensi token desain dan iterasi cepat |
| UI primitives | shadcn/ui + Radix UI | Komponen dapat dikustomisasi dan aksesibilitas baik |
| Ikon | Lucide | Ikon konsisten |
| Server state | TanStack Query | Cache, invalidation, mutation lifecycle, retry terkontrol |
| Global client state | Zustand | Ringan untuk cart, shift aktif, store aktif, jaringan dan queue |
| Form | React Hook Form + Zod | Validasi dan tipe yang konsisten |
| HTTP | `fetch` wrapper atau Axios | Pengaturan header, timeout, dan normalisasi error |
| Offline storage | IndexedDB via Dexie atau LocalForage | Persistensi queue dan katalog minimum |
| PWA | Workbox (opsional) | Cache aset aplikasi dan pengalaman instalasi |
| Unit/component test | Vitest + React Testing Library | Uji perilaku UI dan hook |
| E2E | Playwright | Uji alur login, shift, checkout, offline |
| Lint/format | ESLint + Prettier | Konsistensi kode |

**Keputusan default:** gunakan React + Vite untuk POS/backoffice SPA/PWA. Next.js layak jika ada kebutuhan SSR/portal publik, tetapi SSR tidak otomatis menguntungkan aplikasi internal yang hampir seluruhnya terautentikasi. Jangan mencampur pola Next.js App Router dan SPA tanpa keputusan arsitektur eksplisit.

### 3.2 Strategi state management

#### Server state — TanStack Query

Gunakan untuk pengguna/permissions, store, warehouse, terminal, produk, harga, stok, ledger, shift, transaksi, approval, laporan, dan hasil quote.

Aturan:
- Query key menyertakan `storeId` untuk data cabang, misalnya `['products', storeId, filters]`.
- Saat `storeId` berubah, invalidate cache terkait cabang dan batalkan request lama bila memungkinkan.
- Hasil quote bukan fakta permanen; quote dapat kedaluwarsa.
- Retry otomatis hanya untuk operasi baca yang aman. Mutasi checkout tidak boleh di-retry secara buta.
- Gunakan skeleton pada loading awal; hindari spinner global pada interaksi kecil.
- Katalog boleh memiliki stale time beberapa menit; shift aktif dan data yang berdampak transaksi harus lebih segar.

#### Global client state — Zustand

Gunakan untuk:
- `activeStoreId` dan `activeTerminalId`;
- snapshot UI `activeShift` (backend tetap sumber kebenaran);
- `cartItems` dan pilihan diskon lokal;
- status jaringan dan `pendingSyncCount`;
- status antrean sinkronisasi dari IndexedDB;
- modal global yang memang perlu bertahan lintas komponen.

Jangan menyalin semua respons API ke Zustand. Jangan menyimpan token atau rahasia dalam persist middleware tanpa penilaian keamanan.

#### Form dan validasi

- React Hook Form + Zod untuk login, buka/tutup shift, cash movement, produk, override harga, adjustment stok, approval, dan filter laporan.
- Validasi frontend memperbaiki UX; validasi backend tetap wajib.
- Gunakan integer minor units atau format desimal yang ditentukan kontrak; jangan gunakan floating-point untuk aritmetika uang.
- Kuantitas memakai presisi eksplisit sesuai tipe produk.

### 3.3 Contoh tipe client state

```ts
type AppSession = {
  userId: string;
  displayName: string;
  permissions: string[];
};

type StoreContext = {
  activeStoreId: string | null;
  activeTerminalId: string | null;
};

type ActiveShiftState = {
  id: string;
  status: "open" | "closing" | "closed";
  openedAt: string;
  openingCashMinor: number;
} | null;

type CartItem = {
  productId: string;
  sku: string;
  barcode?: string;
  name: string;
  quantity: string;
  unitPriceMinor: number;
  discount?: { type: "amount" | "percent"; value: number };
  priceVersion?: string;
};

type SyncStatus = "online" | "offline" | "syncing" | "attention";
```

Cart boleh menyimpan snapshot nama/harga untuk rendering, tetapi backend harus mengonfirmasi harga, diskon, pajak, dan stok saat quote/checkout.

---

## 4. Arsitektur dan Struktur Folder

### 4.1 Prinsip

1. **Feature-first:** setiap domain bisnis memiliki folder sendiri.
2. **UI kit terpisah dari domain:** komponen generik tidak mengetahui aturan POS.
3. **Business logic di hooks/services/stores:** halaman merakit komponen, bukan menampung semua logika.
4. **API client terpusat:** mengatur base URL, token, store header, error, timeout, dan correlation ID.
5. **Types per domain:** tipe request/response diletakkan dekat feature atau di `types/api`.
6. **Kontrak tidak ditebak:** schema respons diselaraskan dengan OpenAPI backend.

### 4.2 Directory tree

```text
pos-frontend/
├── public/
│   ├── icons/
│   └── manifest.webmanifest
├── src/
│   ├── app/
│   │   ├── App.tsx
│   │   ├── router.tsx
│   │   ├── providers/
│   │   │   ├── QueryProvider.tsx
│   │   │   └── ThemeProvider.tsx
│   │   ├── layouts/
│   │   │   ├── AuthLayout.tsx
│   │   │   ├── PosLayout.tsx
│   │   │   └── BackofficeLayout.tsx
│   │   └── route-guards/
│   │       ├── RequireAuth.tsx
│   │       ├── RequirePermission.tsx
│   │       ├── RequireStore.tsx
│   │       └── RequireActiveShift.tsx
│   ├── api/
│   │   ├── client.ts
│   │   ├── errors.ts
│   │   ├── auth.api.ts
│   │   ├── stores.api.ts
│   │   ├── products.api.ts
│   │   ├── shifts.api.ts
│   │   ├── pos.api.ts
│   │   ├── inventory.api.ts
│   │   ├── approvals.api.ts
│   │   ├── sync.api.ts
│   │   └── reports.api.ts
│   ├── components/
│   │   ├── ui/                 # Button, Dialog, Input, Table, Badge
│   │   ├── feedback/           # LoadingState, EmptyState, ErrorState
│   │   ├── data-display/       # Money, DateTime, StatusBadge
│   │   ├── navigation/         # Sidebar, Topbar, Breadcrumbs
│   │   └── scanner/            # BarcodeInput, ScannerStatus
│   ├── features/
│   │   ├── auth/
│   │   ├── stores/
│   │   ├── shifts/
│   │   ├── pos/
│   │   │   ├── pages/PosPage.tsx
│   │   │   ├── components/ProductGrid.tsx
│   │   │   ├── components/CartPanel.tsx
│   │   │   ├── components/PaymentDialog.tsx
│   │   │   ├── components/ReceiptView.tsx
│   │   │   ├── hooks/useBarcodeScanner.ts
│   │   │   ├── hooks/usePosShortcuts.ts
│   │   │   ├── hooks/useQuote.ts
│   │   │   ├── pos.store.ts
│   │   │   ├── pos.schemas.ts
│   │   │   └── pos.types.ts
│   │   ├── products/
│   │   ├── pricing/
│   │   ├── inventory/
│   │   ├── transactions/
│   │   ├── approvals/
│   │   ├── reports/
│   │   └── sync/
│   ├── hooks/
│   │   ├── useDebounce.ts
│   │   ├── useOnlineStatus.ts
│   │   └── usePermission.ts
│   ├── stores/
│   │   ├── app-context.store.ts
│   │   ├── shift.store.ts
│   │   ├── network.store.ts
│   │   └── sync.store.ts
│   ├── db/
│   │   ├── indexeddb.ts
│   │   ├── schema.ts
│   │   └── migrations.ts
│   ├── types/
│   │   ├── api/
│   │   ├── auth.ts
│   │   ├── store.ts
│   │   ├── money.ts
│   │   └── common.ts
│   ├── utils/
│   │   ├── currency.ts
│   │   ├── date.ts
│   │   ├── permissions.ts
│   │   ├── idempotency.ts
│   │   └── errors.ts
│   ├── styles/globals.css
│   ├── test/fixtures/
│   └── main.tsx
├── e2e/
├── .env.example
├── package.json
├── tsconfig.json
├── vite.config.ts
└── README.md
```

### 4.3 Pemisahan tanggung jawab

| Lapisan | Tanggung jawab | Tidak boleh |
|---|---|---|
| `components/ui` | Primitives presentasional, aksesibilitas, style | Memanggil endpoint atau mengetahui aturan stok |
| `features/*/components` | Komponen khusus domain | Menyimpan seluruh business flow sendiri |
| `features/*/hooks` | Orkestrasi query/mutation, keyboard, scanner | Menentukan otorisasi final |
| `api/*` | Kontrak HTTP dan normalisasi respons | Mengambil keputusan bisnis final |
| `stores/*` | State interaksi lintas halaman | Menjadi salinan permanen seluruh server state |
| `db/*` | IndexedDB schema, migrasi, repository queue | Menyimpan data sensitif yang tidak diperlukan |
| `utils/*` | Fungsi murni dan formatter | Melakukan side effect jaringan |
| `app/route-guards` | UX guard berdasar sesi/izin/shift | Menggantikan otorisasi backend |

---

## 5. Design System dan Prinsip UX

### 5.1 Gaya visual

- Backoffice: bersih, tabel mudah dipindai, filter jelas, densitas informasi sedang.
- POS: kontras tinggi, tombol besar, minim dekorasi, status transaksi selalu terlihat.
- Gunakan design token untuk warna, spacing, radius, typography, shadow, dan status.
- Jangan mengandalkan warna saja; sertakan teks/ikon.
- Aksi berbahaya (void, refund, penyesuaian stok) memakai dialog konfirmasi yang menjelaskan objek dan dampak.

### 5.2 Komponen standar

`PageHeader`, `Breadcrumbs`, `PermissionGate`, `DataTable`, `SearchInput`, `BarcodeInput`, `StoreSelector`, `MoneyText`, `QuantityInput`, `DateRangePicker`, `StatusBadge`, `NetworkStatus`, `SyncQueueBadge`, `ConfirmDialog`, `ReasonDialog`, `ApprovalDialog`, `LoadingSkeleton`, `EmptyState`, `ErrorState`, `Toast`.

### 5.3 State UI wajib

Semua layar data mendefinisikan:
- **Loading:** skeleton untuk konten utama.
- **Empty:** penjelasan mengapa kosong dan CTA relevan.
- **Error:** pesan ramah, kode/request ID bila ada, tombol coba lagi untuk operasi aman.
- **Forbidden:** akses ditolak dan tidak ada tombol mutasi.
- **Stale/offline:** label cache dan waktu pembaruan terakhir.
- **Submitting:** cegah double click dan tampilkan progress.
- **Conflict:** jelaskan perubahan dan opsi tindakan yang sah.

### 5.4 Responsivitas

- POS dirancang untuk desktop/tablet landscape, resolusi dasar 1280×720.
- Backoffice usable pada tablet dan desktop; layar sempit dapat memakai kolom prioritas atau kartu.
- Jangan memindahkan fokus secara tak terduga setelah produk ditambahkan.

---

## 6. Routing dan Navigasi

| Path | Layar | Guard |
|---|---|---|
| `/login` | Login | Publik |
| `/select-store` | Pilih cabang dan terminal | Auth + izin store |
| `/shifts` | Shift kasir | Auth + store |
| `/pos` | POS utama | Auth + store + `pos.sell`; shift aktif untuk checkout |
| `/products` | Master produk | `product.read` |
| `/stores/:id/pricing` | Harga cabang | `pricing.read` / `pricing.override` |
| `/inventory` | Stok dan ledger | `inventory.read` |
| `/transactions` | Riwayat transaksi | `transaction.read` |
| `/approvals` | Daftar approval | `approval.review` |
| `/reports` | Laporan | `report.sales.read` |
| `/settings/stores` | Manajemen cabang | `store.read` / `store.create` |

Jika shift belum aktif, arahkan kasir ke `/shifts` dengan CTA “Buka Shift”. Pengguna backoffice yang tidak memiliki izin kasir tidak boleh dipaksa membuka shift.

---

## 7. Spesifikasi Layar dan User Flow

### 7.1 Auth dan Select Store

#### `/login`

**Komponen:** username/email, password dengan show/hide, tombol login, loading, error inline, opsional versi aplikasi.

**Flow:**
1. Validasi format lokal.
2. Kirim `POST /api/v1/auth/login`.
3. Simpan sesi sesuai kebijakan keamanan.
4. Ambil profil/permission lewat `GET /api/v1/auth/me`.
5. Jika perlu, arahkan ke `/select-store`; jika hanya satu store, pilih otomatis hanya jika kebijakan mengizinkan.
6. Jika gagal, tampilkan pesan umum tanpa mengungkap apakah username terdaftar.

**State:** tombol disabled saat loading; `401` tetap di login; `429` hormati `Retry-After`; login baru tidak didukung offline kecuali ada desain autentikasi offline eksplisit.

#### `/select-store`

- Tampilkan daftar cabang yang dapat diakses, pencarian, pilihan terminal dan warehouse.
- Pilih cabang lalu muat `GET /stores/:store_id/warehouses` dan `/terminals`.
- Simpan `activeStoreId`/`activeTerminalId`, set header `X-Store-ID`, invalidate cache store-scoped.
- Arahkan ke `/shifts` atau landing page sesuai permission.
- Jangan pernah mengirim request menggunakan store lama setelah konteks berganti.

### 7.2 Shift Kasir — `/shifts`

**Tampilan:** status shift, kasir, terminal, waktu mulai, modal awal, riwayat cash movement, tombol buka/kas masuk/kas keluar/tutup.

**Modal buka shift**
- Field `opening_cash_minor` wajib integer non-negatif dan catatan opsional.
- Kirim `POST /shifts/open`.
- Sukses: update `shifts/active`, store shift, dan tampilkan toast.
- Konflik shift sudah aktif: refresh status dan tampilkan pesan.

**Modal kas masuk/keluar**
- Jenis `cash_in`/`cash_out`, jumlah, alasan wajib, referensi opsional.
- Tidak boleh tanpa shift aktif.
- Backend memvalidasi batas kas keluar dan menghasilkan ledger.

**Modal tutup shift / rekonsiliasi**
- Tampilkan modal awal, penjualan tunai bersih, kas masuk/keluar, kas ekspektasi, input kas aktual, selisih (`actual - expected`), dan alasan jika tidak nol.
- UI hanya preview; backend menghitung final.
- Kirim `POST /shifts/:id/close`.
- Jika transaksi pending sync atau konflik menghalangi penutupan, jelaskan status dan jangan menandai shift tertutup.

**Kesenjangan API:** belum ada endpoint khusus ringkasan shift historis. Tambahkan `GET /api/v1/shifts/:id` atau perluas `GET /shifts/active` dengan ringkasan yang dibutuhkan untuk rekonsiliasi.

### 7.3 POS Utama — `/pos`

#### Layout

Dual-pane:
- **Kiri (~60%):** search/barcode, kategori/filter, grid produk.
- **Kanan (~40%):** cart, diskon, subtotal, total, pembayaran.
- Header tetap: cabang, terminal, kasir, shift, waktu, status jaringan, jumlah queue.

#### Product grid dan barcode

Kartu menampilkan nama, SKU/barcode, harga cabang, stok diketahui jika tersedia, dan status aktif. Klik menambah satu unit. Barcode tidak dikenal menampilkan pesan singkat dan menjaga fokus.

Scanner keyboard-wedge sering mengirim karakter cepat diikuti Enter:
- Gunakan input barcode terfokus atau listener global yang hanya aktif ketika tidak mengetik di input lain/modal.
- Tetapkan batas waktu antarkarakter yang dapat dikonfigurasi.
- Jangan menangkap scan ketika fokus berada di password, field jumlah, atau dialog alasan.
- Enter pada search biasa harus tetap mencari.
- Sediakan fallback input langsung dan konfigurasi prefix/suffix bila perlu.

#### Cart dan kalkulasi

Tampilkan item, harga satuan, kuantitas, diskon, total baris, subtotal, diskon, pajak termasuk harga, total, dan tombol bayar.

- `unitPriceMinor` hanya snapshot UI, bukan harga final.
- Diskon diperiksa berdasar izin dan policy.
- Kupon belum punya endpoint khusus; dukung hanya bila kontrak quote mendefinisikannya.
- Jangan bersihkan cart sebelum checkout dikonfirmasi atau transaksi tersimpan aman ke queue offline.
- Perubahan store/shift/terminal harus memblokir checkout dan meminta penyelesaian cart sesuai policy.

**Pajak-inclusive:** jika harga bruto `G` sudah termasuk tarif `r`, preview nilai sebelum pajak = `G / (1 + r)` dan pajak = `G - nilai sebelum pajak`. Pembulatan harus sama persis dengan backend. Jangan jadikan preview frontend sumber kebenaran; quote backend menentukan rincian final.

#### Quote

Kirim `POST /pos/quote` setelah perubahan cart (debounce singkat), perubahan diskon/kupon, saat membuka pembayaran, atau ketika harga/stok perlu diperbarui. Respons ideal mencakup line terverifikasi, harga/diskon/pajak/total, warning stok/harga, serta `quote_id` dan expiry. Quote kedaluwarsa harus di-refresh. Bila backend belum punya expiry/versi, sepakati mekanismenya.

#### Shortcut keyboard

| Shortcut | Aksi | Guard |
|---|---|---|
| `F1` | Fokus search | Jangan mencuri fokus dari modal |
| `F2` | Dialog diskon | Butuh `pos.discount` |
| `Space` atau `Enter` | Buka pembayaran | Hanya jika cart valid dan tidak sedang mengetik di input |
| `Esc` | Bersihkan cart setelah konfirmasi | Jangan mengalahkan perilaku modal aktif |
| `Ctrl/Cmd + K` (opsional) | Fokus pencarian | Pastikan tidak bertabrakan dengan browser |

Listener harus mengabaikan `input`, `textarea`, `select`, `contenteditable`, dan dialog yang sedang mengambil fokus. Space tidak boleh memicu pembayaran saat pengguna mengetik.

#### Dialog pembayaran

Field: total final quote, metode pembayaran yang diizinkan, uang diterima, kembalian preview (`received - total`), tombol selesaikan.

Flow online:
1. Pastikan shift aktif, cart valid, quote belum kedaluwarsa.
2. Buat idempotency key unik untuk niat checkout.
3. Kirim `POST /pos/checkout`.
4. Cegah submit ganda.
5. Jika sukses, tampilkan nomor transaksi/struk, bersihkan cart, invalidate transaksi/laporan/shift.
6. Jika timeout dan hasil tidak diketahui, jangan membuat key baru atau retry buta; cek status melalui mekanisme yang disepakati.

#### Offline checkout

Tunai saja. Pastikan sesi, store/terminal, shift yang berlaku, katalog/harga cache masih valid, queue tersedia, dan transaksi mempunyai `client_transaction_id` + `idempotency_key`. Setelah tersimpan atomik ke IndexedDB, tampilkan **“Transaksi offline — menunggu sinkronisasi”**. Jangan mengklaim transaksi sudah final di server.

Default: blokir transaksi offline jika stok lokal tidak cukup atau tidak diketahui. Konflik stok saat sinkronisasi harus ditangani sebagai insiden, bukan dilewati.

#### Struk dan cetak

- Tampilan struk 58 mm dan 80 mm.
- `window.print()` dengan print CSS adalah fallback.
- Thermal auto-print mungkin membutuhkan bridge/agent lokal, WebUSB/WebSerial kompatibel, atau aplikasi pendamping.
- Cetak ulang memakai transaksi tersimpan dan permission sesuai.
- Struk offline menampilkan status pending dan nomor lokal.

### 7.4 Master Produk — `/products`

Tabel dengan search nama/SKU/barcode, kategori, status, pagination. Form tambah/edit minimal: SKU, nama, barcode opsional, kategori, unit, status, deskripsi opsional. Backend menangani validasi unik SKU/barcode. Sukses menginvalidasi daftar dan cache katalog. Tampilkan loading, empty dengan CTA, error, forbidden, dan duplicate-field error.

### 7.5 Override Harga Cabang — `/stores/:id/pricing`

Tampilkan cabang, produk, harga dasar, harga cabang, waktu update, dan pembuat. Modal override berisi harga, tanggal efektif jika didukung, dan alasan bila diwajibkan. Perubahan harga tidak boleh mengubah cart berjalan diam-diam; quote berikutnya harus mendeteksi perubahan. Endpoint: `POST /stores/:store_id/products/:id/price`.

### 7.6 Inventory — `/inventory`

Tab stok saat ini, penyesuaian, ledger. Form adjustment: produk, gudang, jenis/kuantitas adjustment sesuai kontrak, alasan, referensi, dan stok sebelum bila tersedia. Untuk aksi kritis, minta approval bila policy mewajibkan. Jangan optimistically menulis stok final sebelum backend mengonfirmasi; sukses menginvalidasi stok dan ledger.

**Kesenjangan API:** endpoint stok terkini belum tercantum. Tambahkan `GET /api/v1/inventory/stock` atau endpoint sejenis.

### 7.7 Laporan — `/reports`

Filter tanggal, cabang, terminal, kasir sesuai izin. KPI: gross sales, diskon, pajak, net sales, jumlah transaksi, rata-rata transaksi. Tampilkan grafik penjualan per waktu dan tabel produk terlaris (unit/nilai). Hormati timezone dan izin lintas cabang. Endpoint: `GET /reports/sales`, `GET /reports/top-products`.

### 7.8 Approval tindakan kritis

Daftar: ID, jenis tindakan, pemohon, cabang, nilai/dampak, alasan, waktu, status. Detail memuat konteks dan audit trail sesuai izin. Review memakai approve/reject dan catatan bila perlu. Jangan izinkan self-approval bila policy melarang. Setelah keputusan, invalidate daftar/detail dan tampilkan status backend (`pending`, `approved`, `rejected`, `expired` bila didukung).

---

## 8. Offline Queue dan Sinkronisasi

### 8.1 Penyimpanan lokal

Gunakan IndexedDB (misalnya Dexie) untuk katalog minimum, snapshot harga/versi/timestamp, transaksi offline, status sinkronisasi, retry count, dan konfigurasi non-sensitif. Jangan menyimpan nomor kartu, CVV, password, token pembayaran, atau data sensitif yang tidak diperlukan. Bersihkan cache sensitif saat logout sesuai kebijakan perangkat bersama.

```ts
type OfflineTransaction = {
  clientTransactionId: string;
  idempotencyKey: string;
  storeId: string;
  terminalId: string;
  shiftId: string;
  cashierId: string;
  createdAt: string;
  currency: "IDR";
  paymentMethod: "cash";
  items: Array<{
    productId: string;
    quantity: string;
    cachedUnitPriceMinor: number;
    cachedPriceVersion?: string;
  }>;
  discount?: { type: "amount" | "percent"; value: number };
  cashReceivedMinor: number;
  expectedTotalMinor: number;
  payloadVersion: number;
  syncStatus: "pending" | "syncing" | "synced" | "conflict" | "failed";
  retryCount: number;
  lastErrorCode?: string;
};
```

Payload final harus disepakati dengan backend; contoh ini bukan jaminan kompatibilitas `POST /sync/batch`.

### 8.2 Indikator jaringan

Header POS menampilkan Online/Offline, Syncing, jumlah pending, dan Perlu perhatian untuk konflik/gagal permanen. `navigator.onLine` hanya petunjuk; deteksi kegagalan request/health check dengan debounce.

### 8.3 Alur sinkronisasi

1. Simpan transaksi secara atomik di IndexedDB sebelum UI menyatakan tersimpan lokal.
2. Queue worker memproses item dengan urutan yang disepakati (misalnya per store/terminal/shift).
3. Saat online, kirim batch ke `POST /sync/batch`.
4. Backend mengembalikan status per item: `synced`, `duplicate`, `conflict`, atau `rejected`.
5. Simpan hasil server ke IndexedDB sebelum mengurangi pending counter.
6. Retry jaringan/5xx dengan exponential backoff; jangan retry tanpa batas untuk validasi/permission.
7. Idempotency key tetap sama pada setiap retry transaksi yang sama.
8. Sediakan retry manual untuk item yang bisa dicoba ulang.
9. Jangan mengubah transaksi secara diam-diam agar lolos konflik.
10. Jika server menolak transaksi, tandai gagal permanen dan jalankan prosedur koreksi/insiden yang tercatat.

### 8.4 Modal resolusi konflik

Tampilkan nomor lokal, waktu, produk/kuantitas, alasan server, dan status uang tunai yang sudah diterima. Sediakan tindakan sesuai policy: retry setelah perbaikan sah, eskalasi supervisor, atau laporan insiden. Jangan menyediakan “force sync” yang melewati stok/otorisasi. Jika uang sudah diterima tetapi transaksi ditolak, catat sebagai insiden rekonsiliasi.

### 8.5 Keputusan offline yang wajib

Sebelum produksi, tetapkan batas validitas shift offline, cara validasi harga/stok, dua terminal menjual stok terakhir secara offline, batas antrean, prosedur uang yang sudah diterima saat sync ditolak, apakah shift boleh ditutup saat queue pending, serta aturan logout/pergantian kasir/perangkat bersama.

---

## 9. Kontrak API dan Pemetaan Aksi UI

### 9.1 Standar request

- Base URL `/api/v1`.
- `Content-Type: application/json`.
- `Authorization: Bearer <access_token>` untuk endpoint terautentikasi.
- `X-Store-ID: <store_id>` untuk request store-scoped.
- Mutasi kritis/checkout memakai `Idempotency-Key` jika backend mendukung.
- Sertakan request/correlation ID bila didukung.
- Encode path parameter dengan aman.

### 9.2 Tipe dasar TypeScript

```ts
type ID = string;
type ISODateTime = string;
type MoneyMinor = number; // integer unit terkecil mata uang; untuk IDR biasanya rupiah bulat
type ApiError = {
  error: string;
  code: string;
  request_id?: string;
  details?: unknown;
};
type ApiEnvelope<T> = { data: T; meta?: { page?: number; page_size?: number; total?: number } };
type UserSummary = { id: ID; name: string; permissions: string[] };
type Store = { id: ID; name: string; code: string };
type Product = {
  id: ID; sku: string; name: string; barcode?: string;
  unit: string; active: boolean; price_minor?: MoneyMinor;
};
type Shift = {
  id: ID; store_id: ID; terminal_id: ID; status: "open" | "closed";
  opened_at: ISODateTime; opening_cash_minor: MoneyMinor;
};
type DiscountInput = { type: "amount" | "percent"; value: number; reason?: string };
type QuoteRequest = {
  shift_id: ID;
  terminal_id: ID;
  items: Array<{ product_id: ID; quantity: string; discount?: DiscountInput }>;
  cart_discount?: DiscountInput;
  coupon_code?: string;
};
type QuoteResponse = {
  quote_id?: ID;
  expires_at?: ISODateTime;
  lines: Array<{
    product_id: ID; quantity: string; unit_price_minor: MoneyMinor;
    discount_minor: MoneyMinor; tax_minor: MoneyMinor; line_total_minor: MoneyMinor;
  }>;
  subtotal_minor: MoneyMinor;
  discount_minor: MoneyMinor;
  tax_minor: MoneyMinor;
  total_minor: MoneyMinor;
  warnings?: Array<{ code: string; message: string }>;
};
```

### 9.3 Tabel pemetaan API

> Payload berikut adalah **usulan kontrak frontend**, bukan klaim bahwa backend sudah menerapkan bentuk persis ini. Selaraskan dengan handler/OpenAPI aktual.

| Nama Fitur UI | HTTP Method & Path | Payload Request (TypeScript interface ringkas) | Ekspektasi Respons & State Update |
|---|---|---|---|
| Login | `POST /api/v1/auth/login` | `interface LoginRequest { username: string; password: string }` | `data: { access_token: string; refresh_token?: string; user?: UserSummary }`; simpan sesi sesuai kebijakan lalu panggil `/auth/me`. |
| Profil & permissions | `GET /api/v1/auth/me` | Tanpa body | `data: UserSummary`; isi session store dan permission guard. |
| Daftar cabang | `GET /api/v1/stores` | Query `?page=1&page_size=20&search=` | `data: Store[]` + meta; isi cache `['stores']`. |
| Buat cabang | `POST /api/v1/stores` | `interface CreateStoreRequest { name: string; code: string; address?: string }` | `data: Store`; invalidate daftar cabang dan tampilkan hasil. |
| Daftar gudang cabang | `GET /api/v1/stores/:store_id/warehouses` | Tanpa body | `data: Array<{ id: ID; name: string; active: boolean }>`; update pilihan gudang. |
| Daftar terminal cabang | `GET /api/v1/stores/:store_id/terminals` | Tanpa body | `data: Array<{ id: ID; name: string; active: boolean }>`; update pilihan terminal. |
| Daftar produk | `GET /api/v1/products` | Query `?search=&category_id=&active=&page=&page_size=` | `data: Product[]` + meta; update tabel/katalog dengan cache store-scoped. |
| Buat produk | `POST /api/v1/products` | `interface CreateProductRequest { sku: string; name: string; barcode?: string; unit: string; active: boolean; category_id?: ID }` | `data: Product`; reset form, toast, invalidate produk/katalog. |
| Override harga cabang | `POST /api/v1/stores/:store_id/products/:id/price` | `interface SetStorePriceRequest { price_minor: MoneyMinor; effective_at?: ISODateTime; reason?: string }` | `data: { product_id: ID; store_id: ID; price_minor: MoneyMinor; version?: string }`; invalidate harga/katalog dan quote ulang cart. |
| Penyesuaian stok | `POST /api/v1/inventory/adjust` | `interface InventoryAdjustRequest { store_id: ID; warehouse_id: ID; product_id: ID; quantity_delta: string; reason: string; reference?: string }` | `data: { ledger_id: ID; stock_after?: string; approval_required?: boolean }`; invalidate stok/ledger; bila perlu tampilkan status approval. |
| Ledger inventori | `GET /api/v1/inventory/ledgers` | Query `?store_id=&warehouse_id=&product_id=&from=&to=&page=` | `data: InventoryLedger[]` + meta; update tabel ledger. |
| Shift aktif | `GET /api/v1/shifts/active` | Tanpa body; store header wajib | `data: Shift \| null` (respons kosong harus didefinisikan); update shift store/route guard. |
| Buka shift | `POST /api/v1/shifts/open` | `interface OpenShiftRequest { terminal_id: ID; opening_cash_minor: MoneyMinor; note?: string }` | `data: Shift`; update shift aktif dan invalidate ringkasan. |
| Tutup shift | `POST /api/v1/shifts/:id/close` | `interface CloseShiftRequest { actual_cash_minor: MoneyMinor; discrepancy_reason?: string; note?: string }` | `data: { shift: Shift; expected_cash_minor: MoneyMinor; actual_cash_minor: MoneyMinor; variance_minor: MoneyMinor }`; tandai closed dan blokir checkout. |
| Kas masuk/keluar | `POST /api/v1/shifts/:id/movements` | `interface ShiftMovementRequest { type: "cash_in" \| "cash_out"; amount_minor: MoneyMinor; reason: string; reference?: string }` | `data: { id: ID; type: string; amount_minor: MoneyMinor; created_at: ISODateTime }`; invalidate ringkasan/movement shift. |
| Quote cart | `POST /api/v1/pos/quote` | `QuoteRequest` | `QuoteResponse`; update total/rincian, warning stok/harga dan expiry. |
| Checkout | `POST /api/v1/pos/checkout` | `interface CheckoutRequest { quote_id?: ID; shift_id: ID; terminal_id: ID; items: QuoteRequest["items"]; payment: { method: "cash"; received_minor: MoneyMinor }; idempotency_key: string; client_transaction_id?: ID }` | `data: { transaction_id: ID; receipt_number: string; total_minor: MoneyMinor; change_minor: MoneyMinor; status: string }`; simpan transaksi, kosongkan cart setelah sukses, tampilkan struk, invalidate transaksi/laporan/shift. |
| Riwayat transaksi | `GET /api/v1/pos/transactions` | Query `?from=&to=&status=&cashier_id=&page=&page_size=` | `data: TransactionSummary[]` + meta; update tabel riwayat. |
| Void transaksi | `POST /api/v1/pos/transactions/:id/void` | `interface VoidRequest { reason: string; approval_id?: ID; idempotency_key: string }` | `data: { transaction_id: ID; status: "voided" \| "pending_approval"; approval_id?: ID }`; invalidate transaksi/ledger/shift/laporan terkait. |
| Refund transaksi | `POST /api/v1/pos/transactions/:id/refund` | `interface RefundRequest { reason: string; items?: Array<{ product_id: ID; quantity: string }>; amount_minor?: MoneyMinor; approval_id?: ID; idempotency_key: string }` | `data: { refund_id: ID; status: string; amount_minor: MoneyMinor; approval_id?: ID }`; invalidate transaksi/laporan/shift/ledger. |
| Buat approval | `POST /api/v1/approvals` | `interface CreateApprovalRequest { action_type: string; resource_type: string; resource_id?: ID; reason: string; payload: Record<string, unknown> }` | `data: { id: ID; status: "pending"; action_type: string }`; tampilkan ID/status dan invalidate daftar approval. |
| Review approval | `POST /api/v1/approvals/:id/review` | `interface ReviewApprovalRequest { decision: "approve" \| "reject"; note?: string }` | `data: { id: ID; status: "approved" \| "rejected" }`; invalidate daftar/detail dan tampilkan hasil. |
| Batch sync offline | `POST /api/v1/sync/batch` | `interface SyncBatchRequest { device_id: ID; batch_id: ID; payload_version: number; transactions: OfflineTransaction[] }` | `data: { results: Array<{ client_transaction_id: ID; status: "synced" \| "duplicate" \| "conflict" \| "rejected"; transaction_id?: ID; code?: string; message?: string }> }`; update IndexedDB, pending counter, konflik, dan status struk. |
| Laporan penjualan | `GET /api/v1/reports/sales` | Query `?from=&to=&store_id=&terminal_id=&group_by=day` | `data: { totals: Record<string, MoneyMinor>; series: Array<{ period: string; total_minor: MoneyMinor; count: number }> }`; update KPI/grafik. |
| Produk terlaris | `GET /api/v1/reports/top-products` | Query `?from=&to=&store_id=&limit=10` | `data: Array<{ product_id: ID; name: string; quantity_sold: string; sales_minor: MoneyMinor }>`; update tabel/grafik. |

### 9.4 Kontrak tambahan yang direkomendasikan

Endpoint berikut belum ada secara eksplisit tetapi dibutuhkan beberapa layar:
- `GET /api/v1/inventory/stock` untuk stok terkini.
- `GET /api/v1/shifts/:id` atau ringkasan shift untuk rekonsiliasi.
- `GET /api/v1/approvals` dan `GET /api/v1/approvals/:id` untuk daftar/detail approval.
- `GET /api/v1/pos/transactions/:id` untuk detail transaksi dan pemeriksaan status setelah timeout.
- Endpoint struk jika tidak disertakan pada detail transaksi.
- Endpoint kategori produk.
- Kontrak kupon/diskon bila kode kupon divalidasi server.
- Kontrak refresh/logout token jika memakai refresh token.
- Endpoint health/connectivity jika dibutuhkan.
- OpenAPI dengan format error, pagination, idempotensi, dan aturan `X-Store-ID`.

---

## 10. HTTP Client, Interceptor, dan Error Handling

### 10.1 API client

Semua request melewati satu client yang mengatur token, store ID, timeout, error, dan idempotency. Contoh pseudocode:

```ts
type RequestOptions = RequestInit & {
  storeScoped?: boolean;
  idempotencyKey?: string;
};

async function apiRequest<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const token = sessionProvider.getAccessToken();
  const storeId = storeProvider.getActiveStoreId();
  const headers = new Headers(options.headers);

  headers.set("Accept", "application/json");
  if (options.body) headers.set("Content-Type", "application/json");
  if (token) headers.set("Authorization", `Bearer ${token}`);
  if (options.storeScoped !== false && storeId) {
    headers.set("X-Store-ID", storeId);
  }
  if (options.idempotencyKey) {
    headers.set("Idempotency-Key", options.idempotencyKey);
  }

  const response = await fetch(`/api/v1${path}`, {
    ...options,
    headers,
  });
  const body = response.status === 204 ? null : await response.json().catch(() => null);
  if (!response.ok) throw normalizeApiError(response.status, body);
  return body as T;
}
```

Implementasi nyata harus mendukung `AbortSignal`, timeout, `204 No Content`, error jaringan/timeout/validasi/izin/konflik/server, dan tidak boleh mencetak token atau data sensitif. Jangan otomatis retry checkout atau mutasi yang tidak idempoten.

### 10.2 Error UX

| Kondisi | Perilaku |
|---|---|
| `400/422` | Tampilkan error field/alert; jangan retry otomatis |
| `401` | Bersihkan sesi sesuai kebijakan dan arahkan login |
| `403` | Tampilkan akses ditolak; jangan retry |
| `404` | Tampilkan data tidak ditemukan |
| `409` | Refresh data terkait dan tampilkan konflik |
| `429` | Hormati `Retry-After` |
| `5xx` | Pesan gangguan server; retry hanya untuk operasi aman |
| Timeout checkout | Status tidak diketahui; periksa idempotency/status, jangan buat transaksi baru |
| Offline | Cache untuk baca yang diizinkan; checkout mengikuti aturan offline tunai |

---

## 11. Keamanan Frontend

- Frontend bukan batas keamanan; backend memvalidasi permission dan store scope.
- Jangan hardcode secret API di bundle.
- Hindari render HTML mentah; sanitasi konten bila diperlukan.
- Gunakan CSP dan HTTPS.
- Jangan menulis token/password/data pembayaran ke console, analytics, atau error tracking.
- Logout di perangkat bersama harus membersihkan state pengguna dan cache sensitif sesuai policy.
- Jangan menghapus transaksi offline pending tanpa alur supervisor/audit.
- Permission yang tidak dimuat berarti aksi sensitif disabled/hidden.
- Tindakan kritis meminta alasan, memakai idempotency key, dan menampilkan hasil backend.

---

## 12. Non-Functional Requirements (NFR)

| Kategori | Target |
|---|---|
| Performa POS | Penambahan item dari barcode ke cart terasa instan; target UI lokal p95 < 100 ms pada perangkat target |
| Respons API | Feedback segera; target API ditetapkan bersama backend |
| Keandalan checkout | Tidak ada double checkout akibat double click/retry; idempotensi wajib |
| Offline | Queue bertahan setelah refresh, tab ditutup, atau browser restart |
| Konsistensi | Total final selalu mengikuti quote/checkout backend |
| Aksesibilitas | Fokus terlihat, label jelas, dialog dapat dioperasikan keyboard |
| Observability | Error UI menyertakan request ID bila tersedia |
| Browser | Tetapkan versi Chrome/Edge untuk perangkat kasir |
| Security | HTTPS, otorisasi backend, token tidak ada di log |
| Maintainability | TypeScript strict, lint, unit test aturan kritis, E2E alur checkout |

---

## 13. Testing dan Acceptance Criteria

### 13.1 Unit/component tests

- Format uang, kembalian, preview pajak.
- Store/shift guards dan permission gate.
- Barcode input/keyboard shortcuts, termasuk ketika modal/input aktif.
- Cart store: tambah, ubah jumlah, hapus, clear dengan konfirmasi.
- Offline queue: persist, retry, dedupe, konflik.
- Normalisasi error API.

### 13.2 Integration tests

- Login → `me` → pilih store → load terminal.
- Buka shift → quote → checkout → struk.
- Kas masuk/keluar → tutup shift dengan selisih.
- Override harga → quote ulang cart.
- Adjustment stok → ledger diperbarui.
- Approval dibuat → reviewer memutuskan → status berubah.
- Offline checkout → reload → online → batch sync → queue synced.
- Timeout checkout tidak membuat transaksi ganda.

### 13.3 E2E acceptance criteria

1. Kasir tanpa shift aktif tidak dapat checkout dan diarahkan membuka shift.
2. `X-Store-ID` selalu cocok dengan cabang aktif.
3. Barcode valid menambah produk satu kali.
4. Total, diskon, pajak, dan kembalian final berasal dari backend.
5. Double-click bayar tidak membuat dua transaksi.
6. Timeout checkout menampilkan status belum diketahui dan tidak membuat idempotency key baru untuk transaksi yang sama.
7. Transaksi offline tersimpan di IndexedDB sebelum dinyatakan tersimpan lokal.
8. Queue bertahan setelah refresh dan tiap item memiliki status sync.
9. Konflik sync tidak hilang dan dapat ditinjau sesuai izin.
10. Pengguna tanpa izin tidak dapat melakukan tindakan melalui UI; backend tetap harus menolak request langsung.
11. Tutup shift menampilkan kas ekspektasi, aktual, selisih, dan hasil final backend.
12. Void/refund/stock adjustment mencatat alasan dan status approval.
13. Semua layar data memiliki loading, empty, error, forbidden, dan stale/offline state yang relevan.
14. Perpindahan store membersihkan state yang tidak kompatibel.
15. Print view tidak menampilkan navigasi/tombol aplikasi.

---

## 14. Prioritas Implementasi

### Phase 0 — Kontrak dan fondasi
- Finalisasi OpenAPI/schema respons.
- Definisi token, permission, `X-Store-ID`, error format, pagination.
- Setup Vite, TypeScript strict, Tailwind, shadcn/ui, router, Query, Zustand.
- API client, route guards, layout, design tokens.

### Phase 1 — Auth, store, shift
- Login/me, pemilihan store/terminal/warehouse.
- Shift: open, movement, close.
- Permission-aware navigation dan state UI.

### Phase 2 — POS online
- Katalog/search/barcode, cart, quote, checkout tunai.
- Shortcut, pembayaran, struk/print.
- Idempotency dan timeout handling.

### Phase 3 — Backoffice
- Master produk dan harga cabang.
- Inventory adjustment dan ledger.
- Transaksi, void/refund, approvals.
- Reports sales/top products.

### Phase 4 — Offline
- IndexedDB schema dan cache katalog.
- Queue transaksi tunai, indikator jaringan, batch sync.
- Konflik, retry, supervisor workflow, telemetry.

### Phase 5 — Hardening
- E2E dan simulasi jaringan terputus.
- Audit aksesibilitas/performa di perangkat target.
- Security review dan uji restore queue.
- Pilot satu cabang sebelum rollout.

---

## 15. Definition of Done untuk AI Coding Agent

Setiap fitur selesai jika:
1. Mengikuti struktur feature-first dan TypeScript strict.
2. Tidak membuat endpoint/field/business rule yang belum disepakati tanpa menandai asumsi.
3. Semua request melalui API client terpusat.
4. Semua mutasi memiliki loading, success, validation error, network error, dan conflict handling yang relevan.
5. Permission gate untuk UX tidak dianggap menggantikan backend authorization.
6. Query keys menyertakan `storeId` untuk data store-scoped.
7. Checkout tidak memiliki retry otomatis yang berpotensi duplikasi.
8. Uang tidak dihitung dengan floating-point.
9. Stok, harga final, pajak, diskon, dan status transaksi bersumber dari backend.
10. Offline transaction tersimpan persisten sebelum UI menyatakan tersimpan lokal.
11. Unit test untuk logika kritis dan E2E untuk transaksi.
12. Tidak ada token/password/data pembayaran sensitif di log.
13. Dokumentasi env dan cara menjalankan aplikasi diperbarui.
14. `npm run lint`, `npm run typecheck`, dan `npm run test` lulus atau kegagalan dijelaskan.

---

## 16. Keputusan yang Harus Difinalisasi Sebelum Produksi

- Cookie HttpOnly atau Bearer token yang dibaca frontend.
- Schema respons dan error API.
- Apakah `GET /stores` difilter berdasarkan akses pengguna.
- Endpoint kategori dan stok terkini.
- Ringkasan shift untuk rekonsiliasi.
- Dukungan `Idempotency-Key` dan status lookup setelah timeout.
- Schema `POST /sync/batch`, hasil per transaksi, dan resolusi konflik.
- Masa berlaku cache harga/stok.
- Kebijakan offline untuk shift, terminal, stok terbatas, dan uang tunai yang sudah diterima.
- Model printer thermal untuk perangkat target.
- Kebijakan refund parsial, void, approval, dan self-approval.
- Dukungan multi-tab dan pencegahan dua tab mengubah cart/shift bersamaan.

---

## 17. Glossary

- **Store/Outlet:** cabang retail.
- **Terminal:** perangkat/register POS.
- **Shift:** sesi kasir dengan modal awal dan rekonsiliasi akhir.
- **Quote:** kalkulasi transaksi yang divalidasi backend sebelum checkout.
- **Idempotency key:** identitas unik agar request yang diulang tidak menggandakan hasil.
- **Offline queue:** transaksi lokal yang menunggu sinkronisasi.
- **Ledger:** catatan mutasi stok atau uang yang dapat diaudit.
- **RBAC:** kontrol akses berbasis role/permission.
- **Optimistic UI:** update tampilan sebelum server mengonfirmasi; harus hati-hati untuk stok/transaksi finansial.

---

**Akhir dokumen.** Dokumen ini adalah PRD frontend dan kontrak implementasi awal. Cocokkan semua interface API dengan spesifikasi backend Go Fiber yang benar-benar tersedia sebelum coding produksi.
