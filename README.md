# Dokumentasi API Portolio Backend

Dokumen ini menjelaskan tujuan, cara penggunaan, alasan penciptaan, cara kerja, serta contoh respons API dari sistem Portolio Backend.

## Tujuan

Sistem Portolio Backend adalah sebuah API RESTful yang dirancang untuk mendukung fungsionalitas inti dari sebuah platform e-commerce atau marketplace. Tujuannya adalah untuk menyediakan layanan backend yang robust, skalabel, dan aman untuk mengelola:

1.  **Manajemen Akun Pengguna:** Pendaftaran, login, pengelolaan profil, dan manajemen saldo (top-up, penarikan).
2.  **Manajemen Produk:** Pembuatan, pembaruan, penghapusan, dan daftar produk dengan fitur visibilitas.
3.  **Manajemen Transaksi:** Proses pembelian produk, konfirmasi pengiriman oleh penjual, konfirmasi penerimaan oleh pembeli, pembatalan transaksi, serta perhitungan pajak.
4.  **Sistem Dukungan Pelanggan:** Pengelolaan tiket dukungan dan komunikasi real-time antara pengguna dan admin.
5.  **Notifikasi Real-time:** Pengiriman notifikasi kepada pengguna terkait aktivitas akun dan transaksi.
6.  **Fitur Admin:** Pengelolaan pengguna (suspensi, ban, unban, penghapusan), moderasi produk, dan pemantauan transaksi serta riwayat saldo.
7.  **Pengunggahan File:** Mendukung pengunggahan gambar dan file untuk produk dan komunikasi.

Sistem ini bertujuan untuk menjadi fondasi yang kuat bagi aplikasi frontend (web atau mobile) yang membutuhkan interaksi kompleks dengan data dan proses bisnis e-commerce.

## Cara Penggunaan

Untuk menjalankan dan berinteraksi dengan sistem Portolio Backend, ikuti langkah-langkah berikut:

### Prasyarat

*   **Go:** Versi 1.18 atau lebih tinggi.
*   **PostgreSQL:** Database PostgreSQL terinstal dan berjalan.
*   **Git:** Untuk mengkloning repositori.

### Instalasi dan Konfigurasi

1.  **Kloning Repositori:**
    ```bash
    git clone <URL_REPOSITORI_ANDA>
    cd portolio-backend
    ```
2.  **Instal Dependensi:**
    ```bash
    go mod tidy
    ```
3.  **Konfigurasi Lingkungan (`.env`):**
    Buat file `.env` di direktori akar proyek (sejajar dengan `go.mod`) dengan isi sebagai berikut. Sesuaikan nilai-nilai sesuai dengan konfigurasi database dan kebutuhan Anda:
    ```
    PORT=8080

    DB_HOST=127.0.0.1
    DB_PORT=5432
    DB_USER=postgres
    DB_PASSWORD=postgres
    DB_NAME=final_project

    JWT_SECRET=your-super-secret-key-change-this-in-production
    JWT_EXPIRE_DURATION=720h # Contoh: 30 hari (30 * 24 jam)
    JWT_ADMIN_EXPIRE_DURATION=15m # Contoh: 15 menit

    GOVT_TAX_PERCENT=0.05 # 5%
    ECOMMERCE_TAX_PERCENT=0.02 # 2%
    PENALTY_WARNING_LIMIT=3 # Jumlah peringatan sebelum akun disuspensi otomatis

    GIN_MODE=debug # atau "release" untuk produksi
    ```
4.  **Buat Database:**
    Pastikan database `final_project` (atau nama yang Anda tentukan di `DB_NAME`) sudah dibuat di PostgreSQL.

### Menjalankan Aplikasi

Jalankan aplikasi dari direktori akar proyek:

```bash
go run cmd/api/main.go
```

Jika berhasil, Anda akan melihat output seperti:
```
2023/10/27 10:00:00 🚀 Server berjalan di port :8080
2023/10/27 10:00:00 ✅ Admin user created.
2023/10/27 10:00:00 ✅ User budi@user.com created.
...
2023/10/27 10:00:00 🎉 Database seeded successfully with varied data!
```
Sistem akan secara otomatis melakukan migrasi skema database dan mengisi data awal (seeding) jika database masih kosong.

