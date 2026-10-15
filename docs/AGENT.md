# Agent Rules — Go Fiber POS Backend

## 1. Tujuan dan Prioritas

Dokumen ini adalah aturan kerja untuk AI coding agent dan developer pada backend POS/inventory multi-cabang.

Urutan acuan jika ada konflik:
1. Keputusan arsitektur yang disepakati dalam TDD.
2. Kebutuhan produk pada `prd.md`.
3. Aturan teknis dalam file ini.
4. Preferensi implementasi sementara.

Jangan mengubah arsitektur, library inti, atau aturan bisnis secara diam-diam. Jika perubahan diperlukan, jelaskan alasan dan dampaknya terlebih dahulu.

## 2. Tech Stack

- Bahasa: Go 1.24+.
- HTTP framework: `github.com/gofiber/fiber/v2`.
- Database: PostgreSQL.
- Database access: `github.com/jackc/pgx/v5` dan `pgxpool` sesuai keputusan TDD.
- Konfigurasi: melalui environment variables; jangan hardcode credential atau secret.
- API: REST/JSON.
- Database schema: dikelola melalui migration versioned, bukan perubahan manual yang tidak tercatat.

**Catatan:** TDD menetapkan `pgx/v5` + `pgxpool`, bukan GORM. Jangan menambahkan `gorm.io/gorm` atau mengganti layer database ke ORM tanpa persetujuan eksplisit dan pembaruan TDD.

## 3. Struktur Proyek

Gunakan modular layered architecture. Struktur awal yang disarankan:

```text
.
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── config/
│   ├── handler/
│   ├── middleware/
│   ├── model/
│   ├── repository/
│   ├── route/
│   ├── service/
│   ├── validation/
│   └── platform/
├── migrations/
├── tests/
├── .env.example
├── go.mod
└── README.md
```

- `handler`: HTTP request/response; jangan menaruh aturan bisnis utama di sini.
- `service`: aturan bisnis dan orkestrasi use case.
- `repository`: query dan akses database.
- `model`: tipe domain/request/response yang sesuai; jangan mengekspos struktur internal database tanpa pertimbangan.
- `middleware`: autentikasi, otorisasi, request ID, dan logging.
- `config`: parsing dan validasi konfigurasi.
- `route`: deklarasi endpoint dan middleware.
- `migrations`: perubahan schema yang berversi dan dapat ditinjau.

Jaga dependensi mengarah ke dalam: handler memanggil service, service menggunakan repository/interface yang diperlukan. Hindari import cycle dan package serba-guna yang terlalu besar.

## 4. Database dan Transaksi

- PostgreSQL adalah sumber kebenaran untuk transaksi online.
- Gunakan parameterized queries; jangan menyusun SQL dengan menggabungkan input pengguna.
- Gunakan foreign key, unique constraint, check constraint, dan index bila relevan.
- Semua mutasi yang saling bergantung harus dilakukan dalam satu transaksi database.
- Untuk checkout, mutasi stok, ledger, item transaksi, pembayaran, dan data transaksi terkait harus konsisten secara atomik.
- Gunakan `pgx.Tx` dengan pola `Begin`/`Commit`/`Rollback` atau helper transaksi repository yang sudah ada. Selalu rollback bila operasi gagal; jangan commit jika ada error.
- Lock saldo stok atau gunakan conditional atomic update untuk mencegah dua checkout menjual unit stok terakhir secara bersamaan.
- Jika memproses beberapa SKU, kunci dalam urutan deterministik untuk mengurangi risiko deadlock.
- Jangan pernah mengubah atau menghapus histori ledger untuk memperbaiki transaksi; gunakan mutasi kompensasi yang dapat diaudit.
- Buat migration baru untuk perubahan schema; jangan mengedit migration yang sudah dipakai di lingkungan bersama/produksi.
- Jangan mengandalkan validasi aplikasi saja untuk invariant yang dapat ditegakkan oleh database.

## 5. Aturan Bisnis yang Tidak Boleh Dilanggar

- Checkout online harus ditolak jika stok tidak mencukupi; stok tidak boleh negatif.
- Harga bersifat tax-inclusive sesuai PRD dan kalkulasi uang harus deterministik.
- Harga dapat berbeda per cabang. Simpan snapshot harga, diskon, pajak, dan total pada transaksi.
- Gunakan idempotency key untuk checkout dan endpoint yang berpotensi diulang.
- Request yang dikirim ulang dengan key sama dan payload sama tidak boleh membuat transaksi duplikat. Key sama dengan payload berbeda harus menghasilkan konflik.
- Void dan refund adalah operasi kompensasi; jangan menghapus transaksi asli.
- Tindakan yang memerlukan approval tidak boleh dieksekusi sebelum disetujui.
- Pemohon tidak boleh menyetujui permintaannya sendiri.
- Setiap mutasi penting harus menghasilkan audit trail.
- Akses data harus memvalidasi permission dan cakupan cabang di backend pada setiap request yang relevan.
- Mode offline hanya mendukung pembayaran tunai. Jangan mengklaim stok global tetap konsisten secara real-time saat terminal terputus.

