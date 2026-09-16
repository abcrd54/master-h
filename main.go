package main

import (
	"context"
	"encoding/json"
	"errors"
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

type state struct {
	adb string
}

func main() {
	s := &state{adb: env("ADB_PATH", "adb")}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/check", s.handleCheck)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /", serveIndex)

	addr := env("LISTEN_ADDR", ":8080")
	server := &http.Server{Addr: addr, Handler: requestLog(auth(mux)), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("master-stb monitor listening on %s", addr)
	log.Fatal(server.ListenAndServe())
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

func auth(next http.Handler) http.Handler {
	token := os.Getenv("DASHBOARD_TOKEN")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		publicPage := r.Method == http.MethodGet && r.URL.Path == "/"
		if token != "" && r.URL.Path != "/health" && !publicPage && r.Header.Get("X-Dashboard-Token") != token {
			writeError(w, http.StatusUnauthorized, "token diperlukan")
			return
		}
		next.ServeHTTP(w, r)
	})
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