### Kredensial Default (dari Seeding)

Setelah seeding, Anda dapat login dengan kredensial berikut:

*   **Admin:**
    *   Email: `admin@admin.com`
    *   Password: `@admin123`
*   **Pengguna Biasa:**
    *   Email: `budi@user.com`
    *   Password: `@users123`
    *   Email: `citra@user.com`
    *   Password: `@users123`
    *   Dan beberapa pengguna acak lainnya (`user1@example.com`, dst.) dengan password yang sama.

### Interaksi API

Anda dapat berinteraksi dengan API menggunakan alat seperti Postman, Insomnia, atau cURL. Lihat bagian "Response dan Code" untuk detail endpoint dan format permintaan/respons.

**Catatan Penting:**
*   Untuk endpoint yang memerlukan autentikasi, sertakan header `Authorization: Bearer <token_jwt_anda>` setelah berhasil login.
*   Token admin memiliki masa berlaku yang lebih pendek dan akan diperbarui secara otomatis di setiap permintaan yang berhasil.

## Alasan Diciptakan

Portolio Backend diciptakan sebagai solusi backend komprehensif untuk memenuhi kebutuhan platform e-commerce atau marketplace modern. Beberapa alasan utama di balik penciptaan sistem ini meliputi:

1.  **Kebutuhan Fungsionalitas Lengkap:** Mengembangkan platform e-commerce membutuhkan berbagai fitur inti seperti manajemen pengguna, produk, transaksi, dan dukungan. Sistem ini mengkonsolidasikan fungsionalitas tersebut dalam satu layanan API yang terintegrasi.
2.  **Skalabilitas dan Performa:** Menggunakan Go (Golang) dan Gin framework, sistem ini dirancang untuk performa tinggi dan skalabilitas, mampu menangani banyak permintaan secara bersamaan, yang krusial untuk aplikasi e-commerce yang berkembang.
3.  **Keamanan:** Implementasi JWT untuk autentikasi pengguna, sesi admin yang terpisah, dan validasi input yang ketat membantu mengamankan data dan operasi. Fitur seperti suspensi dan ban pengguna juga mendukung moderasi platform.
4.  **Pengelolaan Data yang Efisien:** Penggunaan GORM dengan PostgreSQL memungkinkan pengelolaan data yang terstruktur dan efisien, termasuk relasi antar entitas dan transaksi database yang atomik untuk menjaga integritas data.
5.  **Pengalaman Pengguna yang Lebih Baik:** Fitur real-time seperti chat dan notifikasi melalui WebSockets meningkatkan interaksi pengguna dan memberikan informasi instan, yang esensial dalam lingkungan marketplace yang dinamis.
6.  **Kemudahan Pengembangan Frontend:** Dengan API RESTful yang terdefinisi dengan baik dan respons JSON yang konsisten, pengembang frontend dapat dengan mudah mengintegrasikan aplikasi mereka tanpa perlu khawatir tentang logika bisnis di sisi server.
7.  **Administrasi yang Kuat:** Panel admin yang terpisah memungkinkan operator platform untuk memantau dan mengelola aktivitas pengguna, produk, dan transaksi, memastikan kelancaran operasional dan kepatuhan terhadap kebijakan.

Secara keseluruhan, Portolio Backend dibangun untuk menyediakan fondasi teknologi yang solid dan fleksibel bagi pengembangan aplikasi e-commerce yang sukses.

## Cara Kerja

Portolio Backend dibangun menggunakan arsitektur API RESTful dengan Go (Golang) dan framework Gin. Berikut adalah gambaran umum cara kerjanya:

