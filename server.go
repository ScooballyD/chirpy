package main

import (
	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/ScooballyD/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	Platform       string
	Secret         string
	PolkaKey       string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	hits := fmt.Sprintf(
		"<html>\n	<body>\n	<h1>Welcome, Chirpy Admin</h1>\n	<p>Chirpy has been visited %d times!</p>\n</body>\n</html>",
		cfg.fileserverHits.Load())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(hits))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	if cfg.Platform != "dev" {
		w.WriteHeader(403)
		return
	}

	cfg.fileserverHits.Store(0)
	cfg.db.ResetUsers(r.Context())
}

func StartServer(dbQ *database.Queries) {
	cfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             dbQ,
		Platform:       os.Getenv("PLATFORM"),
		Secret:         os.Getenv("SECRET"),
		PolkaKey:       os.Getenv("POLKA_KEY"),
	}
	mux := http.NewServeMux()
	mux.Handle("/app/", http.StripPrefix("/app", cfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /admin/metrics", cfg.metricsHandler)
	mux.HandleFunc("POST /admin/reset", cfg.resetHandler)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", cfg.deleteChirp)
	mux.HandleFunc("GET /api/chirps", cfg.getChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", cfg.getChirps)
	mux.HandleFunc("POST /api/chirps", cfg.validateChirpHandler)
	mux.HandleFunc("POST /api/login", cfg.loginUser)
	mux.HandleFunc("POST /api/polka/webhooks", cfg.upgradeUser)
	mux.HandleFunc("POST /api/refresh", cfg.validateRefreshToken)
	mux.HandleFunc("POST /api/revoke", cfg.revokeRefreshToken)
	mux.HandleFunc("POST /api/users", cfg.createUser)
	mux.HandleFunc("PUT /api/users", cfg.updateUser)

	srv := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv.ListenAndServe()
}
