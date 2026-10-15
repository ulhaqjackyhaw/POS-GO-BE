# Product Requirements Document (PRD)
## Sistem POS, Manajemen Inventori, dan Operasional Multi-Cabang

**Versi:** 1.0  
**Status:** Draft untuk validasi produk  
**Tanggal:** 15 Oktober 2026

---

## 1. Ringkasan Produk

Produk ini adalah sistem Point of Sale (POS) dan manajemen inventori untuk bisnis retail yang memiliki satu atau lebih cabang. Sistem membantu pemilik dan staf mengelola katalog produk, harga, stok, transaksi penjualan, kasir, perpindahan stok antargudang/cabang, persetujuan tindakan kritis, serta laporan operasional.

Sistem dirancang dengan pendekatan **single-tenant per instalasi/akun bisnis**, backend API terpusat, aplikasi web untuk administrasi, serta antarmuka POS yang mendukung transaksi tunai saat koneksi internet terputus melalui mekanisme offline queue dan sinkronisasi.

### 1.1 Keputusan produk utama

- Model bisnis: satu tenant/perusahaan per deployment atau ruang kerja bisnis.
- Kanal: aplikasi web untuk administrasi dan POS; aplikasi mobile dapat digunakan untuk kebutuhan operasional yang sesuai.
- Pembayaran offline: hanya tunai.
- Pembayaran non-tunai: membutuhkan koneksi dan konfirmasi dari integrasi pembayaran.
- Stok tidak boleh negatif pada transaksi online.
- Checkout diblokir apabila stok tidak mencukupi.
- Harga jual bersifat termasuk pajak (*tax-inclusive*).
- Harga dapat berbeda per cabang.
- Tindakan kritis dapat memerlukan persetujuan pengguna lain.
- Perubahan stok dan transaksi keuangan harus memiliki jejak audit.

## 2. Latar Belakang dan Masalah

Bisnis retail multi-cabang sering menghadapi beberapa masalah:
- Data produk, harga, dan stok tersebar di banyak file atau aplikasi.
- Pemilik sulit melihat kondisi operasional setiap cabang secara konsisten.
- Stok dapat tidak akurat akibat penjualan, retur, penyesuaian, atau transfer yang tidak tercatat dengan baik.
- Harga yang berbeda antar-cabang sulit dikelola.
- Pembatalan transaksi, diskon khusus, dan penyesuaian stok berisiko disalahgunakan tanpa kontrol persetujuan.
- Gangguan internet dapat menghentikan transaksi jika POS tidak memiliki mekanisme offline.
- Rekonsiliasi kas dan laporan penjualan memerlukan pekerjaan manual.

Sistem ini bertujuan menyediakan sumber data operasional yang konsisten, kontrol akses yang jelas, dan pencatatan transaksi yang dapat ditelusuri.

## 3. Tujuan Produk dan Indikator Keberhasilan

### 3.1 Tujuan

1. Mempercepat dan menstandarkan proses penjualan di cabang.
2. Menjaga konsistensi saldo stok dan histori mutasi.
3. Memungkinkan pengelolaan harga yang fleksibel per cabang.
4. Meningkatkan kontrol terhadap aktivitas berisiko melalui RBAC, persetujuan, dan audit.
5. Memungkinkan kasir tetap menerima pembayaran tunai ketika koneksi terputus, dengan batasan yang transparan.
6. Menyediakan laporan penjualan, stok, dan kas yang mudah dipahami.

### 3.2 KPI awal

Target berikut adalah sasaran awal yang perlu divalidasi melalui pilot:
- Waktu checkout online untuk transaksi standar: persentil ke-95 di bawah 2 detik, tidak termasuk waktu respons penyedia pembayaran.
- Tidak ada stok negatif yang disebabkan oleh checkout online.
- Setiap transaksi penjualan yang berhasil memiliki item transaksi, pembayaran, dan mutasi stok yang konsisten.
- 100% tindakan kritis tercatat dalam audit log.
- 100% permintaan sinkronisasi yang dikirim ulang dengan idempotency key yang sama tidak membuat transaksi duplikat.
- Selisih kas dapat ditampilkan dan direkonsiliasi pada penutupan shift.
- Konflik transaksi offline ditampilkan secara eksplisit dan tidak diubah diam-diam.

## 4. Ruang Lingkup

### 4.1 Termasuk dalam MVP

