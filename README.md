# Monitor STB

Dashboard untuk memeriksa status dan membuka remote layar satu STB Android berdasarkan alamat IP yang diketik pengguna. Pemeriksaan memastikan port ADB dapat dijangkau, menjalankan `adb connect`, lalu membaca status ADB perangkat.

## Menjalankan di Armbian

```bash
docker compose up -d --build
```

Buka `http://IP-MASTER:8080`, masukkan IP STB, lalu pilih **Periksa**. Port ADB `5555` ditambahkan otomatis; port lain dapat ditulis, misalnya `192.168.50.10:5037`. Tombol **Buka remote layar** hanya muncul saat status `ready`.

Relay remote berjalan di port `8000`. Batasi port ini melalui firewall/ACL Tailscale. Remote pada browser Android memerlukan HTTPS agar WebCodecs tersedia. Jika dashboard diakses melalui HTTPS, proxy-kan relay ke HTTPS port `8443`; dashboard otomatis menggunakan port tersebut.

## Pemeriksaan

```bash
curl http://IP-MASTER:8080/health
docker compose ps
docker compose logs --tail=100 master-stb
```