1.  **Inisialisasi Aplikasi (`main.go`):**
    *   Aplikasi dimulai dengan menginisialisasi Gin router.
    *   Koneksi ke database PostgreSQL dibuat menggunakan GORM.
    *   `AutoMigrate` dijalankan untuk memastikan skema database sesuai dengan model Go.
    *   Direktori media (`media/products`, `media/chat`, dll.) dibuat jika belum ada.
    *   Sebuah `WebsocketHub` diinisialisasi untuk mengelola koneksi WebSocket real-time.
    *   Fungsi `seed.Shop()` dipanggil untuk mengisi data awal ke database jika kosong.
    *   Semua rute API diatur menggunakan `routes.SetupRoutes()`.
    *   Server Gin mulai mendengarkan permintaan HTTP pada port yang dikonfigurasi.

2.  **Routing (`internal/routes/routers.go`):**
    *   Semua endpoint API didefinisikan di sini, dikelompokkan berdasarkan fungsionalitas (akun, toko, admin, upload, websocket).
    *   Setiap rute memetakan URL dan metode HTTP ke fungsi handler yang sesuai.
    *   Middleware diterapkan pada grup rute atau rute individual untuk menangani autentikasi, otorisasi, dan validasi status pengguna.

3.  **Middleware (`internal/middlewares/`):**
    *   **`JWTMiddleware()`:** Memverifikasi token JWT yang dikirim di header `Authorization`. Jika token valid, `UserID` dan `ROLE` pengguna disimpan di `gin.Context`. Middleware ini juga mendukung *grace period* untuk token yang hampir kedaluwarsa, memungkinkan refresh token secara otomatis.
    *   **`AdminAuth()`:** Middleware khusus untuk admin yang memverifikasi token sesi admin yang lebih ketat (termasuk hash token, IP, user agent) dan memperbarui masa berlaku sesi.
    *   **`Authorize()`:** Mengambil `UserID` dari konteks dan memuat data pengguna dari database untuk memastikan pengguna aktif dan valid.
    *   **`AuthorizeAdmin()`:** Memastikan bahwa peran pengguna yang mengakses endpoint adalah `admin`.
    *   **`CheckUserStatus()`:** Memeriksa status akun pengguna (aktif, ditangguhkan, diblokir, dihapus) dan menerapkan tindakan yang sesuai (misalnya, mengembalikan status aktif jika masa blokir habis, atau menangguhkan otomatis jika peringatan penalti melebihi batas).
    *   **`CheckRole()`:** Digunakan untuk endpoint publik yang mungkin diakses oleh pengguna yang tidak login (guest) atau login, untuk menentukan peran mereka tanpa memaksa autentikasi.

4.  **Handler (`internal/handler/`):**
    *   Setiap handler berisi logika bisnis untuk memproses permintaan API tertentu.
    *   **Validasi Input:** Menggunakan `c.ShouldBindJSON()` untuk mengikat payload JSON ke DTO (Data Transfer Object) dan `validator.v10` untuk validasi data.
    *   **Interaksi Database:** Menggunakan GORM untuk berinteraksi dengan database (CRUD operasi, transaksi database).
    *   **Logika Bisnis:** Menerapkan aturan bisnis seperti perhitungan pajak, manajemen stok, perubahan status transaksi, dll.
    *   **Notifikasi:** Mengirim notifikasi ke pengguna lain atau admin melalui WebSocket atau menyimpan ke database.
    *   **Pengunggahan File:** Menggunakan helper `util.SaveUploadedFile` dan `util.DownloadImage` untuk menyimpan file ke sistem file lokal.

5.  **Model (`internal/model/db/` dan `internal/model/dto/`):**
    *   **`db/`:** Mendefinisikan struktur data (entitas) yang merepresentasikan tabel di database (misalnya, `User`, `Product`, `TransactionHistory`).
    *   **`dto/`:** Mendefinisikan struktur data untuk permintaan (request) dan respons (response) API. Ini memastikan format data yang konsisten antara klien dan server.

6.  **Utilitas (`internal/util/`):**
    *   **`response.go`:** Menyediakan fungsi `RespondJSON` yang konsisten untuk mengirim respons sukses atau error dalam format JSON, termasuk detail validasi error.
    *   **`jwt.go`:** Fungsi untuk menghasilkan token JWT untuk pengguna biasa dan token sesi khusus untuk admin.
    *   **`file_upload.go`:** Fungsi untuk menyimpan file yang diunggah dari permintaan HTTP atau mengunduh gambar dari URL eksternal.
    *   **`websocket.go`:** Mengelola koneksi WebSocket. `Hub` bertanggung jawab untuk mendaftarkan/melepaskan klien dan menyiarkan pesan. Fungsi `SendNotificationToUser` dan `SendToUser` digunakan oleh handler lain untuk mengirim pesan real-time.