- Autentikasi dan manajemen pengguna.
- Role-based access control (RBAC) dan pembatasan akses berdasarkan cabang.
- Pengelolaan cabang, gudang, dan terminal POS.
- Katalog produk, SKU, barcode, varian, status aktif/nonaktif.
- Pengaturan harga default dan override harga per cabang.
- Saldo stok per lokasi dan ledger mutasi stok.
- Penerimaan stok, penyesuaian stok, dan transfer stok.
- Shift kasir, modal awal, pergerakan kas, dan penutupan shift.
- Pembuatan transaksi, perhitungan pajak termasuk pajak, diskon sesuai izin, pembayaran tunai, dan struk.
- Pembatalan/void dan proses refund dengan kontrol akses.
- Persetujuan untuk tindakan kritis.
- Audit log.
- POS offline terbatas untuk pembayaran tunai, dengan antrean lokal dan sinkronisasi.
- Laporan dasar penjualan, produk terjual, stok, mutasi, dan rekonsiliasi kas.
- Monitoring, backup, serta pemulihan data untuk operasional produksi.

### 4.2 Tidak termasuk dalam MVP

- Akuntansi umum dan buku besar penuh.
- Payroll dan HR.
- Program loyalitas, poin, kupon kompleks, dan CRM lanjutan.
- Marketplace dan e-commerce publik.
- Pengadaan otomatis berbasis prediksi.
- Multi-tenant SaaS dengan isolasi banyak perusahaan dalam satu deployment, kecuali diputuskan pada fase lanjutan.
- Pembayaran non-tunai ketika offline.
- Sinkronisasi stok global real-time saat terminal benar-benar terputus dari server.
- Integrasi pajak atau e-faktur spesifik negara yang belum disepakati.
- Peramalan permintaan dan analitik berbasis machine learning.

## 5. Pengguna dan Peran

| Peran | Tanggung jawab utama |
|---|---|
| Owner / Business Admin | Konfigurasi bisnis, akses seluruh cabang, laporan, dan kebijakan |
| Store Manager | Operasional cabang, persetujuan tertentu, laporan cabang, stok |
| Supervisor | Mengawasi kasir, menyetujui tindakan sesuai hak akses |
| Cashier | Membuka shift, menjual produk, menerima pembayaran tunai, mencetak struk |
| Inventory Staff | Mengelola penerimaan, penyesuaian, dan transfer stok sesuai izin |
| Auditor / Read-only | Membaca laporan dan audit tanpa mengubah data |
| System Operator | Monitoring teknis, backup, dan pengelolaan deployment; tidak otomatis memiliki hak bisnis |

Hak akses harus ditentukan oleh permission, bukan hanya nama role. Penugasan pengguna ke cabang membatasi data yang dapat dibaca atau diubah.

## 6. Prinsip dan Aturan Bisnis

### 6.1 Identitas dan akses

- Pengguna harus login untuk mengakses fungsi terlindungi.
- Setiap request API memvalidasi identitas, permission, dan cakupan cabang.
- Pengguna hanya dapat mengakses cabang yang ditugaskan, kecuali memiliki permission lintas-cabang.
- Tindakan sensitif memerlukan autentikasi yang valid dan dicatat dalam audit log.
- Pengguna tidak boleh menyetujui permintaan kritis yang dibuatnya sendiri.

### 6.2 Produk dan harga

- SKU harus unik dalam lingkup bisnis.
- Barcode, jika tersedia, harus dapat dicari dengan cepat dan tidak ambigu dalam lingkup yang ditentukan.
- Produk nonaktif tidak dapat ditambahkan ke transaksi baru, tetapi tetap tampil pada histori.
- Harga dasar dapat ditimpa dengan harga khusus cabang.
- Sistem menentukan harga efektif dari harga cabang yang aktif; jika tidak tersedia, gunakan harga default.
- Perubahan harga harus memiliki waktu berlaku dan metadata pembuat.
- Perubahan harga yang berdampak besar dapat memerlukan persetujuan sesuai kebijakan bisnis.

### 6.3 Pajak dan diskon

- Harga jual yang ditampilkan dan disimpan untuk POS bersifat termasuk pajak.
- Tarif dan metode pajak ditentukan melalui konfigurasi yang berlaku.
- Untuk harga termasuk pajak dengan tarif pajak `r`, komponen pajak dihitung dari harga termasuk pajak dengan rumus: `pajak = harga termasuk pajak × r / (1 + r)`.
- Pembulatan mengikuti kebijakan mata uang yang ditetapkan dan harus konsisten antara quote, checkout, struk, serta laporan.
- Diskon hanya tersedia untuk pengguna dengan permission yang sesuai.
- Sistem menyimpan nilai sebelum diskon, diskon, pajak, dan total akhir secara terpisah.