## 6. Format API dan Error Handling

Gunakan JSON dan format error yang konsisten:

```json
{
  "error": "Stok tidak mencukupi",
  "code": "INSUFFICIENT_STOCK"
}
```

- `error`: pesan yang aman dan dapat dipahami pengguna.
- `code`: kode stabil yang dapat dipakai frontend untuk menentukan penanganan.
- Gunakan HTTP status yang sesuai: `400` untuk input tidak valid, `401` untuk belum terautentikasi, `403` untuk tidak berizin, `404` untuk resource yang tidak ditemukan/ tidak boleh diketahui, `409` untuk konflik, dan `500` untuk kegagalan internal.
- Jangan mengirim stack trace, SQL, credential, token, atau detail internal pada response.
- Validasi input di batas API dan ulangi validasi invariant penting di service/database.
- Jangan mengubah bentuk response API yang sudah digunakan frontend tanpa memperbarui kontrak dan tes.

## 7. Uang, Kuantitas, dan Waktu

- Jangan gunakan `float32` atau `float64` untuk nilai uang.
- Gunakan integer dengan satuan yang ditetapkan secara konsisten atau tipe decimal yang sesuai. Untuk Rupiah, tetapkan satu konvensi dan dokumentasikan; jangan mencampur Rupiah utuh dengan satuan pecahan.
- Kuantitas stok dapat memerlukan pecahan untuk produk timbang; tentukan precision eksplisit dan jangan menggunakan floating point tanpa aturan pembulatan yang jelas.
- Gunakan pembulatan yang konsisten untuk harga, diskon, pajak, total transaksi, struk, dan laporan.
- Simpan waktu dengan tipe yang sesuai dan gunakan UTC di backend/database jika memungkinkan; konversikan ke zona waktu lokal hanya untuk tampilan atau kebutuhan bisnis yang eksplisit.

## 8. Keamanan

- Jangan hardcode password, API key, token, DSN, atau secret.
- Sediakan `.env.example` dengan nilai placeholder saja; jangan commit `.env`.
- Gunakan hashing password yang sesuai; jangan menyimpan password plaintext.
- Jangan percaya pada `user_id`, `store_id`, role, atau permission yang dikirim client tanpa verifikasi server.
- Validasi access token dan cakupan cabang melalui middleware/service.
- Terapkan least privilege dan hindari memberikan hak admin secara default.
- Jangan mencatat password, token, data pembayaran sensitif, atau rahasia ke log.
- Gunakan TLS di lingkungan produksi.
- Jangan menambahkan dependency baru tanpa kebutuhan yang jelas dan pemeriksaan pemeliharaan/keamanan.

## 9. Logging dan Observabilitas

- Gunakan structured logging.
- Sertakan request/correlation ID jika tersedia.
- Catat error yang berguna untuk diagnosis tanpa membocorkan data sensitif.
- Endpoint health/readiness tidak boleh membocorkan konfigurasi internal.
- Kegagalan checkout, sinkronisasi offline, worker, dan webhook harus dapat dipantau.

## 10. Testing dan Definition of Done

Setiap fitur harus menyertakan tes yang relevan.

Minimal:
- Unit test untuk aturan bisnis dan perhitungan.
- Integration test untuk query, transaksi database, locking, dan constraint.
- API test untuk status code serta kontrak response.
- Authorization test untuk permission dan batas cabang.
- Concurrency/idempotency test untuk checkout dan retry.
- Offline sync test untuk duplikasi, retry, dan konflik jika fitur terkait berubah.

Sebelum menyatakan pekerjaan selesai:
1. Jalankan `gofmt` pada file Go yang berubah.
2. Jalankan `go test ./...`.
3. Jalankan pemeriksaan statis/linter yang telah dikonfigurasi dalam proyek.
4. Pastikan migration dapat diterapkan dan rollback-nya dipahami.
5. Periksa bahwa tidak ada secret atau data sensitif yang masuk ke repository.
6. Ringkas file yang berubah, keputusan penting, tes yang dijalankan, dan keterbatasan yang masih ada.

Jangan mengklaim tes berhasil jika belum benar-benar dijalankan.

## 11. Cara AI Agent Bekerja

- Baca `prd.md`, TDD, migration, dan pola kode yang sudah ada sebelum mengimplementasikan fitur.
- Kerjakan perubahan sekecil mungkin dan sesuai scope tugas.
- Ikuti konvensi penamaan dan pola yang sudah dipakai di repository.
- Jangan membuat endpoint, tabel, status transaksi, atau permission baru tanpa kebutuhan yang jelas.
- Jangan mengganti dependency atau melakukan refactor besar sebagai efek samping tugas kecil.
- Jika PRD/TDD ambigu atau bertentangan, sebutkan konflik dan ajukan pilihan; jangan menebak aturan bisnis berisiko.
- Untuk perubahan database, sertakan migration dan tes integrasi yang relevan.
- Untuk perubahan API, perbarui dokumentasi/kontrak dan tes.
- Berikan ringkasan implementasi, asumsi, risiko, dan cara menjalankan tes pada akhir pekerjaan.