7.  **WebSocket (`/ws` endpoints):**
    *   Klien membuat koneksi WebSocket ke endpoint `/ws/chat/:transaction_id` atau `/ws/notifications`.
    *   Setelah koneksi terjalin, klien dapat mengirim dan menerima pesan JSON secara real-time.
    *   Untuk chat, pesan disimpan ke database dan diteruskan ke penerima yang relevan melalui WebSocket.
    *   Untuk notifikasi, notifikasi yang belum dibaca akan dikirim saat koneksi dibuat, dan notifikasi baru akan dikirim secara real-time.

Secara keseluruhan, sistem ini beroperasi dengan menerima permintaan HTTP, memvalidasinya, memproses logika bisnis dengan berinteraksi dengan database, dan mengirimkan respons JSON yang terstruktur. Untuk fungsionalitas real-time, ia memanfaatkan koneksi WebSocket yang persisten.

## Response dan Code

Berikut adalah beberapa contoh respons API yang umum dari sistem ini:

### 1. Pendaftaran Pengguna Baru

*   **Endpoint:** `POST /api/account/register`
*   **Request Body:**
    ```json
    {
      "full_name": "John Doe",
      "email": "john.doe@example.com",
      "password": "Password123!"
    }
    ```
*   **HTTP Status Code:** `201 Created`
*   **Response Body:**
    ```json
    {
      "status": "success",
      "code": 201,
      "message": "Operasi berhasil.",
      "data": {
        "full_name": "John Doe",
        "email": "john.doe@example.com",
        "balance": 0,
        "created_at": "2023-10-27T10:00:00Z"
      },
      "timestamp": "2023-10-27T10:00:00Z"
    }
    ```

### 2. Login Pengguna

*   **Endpoint:** `POST /api/account/login`
*   **Request Body:**
    ```json
    {
      "email": "john.doe@example.com",
      "password": "Password123!"
    }
    ```
*   **HTTP Status Code:** `200 OK`
*   **Response Body:**
    ```json
    {
      "status": "success",
      "code": 200,
      "message": "Login berhasil! Selamat datang kembali.",
      "timestamp": "2023-10-27T10:00:00Z"
    }
    ```
    **Catatan:** Token JWT akan dikirim di header `Authorization: Bearer <token_jwt_anda>`.

### 3. Mendapatkan Saldo dan Riwayat Transaksi

*   **Endpoint:** `GET /api/account/balance`
*   **HTTP Status Code:** `200 OK`
*   **Response Body:**
    ```json
    {
      "status": "success",
      "code": 200,
      "message": "Operasi berhasil.",
      "data": {
        "balance": 500000,
        "histories": [
          {
            "id": 1,
            "created_at": "2023-10-26 14:30:00",
            "updated_at": "2023-10-26 14:30:00",
            "description": "Top Up saldo sebesar 100000",
            "amount": 100000,
            "last_balance": 400000,
            "final_balance": 500000,
            "status": "credit"
          },
          {
            "id": 2,
            "created_at": "2023-10-25 09:15:00",
            "updated_at": "2023-10-25 09:15:00",
            "description": "Pembelian 'Smartphone Terbaru X1' x1 (termasuk pajak pemerintah Rp175000)",
            "amount": -3675000,
            "last_balance": 4175000,
            "final_balance": 500000,
            "status": "debit"
          }
        ]
      },
      "timestamp": "2023-10-27T10:00:00Z"
    }
    ```

### 4. Mendapatkan Daftar Produk