### 6.4 Stok dan inventori

- Saldo stok disimpan per produk/varian dan lokasi stok.
- Setiap perubahan stok menghasilkan catatan ledger yang tidak diubah atau dihapus melalui operasi normal.
- Saldo stok dan ledger diperbarui dalam satu transaksi database.
- Checkout online ditolak jika stok tidak mencukupi.
- Stok tidak boleh negatif akibat transaksi online.
- Penyesuaian stok wajib memiliki alasan; penyesuaian di atas ambang batas dapat memerlukan persetujuan.
- Transfer stok memiliki status dan tahapan yang dapat ditelusuri, termasuk barang dalam perjalanan bila diperlukan.
- Pembatalan atau refund yang mengembalikan stok harus menghasilkan mutasi kompensasi, bukan menghapus histori awal.

### 6.5 Penjualan dan pembayaran

- Setiap transaksi memiliki nomor unik dan idempotency key.
- Pengulangan request checkout dengan idempotency key yang sama harus mengembalikan hasil transaksi yang sama, bukan membuat penjualan baru.
- Checkout online memvalidasi harga, status produk, izin diskon, shift aktif, dan stok sebelum commit.
- Pembayaran tunai mencatat jumlah diterima dan kembalian.
- Pembayaran non-tunai membutuhkan konfirmasi sesuai integrasi yang dipilih.
- Transaksi selesai tidak boleh diedit langsung; koreksi dilakukan melalui void/refund yang tercatat.
- Void/refund mengikuti permission dan persetujuan yang dikonfigurasi.

### 6.6 Shift kasir

- Kasir harus membuka shift sebelum melakukan penjualan jika kebijakan cabang mewajibkannya.
- Pembukaan shift mencatat saldo/modal awal.
- Pergerakan kas masuk/keluar selain penjualan harus memiliki kategori, alasan, nominal, dan pengguna.
- Penutupan shift membandingkan kas yang diharapkan dengan kas aktual.
- Selisih kas dicatat dan dapat memerlukan alasan atau persetujuan.
- Shift yang sudah ditutup tidak dapat dibuka ulang tanpa proses administratif yang diaudit.

### 6.7 Persetujuan dan audit

- Tindakan kritis dapat memerlukan approval, misalnya diskon di atas batas, void transaksi, refund, penyesuaian stok besar, atau perubahan harga tertentu.
- Pemohon dan pemberi persetujuan harus berbeda.
- Persetujuan terikat pada versi/detail permintaan yang disetujui; perubahan detail membatalkan persetujuan lama.
- Audit log mencatat aktor, waktu, tindakan, entitas, hasil, alasan, dan metadata relevan.
- Audit log tidak boleh dapat diedit melalui antarmuka bisnis biasa.

### 6.8 Mode offline

- Mode offline hanya mendukung transaksi tunai.
- Katalog dan harga terakhir yang tersinkron dapat disimpan secara lokal sesuai kebijakan keamanan perangkat.
- Transaksi offline disimpan dalam antrean lokal persisten dan diberi ID unik dari perangkat.
- Saat koneksi kembali, perangkat mengirim batch transaksi dengan idempotency key.
- Sistem membedakan transaksi tersinkron, menunggu sinkronisasi, dan konflik.
- Harga atau stok yang berubah saat offline dapat menyebabkan konflik saat sinkronisasi; konflik tidak boleh diselesaikan diam-diam.
- Offline tidak dapat menjamin ketersediaan stok global secara real-time. Untuk mengurangi overselling, bisnis dapat menetapkan kuota stok offline per terminal/SKU. Jika jaminan stok ketat diperlukan tanpa kuota, penjualan harus online-only.
- Kebijakan masa berlaku cache harga, batas antrean, dan prosedur konflik harus dapat dikonfigurasi atau didokumentasikan.

## 7. Kebutuhan Fungsional

Prioritas: **P0** = wajib untuk rilis MVP; **P1** = penting untuk rilis awal atau setelah MVP; **P2** = pengembangan lanjutan.

### FR-01 — Autentikasi dan sesi (P0)

- Pengguna dapat login dengan kredensial yang valid.
- API menerbitkan access token dan mekanisme refresh token.
- Pengguna dapat logout dan sesi dapat dicabut.
- Login gagal memberikan pesan aman tanpa membocorkan apakah username atau password tertentu benar.
- Sistem menerapkan pembatasan percobaan login dan pencatatan aktivitas autentikasi.

