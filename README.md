# startup-service

Service kecil berbasis Go untuk **menghidupkan** dan **mematikan** resource GCP (Compute Engine VM dan Cloud SQL) secara terjadwal, berdasarkan konfigurasi `desired state` di file JSON.

Tujuan utamanya adalah **efisiensi biaya**: instance development, runner, atau workload internal yang tidak dipakai 24 jam (misalnya hanya digunakan saat jam kerja) dapat dimatikan otomatis di luar jam kerja, lalu dihidupkan kembali saat dibutuhkan.

---

## Fitur

- Mengelola banyak project GCP sekaligus dalam satu run.
- Mengatur status **GCE Instance**: `running` ↔ `stopped` (via Compute API `Start`/`Stop`).
- Mengatur status **Cloud SQL**: `active` ↔ `inactive` (via `ActivationPolicy` = `ALWAYS` / `NEVER`, sehingga instance tidak perlu di-delete).
- Idempotent: kalau instance sudah berada di state yang diinginkan, service tidak melakukan aksi tambahan.
- Mendukung dua mode autentikasi: **Workload Identity / ADC** (recommended) dan **Service Account Key file** (legacy).
- **Notifikasi Google Chat**: ringkasan status semua project & instance dikirim ke Google Chat via webhook setelah run selesai.

---

## Use Case Utama

| Skenario | Cara Pakai |
|---|---|
| Mematikan resource setiap hari pukul 19:00 WIB | Jalankan service dengan config yang semua state-nya `stopped`/`inactive`, lewat Cloud Scheduler. |
| Menghidupkan kembali pukul 07:00 WIB | Jalankan service dengan config yang state-nya `running`/`active`. |
| Mematikan resource yang idle/tidak dipakai 24 jam | Set state `stopped`/`inactive` lewat schedule harian, atau trigger manual. |
| Stop sementara untuk maintenance | Edit config, run sekali. |

Pendekatan paling umum: punya **dua file config** (`config.work-hours.json` dan `config.off-hours.json`), lalu Cloud Scheduler menjalankan service dengan config yang sesuai pada jamnya.

---

## Struktur Project

```
.
├── main.go                       # Entry point: baca config, auth, loop tiap project
├── go.mod / go.sum               # Go module & dependencies
├── config/
│   ├── config.json               # Konfigurasi project & desired state
│   └── service-account.json      # (Opsional) SA key file, hanya jika tidak pakai Workload Identity
└── helpers/
    ├── config.go                 # Struct & loader untuk config.json
    ├── auth.go                   # Logic autentikasi (Workload Identity / SA key)
    ├── gce.go                    # ManageGCE: Start/Stop VM via Compute Engine API
    └── cloudsql.go               # ManageCloudSQL: patch ActivationPolicy via Cloud SQL Admin API
```

---

## Prasyarat

- **Go** versi `1.25.6` atau lebih baru (lihat `go.mod`).
- **Akses ke project GCP** target dengan IAM yang sesuai (lihat section IAM).
- **`gcloud` CLI** terinstal — hanya dibutuhkan jika menggunakan mode Service Account Key file (`auth.go` memanggil `gcloud auth activate-service-account`).
- **API yang harus di-enable** di tiap project:
  - `compute.googleapis.com` (Compute Engine API)
  - `sqladmin.googleapis.com` (Cloud SQL Admin API)

---

## Konfigurasi (`config/config.json`)

```json
{
  "auth": {
    "workload_identity": true,
    "service_account_key_file": "config/service-account.json"
  },
  "google_chat_webhooks": [
    "https://chat.googleapis.com/v1/spaces/XXXX/messages?key=...&token=..."
  ],
  "projects": {
    "service-a": {
      "project_id": "dev",
      "zone": "asia-southeast2-a",
      "gce_instances": {
        "gce-01": "running"
      },
      "cloud_sql_instances": {
        "postgresql": "active"
      }
    },
    "service-b": {
      "project_id": "shared",
      "zone": "asia-southeast2-a",
      "gce_instances": {
        "gce-02": "stopped"
      },
      "cloud_sql_instances": {
        "mysql": "inactive"
      }
    }
  }
}
```

### Penjelasan field

