# Product Requirements: Monitor STB

## Tujuan

Menyediakan halaman web ringan untuk memeriksa konektivitas, status ADB, dan remote layar satu STB Android berdasarkan IP yang dimasukkan pengguna.

## Ruang lingkup

- Pengguna mengetik alamat IPv4 atau IPv6 STB.
- Port ADB `5555` digunakan otomatis bila port tidak dimasukkan.
- Aplikasi memeriksa koneksi TCP, melakukan `adb connect`, lalu membaca `adb get-state`.
- Hasil menampilkan alamat yang diperiksa, waktu pemeriksaan, dan status: `ready`, `unreachable`, `offline`, atau `unauthorized`.
- Saat status `ready`, pengguna dapat membuka remote layar scrcpy dengan H.264 720p tanpa audio.
- Alamat terakhir disimpan hanya di browser pengguna.

## Di luar ruang lingkup

- Daftar perangkat yang disimpan di server.
- Kontrol remote, reboot, aplikasi, atau key event.
- Unggah dan instal APK.
- Eksekusi script pada STB.
- Proxy GOST.

## API

| Method | Endpoint | Keterangan |
| --- | --- | --- |
| `POST` | `/api/check` | Memeriksa IP STB dari JSON `{ "address": "192.168.50.10" }`. |
| `GET` | `/health` | Pemeriksaan kesehatan layanan. |

Relay remote scrcpy berjalan sebagai service terpisah pada port `8000`.

## Batasan

- Input hanya menerima alamat IP, bukan hostname.
- Pemeriksaan memiliki batas waktu 12 detik.
- ADB dijalankan dari container dashboard dengan key bersama pada volume Docker.