### FR-02 — Role dan cakupan cabang (P0)

- Admin dapat membuat role dan menghubungkan permission.
- Admin dapat menugaskan pengguna ke satu atau beberapa cabang.
- Backend memvalidasi permission dan cakupan cabang pada setiap operasi yang relevan.
- Akses lintas-cabang harus diberikan secara eksplisit.

### FR-03 — Manajemen organisasi (P0)

- Admin dapat membuat dan mengubah cabang, gudang, dan terminal POS.
- Kode cabang/gudang/terminal harus unik dalam lingkup bisnis.
- Entitas yang sudah memiliki histori transaksi tidak boleh dihapus secara destruktif; gunakan status nonaktif.

### FR-04 — Katalog produk (P0)

- Pengguna berizin dapat membuat, membaca, mengubah, dan menonaktifkan produk.
- Produk dapat memiliki SKU, barcode, nama, kategori, satuan, varian, dan status.
- POS dapat mencari produk melalui nama, SKU, atau barcode.
- Produk dan varian yang telah digunakan dalam transaksi tetap dapat ditampilkan pada histori.

### FR-05 — Harga per cabang (P0)

- Pengguna berizin dapat menetapkan harga dasar.
- Pengguna berizin dapat membuat override harga untuk cabang tertentu.
- Resolver harga memilih harga efektif berdasarkan cabang dan waktu transaksi.
- Quote dan transaksi menyimpan snapshot harga agar histori tidak berubah ketika harga katalog diperbarui.
- Periode harga yang bertumpang tindih harus ditolak atau diselesaikan secara eksplisit.

### FR-06 — Saldo dan ledger inventori (P0)

- Sistem menampilkan saldo per produk/varian dan lokasi.
- Sistem mencatat penerimaan, penjualan, retur, transfer, dan penyesuaian sebagai mutasi terpisah.
- Saldo dan ledger konsisten secara atomik.
- Penyesuaian memerlukan alasan dan identitas pengguna.
- Riwayat mutasi dapat difilter berdasarkan produk, lokasi, jenis mutasi, dan periode.

### FR-07 — Transfer stok (P1)

- Pengguna berizin dapat membuat transfer dari lokasi asal ke lokasi tujuan.
- Transfer memiliki status draft, dikirim, diterima, dibatalkan, atau status setara yang disepakati.
- Sistem memvalidasi stok saat pengiriman.
- Penerimaan mencatat jumlah aktual dan selisih.
- Setiap transisi status dicatat dan dapat diaudit.

### FR-08 — Shift kasir (P0)

- Kasir dapat membuka shift dengan modal awal.
- Sistem mencatat penjualan yang terkait dengan shift.
- Pengguna berizin dapat mencatat kas masuk/keluar non-penjualan.
- Kasir atau supervisor dapat melakukan penutupan shift sesuai permission.
- Laporan penutupan menampilkan kas yang diharapkan, kas aktual, dan selisih.

### FR-09 — Quote transaksi (P0)

- POS dapat membuat quote sebelum checkout.
- Quote menampilkan item, harga efektif, diskon, pajak termasuk pajak, subtotal, dan total.
- Backend memvalidasi ulang data penting saat checkout; quote bukan jaminan stok permanen.
- Perubahan harga atau stok harus ditangani dengan respons konflik yang jelas.

### FR-10 — Checkout tunai online (P0)

- Kasir dapat menyelesaikan transaksi tunai.
- Backend mengunci/memvalidasi saldo stok secara aman untuk mencegah dua checkout menjual stok terakhir secara bersamaan.
- Pembuatan transaksi, item, pembayaran, pengurangan stok, ledger, dan audit/outbox terkait harus berhasil atau gagal bersama-sama dalam satu batas transaksi yang sesuai.
- Checkout tidak boleh menghasilkan stok negatif.
- Request yang diulang dengan idempotency key sama tidak membuat transaksi ganda.

### FR-11 — Struk dan histori transaksi (P0)

- POS dapat menampilkan dan mencetak atau membagikan struk.
- Histori dapat dicari berdasarkan nomor transaksi, tanggal, kasir, dan cabang sesuai izin.
- Struk menampilkan item, kuantitas, harga, diskon, pajak, total, pembayaran, dan kembalian.

### FR-12 — Void dan refund (P0)