| Field | Tipe | Keterangan |
|---|---|---|
| `auth.workload_identity` | bool | Jika `true`, pakai Application Default Credentials (Workload Identity / Cloud Run / GCE service account). Tidak butuh key file. |
| `auth.service_account_key_file` | string | Path ke SA key JSON. Hanya dipakai jika `workload_identity = false`. |
| `google_chat_webhooks` | array of string | (Opsional) Daftar URL webhook Google Chat. Jika kosong / tidak ada, notifikasi di-skip. Bisa diisi lebih dari satu untuk fan-out ke beberapa space. |
| `projects` | object | Map. Key bebas (label internal), value adalah konfigurasi per-project. |
| `projects.<key>.project_id` | string | GCP Project ID asli. |
| `projects.<key>.zone` | string | Zone GCE (mis. `asia-southeast2-a`). Cloud SQL tidak butuh zone. |
| `projects.<key>.gce_instances` | map | `nama_instance` → `"running"` / `"stopped"` |
| `projects.<key>.cloud_sql_instances` | map | `nama_instance` → `"active"` / `"inactive"` |

### Nilai `desired state` yang valid

- **GCE**: `running` (start) atau `stopped` (stop). Selain itu akan di-skip dengan log warning.
- **Cloud SQL**: `active` (ActivationPolicy `ALWAYS`) atau `inactive` (ActivationPolicy `NEVER`). Selain itu akan di-skip.

> Untuk Cloud SQL, perubahan `ActivationPolicy` jauh lebih hemat biaya dibanding delete + restore, karena storage tetap dipertahankan tetapi compute tidak dihitung saat `NEVER`.

---

## Autentikasi

### Mode 1: Workload Identity / ADC (recommended)

Set di config:
```json
"auth": { "workload_identity": true }
```

Cocok untuk dijalankan di:
- **Cloud Run / Cloud Run Jobs** (assign service account ke service).
- **GKE** dengan Workload Identity binding.
- **GCE VM** dengan service account terlampir.
- **Lokal**: jalankan `gcloud auth application-default login` dulu.

### Mode 2: Service Account Key file (legacy)

Set di config:
```json
"auth": {
  "workload_identity": false,
  "service_account_key_file": "config/service-account.json"
}
```

Tempatkan file JSON SA key di path yang ditunjuk. Mode ini akan menjalankan `gcloud auth activate-service-account` di belakang layar, jadi `gcloud` CLI **wajib** ada di PATH.

> ⚠️ **Hindari commit `service-account.json` ke Git**. Tambahkan ke `.gitignore`.

---

## IAM Permission yang Dibutuhkan

Service account yang dipakai harus punya minimal role berikut di tiap project target:

- `roles/compute.instanceAdmin.v1` — untuk start/stop GCE instance.
- `roles/cloudsql.editor` — untuk patch ActivationPolicy Cloud SQL.

Atau, kalau mau lebih ketat, buat **custom role** dengan permission minimal:

```
compute.instances.get
compute.instances.start
compute.instances.stop
compute.zoneOperations.get
cloudsql.instances.get
cloudsql.instances.update
```

---

## Build & Run

### Build binary

```bash
go mod download
go build -o startup-service .
```

### Run lokal

```bash
./startup-service
```

Service akan membaca `config/config.json` (path relatif terhadap working directory), melakukan autentikasi, lalu memproses semua project secara berurutan.

### Run langsung tanpa build

```bash
go run .
```

### Output

Semua aksi dicatat ke stdout via `log` standar Go, contoh:
```
=== Processing project: service-a (project_id=dev) ===
Managing Cloud SQL instances...
[cloudsql] Checking instance=postgresql desired=active
[cloudsql] instance=postgresql currentActivationPolicy=NEVER
[cloudsql] Updating activationPolicy NEVER -> ALWAYS
Managing GCE instances...
[gce] Checking instance=gce-01 desired=running
[gce] instance=gce-01 current=TERMINATED
[gce] Starting instance=gce-01
```

---

## Pola Deployment untuk Cost Efficiency

### Pola A — Cloud Run Jobs + Cloud Scheduler (recommended)

1. Build container image:
   ```bash
   gcloud builds submit --tag gcr.io/PROJECT_ID/startup-service
   ```
2. Buat dua **Cloud Run Jobs**:
   - `startup-service-stop` — image yang sama, env/config diarahkan ke config "off-hours".
   - `startup-service-start` — config "work-hours".
3. Buat dua **Cloud Scheduler** (timezone `Asia/Jakarta`):
   - Stop: `0 19 * * 1-5` → trigger job stop tiap hari kerja jam 19:00 WIB.
   - Start: `0 7 * * 1-5` → trigger job start tiap hari kerja jam 07:00 WIB.