*   **Endpoint:** `GET /api/shop/products`
*   **HTTP Status Code:** `200 OK`
*   **Response Body:**
    ```json
    {
      "status": "success",
      "code": 200,
      "message": "Operasi berhasil.",
      "data": [
        {
          "id": 1,
          "title": "Smartphone Terbaru X1",
          "price": 3500000,
          "stock": 15,
          "categories": ["Elektronik", "Gadget"],
          "images": [
            {
              "id": 1,
              "product_id": 1,
              "image_url": "/media/products/dummy_image1.png"
            }
          ],
          "rating": 4.5,
          "reviews": [
            {
              "user_id": 2,
              "full_name": "Citra Dewi",
              "rating": 5,
              "comment": "Sangat puas dengan smartphone ini! Cepat dan kameranya bagus."
            }
          ]
        }
      ],
      "timestamp": "2023-10-27T10:00:00Z"
    }
    ```

### 5. Membuat Produk Baru

*   **Endpoint:** `POST /api/shop/products`
*   **Request Body:**
    ```json
    {
      "title": "Smartwatch Canggih",
      "price": 1500000,
      "stock": 10,
      "visibility": "all",
      "categories": "Elektronik,Wearable",
      "images_links": [
        "https://example.com/smartwatch_front.jpg",
        "https://example.com/smartwatch_back.jpg"
      ]
    }
    ```
*   **HTTP Status Code:** `201 Created`
*   **Response Body:**
    ```json
    {
      "status": "success",
      "code": 201,
      "message": "Operasi berhasil.",
      "data": {
        "id": 10,
        "user_id": 1,
        "title": "Smartwatch Canggih",
        "price": 1500000,
        "stock": 10,
        "visibility": "all",
        "categories": ["Elektronik", "Wearable"],
        "is_active": true,
        "images": [
          {
            "id": 5,
            "product_id": 10,
            "image_url": "/media/products/a1b2c3d4e5f6.jpg"
          }
        ],
        "created_at": "2023-10-27T10:00:00Z",
        "updated_at": "2023-10-27T10:00:00Z",
        "deleted_at": null
      },
      "timestamp": "2023-10-27T10:00:00Z"
    }
    ```

### 6. Pembelian Produk

*   **Endpoint:** `POST /api/shop/purchase`
*   **Request Body:**
    ```json
    [
      {
        "product_id": 2,
        "quantity": 1
      }
    ]
    ```
*   **HTTP Status Code:** `200 OK`
*   **Response Body:**
    ```json
    {
      "status": "success",
      "code": 200,
      "message": "Pembelian berhasil! Menunggu konfirmasi penjual.",
      "timestamp": "2023-10-27T10:00:00Z"
    }
    ```

### 7. Login Admin

*   **Endpoint:** `POST /api/admin/login`
*   **Request Body:**
    ```json
    {
      "email": "admin@admin.com",
      "password": "@admin123"
    }
    ```
*   **HTTP Status Code:** `200 OK`
*   **Response Body:**
    ```json
    {
      "status": "success",
      "code": 200,
      "message": "Login admin berhasil! Selamat datang di panel admin.",
      "timestamp": "2023-10-27T10:00:00Z"
    }
    ```
    **Catatan:** Token sesi admin akan dikirim di header `Authorization: Bearer <token_sesi_admin_anda>`.

### 8. Mendapatkan Daftar Pengguna (Admin)

*   **Endpoint:** `GET /api/admin/users`
*   **HTTP Status Code:** `200 OK`
*   **Response Body:**
    ```json
    {
      "status": "success",
      "code": 200,
      "message": "Operasi berhasil.",
      "data": {
        "total_records": 15,
        "page": 1,
        "limit": 20,
        "users": [
          {
            "id": 1,
            "full_name": "Super Admin",
            "email": "admin@admin.com",
            "role": "admin",
            "balance": 1000000,
            "status": "active",
            "ban_until": null,
            "ban_reason": "",
            "penalty_warnings": 0,
            "created_at": "2023-10-27T09:00:00Z",
            "updated_at": "2023-10-27T09:00:00Z",
            "deleted_at": null
          },
          {
            "id": 2,
            "full_name": "Budi Santoso",
            "email": "budi@user.com",
            "role": "user",
            "balance": 500000,
            "status": "active",
            "ban_until": null,
            "ban_reason": "",
            "penalty_warnings": 0,
            "created_at": "2023-10-27T09:01:00Z",
            "updated_at": "2023-10-27T09:01:00Z",
            "deleted_at": null
          }
        ]
      },
      "timestamp": "2023-10-27T10:00:00Z"
    }
    ```