- Void/refund memerlukan permission yang sesuai.
- Alasan wajib diisi.
- Jika perlu, sistem meminta approval dari pengguna berbeda.
- Refund dicatat sebagai transaksi kompensasi dan tidak menghapus transaksi awal.
- Kebijakan pengembalian stok dan pengembalian dana ditentukan secara eksplisit.

### FR-13 — Pembayaran non-tunai (P1)

- Sistem menyediakan adapter untuk integrasi penyedia pembayaran.
- Status pembayaran diverifikasi melalui respons server dan/atau webhook yang tervalidasi.
- Callback duplikat tidak boleh menggandakan pembayaran.
- Status transaksi harus jelas untuk pembayaran tertunda, berhasil, gagal, atau dibatalkan.

### FR-14 — Approval tindakan kritis (P1)

- Sistem dapat membuat permintaan approval dengan jenis tindakan, nilai, alasan, pemohon, dan snapshot data.
- Approver yang memenuhi syarat dapat menyetujui atau menolak.
- Pemohon tidak dapat menyetujui permintaannya sendiri.
- Eksekusi hanya menggunakan detail yang disetujui.
- Keputusan approval tercatat dalam audit log.

### FR-15 — Sinkronisasi offline (P1)

- Aplikasi POS dapat menyimpan transaksi tunai saat offline.
- Setiap transaksi lokal memiliki ID perangkat dan idempotency key yang stabil.
- Sinkronisasi mendukung retry yang aman.
- Server mengembalikan status per transaksi dan alasan konflik.
- UI menampilkan jumlah transaksi menunggu, berhasil, dan konflik.
- Proses pemulihan setelah aplikasi/perangkat restart tidak menghilangkan antrean yang belum tersinkron.

### FR-16 — Laporan (P0 dasar, P1 lanjutan)

- Laporan penjualan per periode, cabang, dan kasir.
- Ringkasan jumlah transaksi, omzet bruto, diskon, pajak, refund, dan penjualan bersih.
- Produk terlaris berdasarkan kuantitas dan nilai.
- Laporan stok dan mutasi.
- Laporan penutupan shift dan selisih kas.
- Ekspor CSV dapat ditambahkan sesuai kebutuhan operasional.

### FR-17 — Audit log (P0)

- Sistem mencatat perubahan master data dan tindakan berisiko.
- Log memuat aktor, waktu, tindakan, objek, hasil, alasan, dan correlation/request ID bila tersedia.
- Pengguna bisnis tidak dapat mengubah atau menghapus audit log melalui UI biasa.

### FR-18 — Operasional sistem (P0)

- Tersedia endpoint health/readiness.
- Error API menggunakan format konsisten dan tidak mengekspos stack trace atau rahasia.
- Log terstruktur dapat dikorelasikan dengan request.
- Database memiliki prosedur backup dan uji restore.
- Rahasia tidak disimpan dalam source code.

## 8. Kebutuhan Nonfungsional

### NFR-01 — Keamanan

- Password disimpan menggunakan algoritma hashing yang sesuai dan parameter yang layak.
- Token memiliki masa berlaku dan dapat dicabut melalui mekanisme yang dirancang.
- Validasi permission dilakukan di backend, bukan hanya menyembunyikan tombol di UI.
- Semua input divalidasi; query database menggunakan parameter.
- Proteksi terhadap brute force, injection, akses lintas-cabang, dan replay request diterapkan.
- Data sensitif tidak ditulis ke log.
- Komunikasi produksi menggunakan TLS.

### NFR-02 — Konsistensi data

- PostgreSQL menjadi sumber kebenaran untuk transaksi online.
- Perubahan stok, ledger, dan transaksi terkait bersifat atomik.
- Constraint database digunakan untuk menjaga integritas.
- Proses konkurensi checkout menggunakan locking atau mekanisme atomik yang benar.
- Perbaikan data dilakukan melalui operasi kompensasi yang dapat diaudit.

### NFR-03 — Performa dan skalabilitas

- API umum ditargetkan memiliki p95 di bawah 500 ms pada beban normal, tidak termasuk layanan eksternal.
- Checkout online standar ditargetkan memiliki p95 di bawah 2 detik pada beban yang disepakati.
- Sistem harus dapat menambah kapasitas backend secara horizontal jika diperlukan.
- Laporan berat tidak boleh menghambat jalur checkout; gunakan query terindeks dan optimasi yang terukur.

### NFR-04 — Ketersediaan dan pemulihan