4. Pastikan service account Cloud Run Job punya IAM yang dibutuhkan dan `auth.workload_identity = true` di config.

### Pola B — Cron tradisional di VM

```cron
# /etc/crontab — timezone server harus WIB
0 19 * * 1-5 ops cd /opt/startup-service && ./startup-service -config config/off-hours.json
0 7  * * 1-5 ops cd /opt/startup-service && ./startup-service -config config/work-hours.json
```

> Saat ini path config masih di-hardcode ke `config/config.json` di `main.go`. Untuk dukungan flag `-config`, perlu sedikit modifikasi (lihat section di bawah).

### Pola C — Mematikan resource yang idle 24 jam

Jalankan service dengan state `stopped`/`inactive` sekali sehari (mis. tengah malam) terhadap project sandbox/dev. Tim yang butuh menghidupkan tinggal trigger run dengan config "active" atau hidupkan manual via Console.

---

## Notifikasi Google Chat

Setelah semua project selesai diproses, service mengumpulkan **status summary** per project per instance, lalu mengirimkannya ke setiap webhook yang terdaftar di `google_chat_webhooks`.

### Cara membuat webhook Google Chat

1. Buka space Google Chat tujuan.
2. Klik nama space → **Apps & integrations** → **Webhooks** → **Add webhook**.
3. Beri nama (mis. `startup-service`), copy URL webhook yang muncul.
4. Tempel URL ke array `google_chat_webhooks` di `config.json`.

### Format pesan yang dikirim

```
Process completed. Status summary:
Project: service-a
  - gce-01: Running
  - postgresql: Active

Project: service-b
  - gce-02: Stopped
  - mysql: Inactive
```

### Status value yang mungkin muncul

| Status | Artinya |
|---|---|
| `Running` / `Stopped` | GCE instance berhasil di-set ke desired state. |
| `Active` / `Inactive` | Cloud SQL berhasil di-patch ActivationPolicy-nya. |
| `ERROR` | Gagal `Get` ke instance (mis. tidak ditemukan, permission ditolak). |
| `FAILED` | API call dieksekusi tapi return error (mis. patch gagal). |
| `INVALID` | Value desired state di config tidak valid. |

> Webhook dipanggil **sekali per run**, bukan per instance. Jika `google_chat_webhooks` kosong atau tidak diisi, langkah notifikasi otomatis di-skip — service tetap berjalan normal.

> ⚠️ URL webhook bersifat sensitif (siapa pun dengan URL bisa kirim pesan ke space). Hindari commit ke Git; lebih baik inject lewat secret manager atau env var di Cloud Run.

---

## Catatan Pengembangan

Beberapa hal yang sebaiknya diketahui sebelum production:

1. **Status summary per instance.**
   `ManageCloudSQL` (dan kemungkinan `ManageGCE`, jika sudah disesuaikan) menerima parameter `status map[string]string` yang dipakai untuk mencatat hasil tiap instance (`Running`, `Stopped`, `ERROR`, `FAILED`, `INVALID`). Map ini lalu digabungkan via `appendMaps` ke `statusSummary` global, dan diformat oleh `formatStatusSummary` untuk dikirim ke Google Chat. Pastikan signature pemanggilan di `main.go` konsisten dengan deklarasi fungsi di `helpers/`.

2. **Path config masih hardcoded** di `main.go` (`config/config.json`). Untuk fleksibilitas multi-jadwal (work-hours / off-hours), tambahkan flag CLI:
   ```go
   configPath := flag.String("config", "config/config.json", "path ke config")
   flag.Parse()
   cfg, err := helpers.ReadConfig(*configPath)
   ```

3. **Operation tidak di-`Wait`.** Pada `gce.go` hasil `Start`/`Stop` operation di-discard (`_ = op`). Ini berarti service exit sebelum operation benar-benar selesai. Untuk job sekali jalan biasanya OK, tapi kalau perlu kepastian (misal status di notifikasi Google Chat ingin merefleksikan state final), panggil `op.Wait(ctx)` sebelum menulis ke `status` map.

4. **Tidak ada retry/backoff.** Kalau ingin lebih robust di production, tambahkan retry untuk error transient (rate limit, 5xx). Saat ini error transient akan langsung tercatat sebagai `ERROR`/`FAILED` di notifikasi.

---

## Lisensi

Internal use. Sesuaikan dengan kebijakan organisasi.