### 9. Mengunggah Gambar

*   **Endpoint:** `POST /api/upload/image?type=product`
*   **Request Body:** `multipart/form-data` dengan field `image` berisi file gambar.
*   **HTTP Status Code:** `200 OK`
*   **Response Body:**
    ```json
    {
      "status": "success",
      "code": 200,
      "message": "File berhasil diunggah!",
      "data": {
        "file_url": "/media/products/48111d67f4b61629.png",
        "file_name": "my_product_image.png",
        "file_size": 123456
      },
      "timestamp": "2023-10-27T10:00:00Z"
    }
    ```

### 10. Pesan WebSocket (Chat)

*   **Endpoint:** `WS /ws/chat/:transaction_id`
*   **Pesan yang Dikirim Klien (JSON):**
    ```json
    {
      "message_type": "text",
      "content": "Halo, apakah produk ini masih tersedia?"
    }
    ```
*   **Pesan yang Diterima Klien (JSON):**
    *   **Untuk Pengirim (Success):**
        ```json
        {
          "type": "success",
          "message": "Pesan berhasil dikirim dan disimpan.",
          "chat_message": {
            "id": 1,
            "transaction_id": 123,
            "sender_id": 1,
            "sender_name": "",
            "message_type": "text",
            "content": "Halo, apakah produk ini masih tersedia?",
            "file_url": "",
            "created_at": "2023-10-27T10:00:00Z"
          }
        }
        ```
    *   **Untuk Penerima (New Message):**
        ```json
        {
          "type": "chat_message",
          "message": {
            "id": 1,
            "transaction_id": 123,
            "sender_id": 1,
            "sender_name": "",
            "message_type": "text",
            "content": "Halo, apakah produk ini masih tersedia?",
            "file_url": "",
            "created_at": "2023-10-27T10:00:00Z"
          }
        }
        ```

### 11. Notifikasi WebSocket

*   **Endpoint:** `WS /ws/notifications`
*   **Pesan yang Diterima Klien (JSON):**
    ```json
    {
      "type": "notification",
      "message": {
        "id": 1,
        "user_id": 2,
        "type": "sale",
        "message": "Produk 'Smartphone Terbaru X1' Anda telah dibeli oleh John Doe (x1). Menunggu konfirmasi pengiriman.",
        "related_id": 123,
        "created_at": "2023-10-27T10:00:00Z",
        "is_read": false
      }
    }
    ```

## Error Response dan Code

Sistem ini menggunakan format respons error yang konsisten untuk memudahkan penanganan di sisi klien.

### Format Umum Error Response

```json
{
  "status": "error",
  "code": <HTTP_STATUS_CODE>,
  "message": "<Pesan error umum>",
  "fields": [
    {
      "field": "<Nama field yang bermasalah>",
      "reason": "<Alasan validasi gagal>"
    }
  ],
  "example_request": <Contoh body request yang benar jika error validasi>,
  "timestamp": "2023-10-27T10:00:00Z"
}
```

### Contoh Error Responses

#### 1. Validasi Gagal (HTTP 400 Bad Request)

Terjadi ketika input tidak memenuhi aturan validasi (misalnya, format email salah, password terlalu lemah, field wajib kosong).

*   **Endpoint:** `POST /api/account/register`
*   **Request Body (contoh input tidak valid):**
    ```json
    {
      "full_name": "Jo",
      "email": "invalid-email",
      "password": "weak"
    }
    ```
*   **HTTP Status Code:** `400 Bad Request`
*   **Response Body:**
    ```json
    {
      "status": "error",
      "code": 400,
      "message": "Validasi gagal. Mohon periksa kembali input Anda.",
      "fields": [
        {
          "field": "FullName",
          "reason": "FullName minimal 3 karakter."
        },
        {
          "field": "Email",
          "reason": "Email harus berupa alamat email yang valid."
        },
        {
          "field": "Password",
          "reason": "Password minimal 8 karakter."
        }
      ],
      "example_request": {
        "full_name": "John Doe",
        "email": "john.doe@example.com",
        "password": "Password123!"
      },
      "timestamp": "2023-10-27T10:00:00Z"
    }
    ```