- Kegagalan proses tidak boleh meninggalkan transaksi setengah jadi.
- Worker dan webhook harus aman terhadap retry.
- Backup harus diuji dengan pemulihan berkala.
- Target RPO/RTO ditentukan sebelum produksi; usulan awal RPO 15 menit dan RTO 4 jam, bergantung infrastruktur dan biaya.
- Gangguan koneksi pada terminal ditangani melalui mode offline terbatas.

### NFR-05 — Usability dan aksesibilitas

- Alur scan barcode dan checkout dioptimalkan untuk penggunaan kasir yang cepat.
- Pesan error menjelaskan tindakan yang dapat dilakukan pengguna.
- UI mendukung ukuran layar yang ditargetkan.
- Status online/offline dan antrean sinkronisasi harus terlihat jelas.
- Konfirmasi digunakan untuk tindakan destruktif atau berisiko tinggi.

### NFR-06 — Observabilitas

- Sistem memiliki log terstruktur, metrik aplikasi, dan alert untuk error kritis.
- Pantau kegagalan checkout, konflik sinkronisasi, antrean worker, kegagalan webhook, serta selisih rekonsiliasi.
- Setiap request penting memiliki correlation ID.

## 9. Alur Pengguna Utama

### 9.1 Penjualan online

1. Kasir login dan memilih terminal/cabang yang diizinkan.
2. Kasir membuka shift jika belum ada shift aktif.
3. Kasir memindai barcode atau mencari produk.
4. Sistem menampilkan harga efektif untuk cabang tersebut.
5. Kasir mengatur kuantitas dan diskon sesuai izin.
6. POS meminta quote untuk menghitung total.
7. Kasir memilih pembayaran tunai dan memasukkan nominal yang diterima.
8. Backend memvalidasi shift, harga, izin, idempotency key, dan stok.
9. Backend menyimpan transaksi, pembayaran, pengurangan stok, ledger, dan audit terkait secara konsisten.
10. POS menampilkan nomor transaksi, kembalian, dan struk.

### 9.2 Penjualan offline

1. POS mendeteksi koneksi server tidak tersedia.
2. POS menggunakan katalog dan harga cache yang masih diizinkan kebijakan.
3. Kasir membuat transaksi tunai.
4. Transaksi disimpan secara persisten di perangkat dengan ID unik dan status menunggu sinkronisasi.
5. Saat koneksi kembali, POS mengirim antrean ke server.
6. Server memproses setiap transaksi secara idempotent dan memeriksa aturan konflik.
7. POS menampilkan hasil: tersinkron atau konflik yang perlu ditangani.

### 9.3 Penyesuaian stok dengan approval

1. Staf memilih lokasi, produk, kuantitas perubahan, dan alasan.
2. Jika ambang approval terpenuhi, sistem membuat permintaan menunggu persetujuan.
3. Approver lain memeriksa detail dan menyetujui atau menolak.
4. Sistem memvalidasi kembali stok dan detail permintaan.
5. Jika disetujui dan masih valid, sistem mem-posting mutasi stok.
6. Semua tahap dicatat di audit log.

### 9.4 Transfer stok

1. Staf membuat draft transfer dari lokasi asal ke lokasi tujuan.
2. Sistem memvalidasi akses dan produk.
3. Saat pengiriman, stok asal divalidasi dan dikurangi atau dipindahkan ke status transit sesuai model inventori yang dipilih.
4. Lokasi tujuan mengonfirmasi jumlah diterima.
5. Selisih dicatat dan dapat memicu approval atau investigasi.
6. Transfer selesai setelah semua item direkonsiliasi.

## 10. Entitas Data Tingkat Produk

Entitas utama yang diharapkan:
- **Business/Tenant**: ruang lingkup bisnis.
- **Store**: cabang.
- **Warehouse/Stock Location**: lokasi stok.
- **Terminal**: perangkat POS.
- **User, Role, Permission, User Store Assignment**: identitas dan akses.
- **Product, Product Variant, Category, Barcode**: katalog.
- **Store Product / Product Price**: ketersediaan dan harga per cabang.
- **Inventory Balance**: saldo stok terkini.
- **Inventory Ledger**: histori mutasi stok.
- **Stock Transfer, Stock Transfer Item**: transfer antarlokasi.
- **Cashier Shift, Cash Movement**: operasional kas.
- **Sales Transaction, Transaction Item, Payment**: penjualan.
- **Refund / Void**: transaksi kompensasi.
- **Approval Request**: permintaan dan keputusan persetujuan.
- **Audit Log**: jejak aktivitas.
- **Sync Batch / Offline Transaction Reference**: rekonsiliasi transaksi offline.
- **Outbox Event**: pengiriman event secara andal ke worker/integrasi.

