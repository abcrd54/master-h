# Monitor STB

Dashboard ringan untuk memeriksa status satu STB Android berdasarkan alamat IP yang diketik pengguna. Pemeriksaan memastikan port ADB dapat dijangkau, menjalankan `adb connect`, lalu membaca status ADB perangkat.

## Menjalankan di Armbian

```bash
cp .env.example .env
nano .env
docker compose up -d --build
```

Buka `http://IP-MASTER:8080`, masukkan IP STB, lalu pilih **Periksa**. Port ADB `5555` ditambahkan otomatis; port lain dapat ditulis, misalnya `192.168.50.10:5037`.

Jika `DASHBOARD_TOKEN` diisi, buka pertama kali dengan `http://IP-MASTER:8080/?token=TOKEN`. Browser menyimpan token secara lokal.

## Pemeriksaan

```bash
curl http://IP-MASTER:8080/health
docker compose ps
docker compose logs --tail=100 master-stb
```
