# Product Requirements: Monitor STB

## Tujuan

Menyediakan halaman web ringan untuk memeriksa konektivitas dan status ADB satu STB Android berdasarkan IP yang dimasukkan pengguna.

## Ruang lingkup

- Pengguna mengetik alamat IPv4 atau IPv6 STB.
- Port ADB `5555` digunakan otomatis bila port tidak dimasukkan.
- Aplikasi memeriksa koneksi TCP, melakukan `adb connect`, lalu membaca `adb get-state`.
- Hasil menampilkan alamat yang diperiksa, waktu pemeriksaan, dan status: `ready`, `unreachable`, `offline`, atau `unauthorized`.
- Alamat terakhir disimpan hanya di browser pengguna.
- Token dashboard opsional melindungi API.

## Di luar ruang lingkup

- Daftar perangkat yang disimpan di server.
- Kontrol remote, reboot, aplikasi, atau key event.
- Remote layar scrcpy.
- Unggah dan instal APK.
- Eksekusi script pada STB.
- Proxy GOST.

## API

| Method | Endpoint | Keterangan |
| --- | --- | --- |
| `POST` | `/api/check` | Memeriksa IP STB dari JSON `{ "address": "192.168.50.10" }`. |
| `GET` | `/health` | Pemeriksaan kesehatan layanan. |

## Batasan

- Input hanya menerima alamat IP, bukan hostname.
- Pemeriksaan memiliki batas waktu 12 detik.
- ADB dijalankan dari container dashboard dengan key bersama pada volume Docker.