Nilai uang harus memakai tipe numerik fixed precision atau satuan minor integer yang konsisten; hindari floating point untuk perhitungan uang. Setiap tabel transaksi menyimpan waktu, aktor, dan referensi bisnis yang dibutuhkan untuk audit.

## 11. Integrasi dan Antarmuka

- **Backend API:** REST/JSON.
- **Database utama:** PostgreSQL.
- **Web admin/POS:** aplikasi web.
- **Mobile:** aplikasi mobile untuk fungsi yang diprioritaskan.
- **Pembayaran:** adapter penyedia pembayaran untuk kanal non-tunai yang disepakati.
- **Struk:** printer yang didukung atau output digital.
- **Monitoring:** log, metrics, tracing/alert sesuai kebutuhan operasional.

Detail kontrak API, skema database, status transaksi, strategi locking, dan format sinkronisasi ditetapkan dalam Technical Design Document (TDD) dan API specification.

## 12. Kebijakan Error dan Konflik

API harus menggunakan kode/status yang konsisten. Contoh kategori:
- `VALIDATION_ERROR`: input tidak valid.
- `UNAUTHENTICATED`: sesi tidak valid atau belum login.
- `FORBIDDEN`: permission atau cakupan cabang tidak memadai.
- `INSUFFICIENT_STOCK`: stok tidak cukup.
- `PRICE_CHANGED`: harga berubah sejak quote.
- `SHIFT_NOT_OPEN`: tidak ada shift aktif yang memenuhi syarat.
- `IDEMPOTENCY_CONFLICT`: key yang sama digunakan dengan payload berbeda.
- `APPROVAL_REQUIRED`: tindakan memerlukan persetujuan.
- `SYNC_CONFLICT`: transaksi offline tidak dapat diterima otomatis.
- `PAYMENT_PENDING`: status pembayaran belum final.

Pesan kepada pengguna harus cukup jelas untuk tindakan berikutnya, tetapi tidak boleh mengungkap detail internal atau data yang tidak berhak dilihat pengguna.

## 13. Kriteria Penerimaan MVP

MVP dapat diterima apabila:
1. Pengguna hanya dapat mengakses fungsi dan cabang sesuai permission.
2. Produk dapat dibuat, dicari, dinonaktifkan, dan digunakan pada transaksi.
3. Harga cabang dipilih dengan benar dan snapshot harga transaksi tidak berubah setelah harga master berubah.
4. Checkout tunai online berhasil ketika stok cukup dan ditolak ketika stok tidak cukup.
5. Dua checkout paralel terhadap satu unit stok tidak dapat sama-sama sukses.
6. Pengulangan request dengan idempotency key sama tidak menggandakan transaksi.
7. Saldo stok dan ledger tetap konsisten setelah penjualan, void, refund, dan penyesuaian.
8. Shift dapat dibuka dan ditutup serta menampilkan rekonsiliasi kas.
9. Tindakan kritis yang memerlukan approval tidak dapat dieksekusi sebelum disetujui.
10. Pemohon tidak dapat menyetujui permintaannya sendiri.
11. Audit log tersedia untuk perubahan master data dan tindakan berisiko.
12. Transaksi offline tunai bertahan setelah aplikasi dimulai ulang dan dapat disinkronkan tanpa duplikasi.
13. Konflik offline ditampilkan dan memerlukan penyelesaian yang eksplisit.
14. Backup database berhasil dipulihkan pada lingkungan uji.
15. Tidak ada temuan keamanan kritis/tinggi yang belum ditangani pada rilis produksi.

## 14. Strategi Pengujian

- **Unit test:** kalkulasi harga/pajak, resolver harga, izin, status transaksi, validasi stok.
- **Integration test:** database, locking, transaksi atomik, ledger, idempotency, approval.
- **API contract test:** format request/response dan kode error.
- **Security test:** akses lintas-cabang, privilege escalation, brute force, validasi token, injection.
- **Concurrency test:** banyak checkout bersamaan terhadap stok terbatas.
- **Offline test:** restart aplikasi, retry, batch parsial, koneksi putus di tengah sinkronisasi, konflik harga/stok.
- **End-to-end test:** alur penjualan, shift, void/refund, transfer, approval, laporan.
- **Recovery test:** restore backup dan pemulihan worker/outbox.
- **Performance test:** pencarian produk, quote, checkout, dan laporan pada beban yang disepakati.

## 15. Risiko dan Mitigasi