#### 2. Tidak Terotorisasi (HTTP 401 Unauthorized)

Terjadi ketika permintaan tidak memiliki token autentikasi yang valid atau kredensial salah.

*   **Endpoint:** `POST /api/account/login`
*   **Request Body:**
    ```json
    {
      "email": "john.doe@example.com",
      "password": "wrong_password"
    }
    ```
*   **HTTP Status Code:** `401 Unauthorized`
*   **Response Body:**
    ```json
    {
      "status": "error",
      "code": 401,
      "message": "Email atau kata sandi tidak valid. Mohon coba lagi.",
      "fields": [
        {
          "field": "General",
          "reason": "Email atau kata sandi tidak valid. Mohon coba lagi."
        }
      ],
      "timestamp": "2023-10-27T10:00:00Z"
    }
    ```
*   **Contoh lain (token tidak ada/invalid):**
    ```json
    {
      "status": "error",
      "code": 401,
      "message": "Tidak terotorisasi. Mohon login untuk mengakses sumber daya ini.",
      "fields": [
        {
          "field": "General",
          "reason": "Tidak terotorisasi. Mohon login untuk mengakses sumber daya ini."
        }
      ],
      "timestamp": "2023-10-27T10:00:00Z"
    }
    ```

#### 3. Terlarang (HTTP 403 Forbidden)

Terjadi ketika pengguna memiliki token valid tetapi tidak memiliki izin untuk melakukan tindakan tersebut, atau akunnya dalam status terlarang.

*   **Endpoint:** `POST /api/account/login` (jika akun diblokir)
*   **Request Body:** (sama seperti login)
*   **HTTP Status Code:** `403 Forbidden`
*   **Response Body:**
    ```json
    {
      "status": "error",
      "code": 403,
      "message": "Akun Anda diblokir secara permanen. Mohon hubungi dukungan.",
      "fields": [
        {
          "field": "General",
          "reason": "Akun Anda diblokir secara permanen. Mohon hubungi dukungan."
        }
      ],
      "timestamp": "2023-10-27T10:00:00Z"
    }
    ```
*   **Contoh lain (akses admin tanpa izin):**
    ```json
    {
      "status": "error",
      "code": 403,
      "message": "Terlarang. Anda tidak memiliki izin untuk melakukan tindakan ini.",
      "fields": [
        {
          "field": "General",
          "reason": "Terlarang. Anda tidak memiliki izin untuk melakukan tindakan ini."
        }
      ],
      "timestamp": "2023-10-27T10:00:00Z"
    }
    ```

#### 4. Tidak Ditemukan (HTTP 404 Not Found)

Terjadi ketika sumber daya yang diminta tidak ada.

*   **Endpoint:** `GET /api/shop/products/999` (produk dengan ID 999 tidak ada)
*   **HTTP Status Code:** `404 Not Found`
*   **Response Body:**
    ```json
    {
      "status": "error",
      "code": 404,
      "message": "Produk tidak ditemukan atau tidak tersedia.",
      "fields": [
        {
          "field": "General",
          "reason": "Produk tidak ditemukan atau tidak tersedia."
        }
      ],
      "timestamp": "2023-10-27T10:00:00Z"
    }
    ```

#### 5. Kesalahan Server Internal (HTTP 500 Internal Server Error)

Terjadi ketika ada masalah tak terduga di sisi server.

*   **HTTP Status Code:** `500 Internal Server Error`
*   **Response Body:**
    ```json
    {
      "status": "error",
      "code": 500,
      "message": "Oops! Terjadi kesalahan pada server kami. Silakan coba lagi nanti.",
      "fields": [
        {
          "field": "General",
          "reason": "Oops! Terjadi kesalahan pada server kami. Silakan coba lagi nanti."
        }
      ],
      "timestamp": "2023-10-27T10:00:00Z"
    }
    ```