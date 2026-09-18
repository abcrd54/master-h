package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type CheckRequest struct {
	Address string `json:"address"`
}

type CheckResult struct {
	Address     string `json:"address"`
	Status      string `json:"status"`
	LastChecked string `json:"lastChecked"`
	Error       string `json:"error,omitempty"`
}

type InputRequest struct {
	Address  string `json:"address"`
	Action   string `json:"action"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	X2       int    `json:"x2"`
	Y2       int    `json:"y2"`
	Duration int    `json:"duration"`
}

type state struct {
	adb string
}

func main() {
	s := &state{adb: env("ADB_PATH", "adb")}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/check", s.handleCheck)
	mux.HandleFunc("GET /api/iphone-stream", s.handleIPhoneStream)
	mux.HandleFunc("POST /api/input", s.handleInput)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /", serveIndex)

	addr := env("LISTEN_ADDR", ":8080")
	server := &http.Server{Addr: addr, Handler: requestLog(mux), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("master-stb monitor listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}

func (s *state) handleIPhoneStream(w http.ResponseWriter, r *http.Request) {
	address, err := normalizeAddress(r.URL.Query().Get("address"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream tidak didukung", http.StatusInternalServerError)
		return
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
		image, cmdErr := run(ctx, s.adb, "-s", address, "exec-out", "screencap", "-p")
		cancel()
		if cmdErr == nil && len(image) > 0 {
			_, _ = fmt.Fprintf(w, "--frame\r\nContent-Type: image/png\r\nContent-Length: %d\r\n\r\n", len(image))
			_, _ = w.Write([]byte(image))
			_, _ = w.Write([]byte("\r\n"))
			flusher.Flush()
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *state) handleInput(w http.ResponseWriter, r *http.Request) {
	var req InputRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON tidak valid")
		return
	}
	address, err := normalizeAddress(req.Address)
	if err != nil || req.X < 0 || req.Y < 0 || req.X > 10000 || req.Y > 10000 || req.X2 < 0 || req.Y2 < 0 || req.X2 > 10000 || req.Y2 > 10000 {
		writeError(w, http.StatusBadRequest, "input atau alamat tidak valid")
		return
	}
	args := []string{"shell", "input"}
	switch req.Action {
	case "tap":
		args = append(args, "tap", strconv.Itoa(req.X), strconv.Itoa(req.Y))
	case "swipe":
		duration := req.Duration
		if duration < 50 || duration > 5000 {
			duration = 400
		}
		args = append(args, "swipe", strconv.Itoa(req.X), strconv.Itoa(req.Y), strconv.Itoa(req.X2), strconv.Itoa(req.Y2), strconv.Itoa(duration))
	default:
		writeError(w, http.StatusBadRequest, "aksi input tidak didukung")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if out, cmdErr := run(ctx, s.adb, append([]string{"-s", address}, args...)...); cmdErr != nil {
		writeError(w, http.StatusBadGateway, shortOutput(out, cmdErr))
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *state) handleCheck(w http.ResponseWriter, r *http.Request) {
	var req CheckRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "JSON tidak valid")
		return
	}
	address, err := normalizeAddress(req.Address)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	status, message := s.check(ctx, address)
	writeJSON(w, http.StatusOK, CheckResult{Address: address, Status: status, LastChecked: time.Now().Format(time.RFC3339), Error: message})
}

// normalizeAddress permits an IP address with an optional ADB TCP port.
func normalizeAddress(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", errors.New("IP STB wajib diisi")
	}
	if ip := net.ParseIP(input); ip != nil {
		return net.JoinHostPort(ip.String(), "5555"), nil
	}
	host, port, err := net.SplitHostPort(input)
	if err != nil || net.ParseIP(host) == nil {
		return "", errors.New("masukkan alamat IP, dengan port opsional (contoh: 192.168.50.10 atau 192.168.50.10:5555)")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return "", errors.New("port tidak valid")
	}
	return net.JoinHostPort(host, port), nil
}

func (s *state) check(ctx context.Context, address string) (string, string) {
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return "unreachable", shortError(err)
	}
	_ = conn.Close()
	_, _ = run(ctx, s.adb, "connect", address)
	out, err := run(ctx, s.adb, "-s", address, "get-state")
	text := strings.ToLower(out)
	if strings.Contains(text, "unauthorized") {
		return "unauthorized", "otorisasi RSA belum disetujui pada STB"
	}
	if strings.Contains(text, "offline") {
		return "offline", "ADB offline"
	}
	if err != nil || strings.TrimSpace(out) != "device" {
		return "offline", shortError(err)
	}
	return "ready", ""
}

func run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	b, err := cmd.CombinedOutput()
	return string(b), err
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	b, err := os.ReadFile(env("WEB_FILE", "web/index.html"))
	if err != nil {
		http.Error(w, "dashboard unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(b)
}

func requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func shortError(err error) string {
	if err == nil {
		return "ADB belum siap"
	}
	return shortOutput("", err)
}

func shortOutput(out string, err error) string {
	s := strings.TrimSpace(out)
	if s == "" && err != nil {
		s = err.Error()
	}
	if len(s) > 240 {
		s = s[:240]
	}
	return s
}