| Risiko | Dampak | Mitigasi |
|---|---|---|
| Penjualan offline melebihi stok global | Overselling | Batasi offline tunai, kuota stok offline per terminal/SKU, atau wajib online |
| Request checkout terkirim ulang | Transaksi ganda | Idempotency key dan unique constraint |
| Dua kasir menjual stok terakhir | Stok negatif | Row lock/atomic stock update dan concurrency test |
| Penyalahgunaan void/refund/diskon | Kerugian finansial | RBAC, approval, alasan wajib, audit log |
| Harga cabang tidak konsisten | Harga salah | Resolver harga terpusat, periode efektif, snapshot transaksi |
| Callback pembayaran duplikat | Pembayaran ganda | Verifikasi signature dan idempotent webhook |
| Perbedaan kas | Kerugian atau kesalahan rekonsiliasi | Shift, cash movement, close-out, laporan selisih |
| Kehilangan data perangkat offline | Transaksi hilang | Penyimpanan lokal persisten dan pengujian restart |
| Backup tidak dapat dipulihkan | Downtime/data hilang | Uji restore terjadwal dan dokumentasi prosedur |

## 16. Rilis dan Tahapan Implementasi

### Tahap 0 — Fondasi
- Struktur proyek, konfigurasi, logging, health check.
- Migrasi database dan pipeline CI.
- Autentikasi, RBAC, cakupan cabang.
- Organisasi cabang, gudang, terminal.

### Tahap 1 — Master data dan inventori
- Katalog produk dan barcode.
- Harga default dan harga cabang.
- Inventory balance dan ledger.
- Penyesuaian stok serta audit log.

### Tahap 2 — POS online
- Shift kasir.
- Quote dan kalkulasi harga/pajak.
- Checkout tunai atomik dan idempotent.
- Struk, histori, void/refund dasar.
- Laporan dasar.

### Tahap 3 — Kontrol operasional
- Approval tindakan kritis.
- Transfer stok lengkap.
- Rekonsiliasi kas dan laporan lanjutan.

### Tahap 4 — Offline dan integrasi
- Antrean offline tunai dan sinkronisasi.
- Integrasi pembayaran non-tunai.
- Monitoring, alert, uji restore, dan pilot cabang.

Offline sync dan pembayaran non-tunai harus diuji secara terpisah dari alur checkout online inti.

## 17. Asumsi dan Keputusan yang Masih Perlu Divalidasi

Sebelum implementasi produksi, pemilik produk perlu menetapkan:
- Jenis bisnis retail, kategori produk, satuan, dan kebutuhan varian.
- Mata uang, aturan pembulatan, tarif dan perlakuan pajak yang berlaku.
- Apakah satu produk dapat memiliki banyak barcode.
- Kebijakan stok tersedia, stok rusak, stok reservasi, dan barang dalam perjalanan.
- Ambang batas diskon, void, refund, dan penyesuaian stok yang membutuhkan approval.
- Aturan pengembalian uang dan apakah refund selalu mengembalikan stok.
- Batas waktu cache harga offline dan kuota stok offline.
- Perangkat printer/barcode scanner dan platform POS yang didukung.
- Penyedia pembayaran non-tunai.
- Target RPO/RTO, jumlah cabang, terminal aktif, transaksi harian, dan retensi audit.
- Apakah model single-tenant berarti satu deployment per bisnis atau satu workspace bisnis pada satu deployment.

## 18. Glosarium

- **POS:** Point of Sale, sistem pencatatan penjualan di kasir.
- **SKU:** Stock Keeping Unit, kode identifikasi produk.
- **RBAC:** Role-Based Access Control, kontrol akses berdasarkan role dan permission.
- **Ledger:** Catatan mutasi yang menjadi histori pergerakan stok atau uang.
- **Idempotency key:** Kunci yang memastikan pengulangan request tidak membuat efek ganda.
- **Tax-inclusive:** Harga yang sudah mencakup komponen pajak.
- **Void:** Pembatalan transaksi sesuai kebijakan sebelum/ setelah penyelesaian tertentu.
- **Refund:** Pengembalian dana yang dicatat sebagai transaksi kompensasi.
- **Outbox:** Pola penyimpanan event bersama transaksi database untuk diproses secara andal.
- **Offline queue:** Antrean transaksi lokal yang menunggu sinkronisasi ke server.

---

**Dokumen ini adalah PRD produk.** Rincian implementasi seperti struktur tabel lengkap, endpoint API, locking database, token, dan strategi worker ditetapkan pada TDD serta spesifikasi API.
