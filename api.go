package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ScooballyD/chirpy/internal/auth"
	"github.com/ScooballyD/chirpy/internal/database"
	"github.com/google/uuid"
)

type Chirp struct {
	Id        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserId    uuid.UUID `json:"user_id"`
}

type RespVal struct {
	Error string `json:"error"`
	Valid bool   `json:"valid"`
}

func respondWithError(w http.ResponseWriter, er string, code int) {
	resp := RespVal{
		Error: er,
		Valid: false,
	}

	dat, err := json.Marshal(resp)
	if err != nil {
		log.Printf("marshal error: %v", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, pl interface{}, code int) {
	dat, err := json.Marshal(pl)
	if err != nil {
		log.Printf("marshal error: %v", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func (cfg *apiConfig) revokeRefreshToken(w http.ResponseWriter, r *http.Request) {
	Rtkn, err := auth.GetRefreshToken(r.Header)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = cfg.db.RevokeToken(r.Context(), Rtkn)
	if err != nil {
		respondWithError(w, fmt.Sprintf("error revoking token: %v", err), 401)
		return
	}
	respondWithJSON(w, nil, 204)
}

func (cfg *apiConfig) saveChirp(chirp Chirp, r *http.Request, w http.ResponseWriter) {
	chrp, err := cfg.db.CreateChirp(
		r.Context(),
		database.CreateChirpParams{
			Body:   chirp.Body,
			UserID: chirp.UserId,
		})
	if err != nil {
		fmt.Printf("unable to save chirp: %v", err)
		w.WriteHeader(500)
		return
	}

	resp := Chirp{
		Id:        chrp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserId:    chirp.UserId,
	}

	respondWithJSON(w, resp, 201)
}

func (cfg *apiConfig) validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	tkn, err := auth.GetBearerToken(r.Header)
	if err != nil {
		fmt.Println(err)
		return
	}
	id, err := auth.ValidateJWT(tkn, cfg.Secret)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, "Unauthorized", 401)
		return
	}

	decoder := json.NewDecoder(r.Body)
	chirp := Chirp{}
	err = decoder.Decode(&chirp)

	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		respondWithError(w, fmt.Sprintf("%v", err), 400)
		return
	}

	if len(chirp.Body) > 140 {
		respondWithError(w, "Chirp too long", 400)
		return
	}

	filter := "kerfuffle sharbert fornax"
	words := strings.Split(chirp.Body, " ")
	var cleaned []string

	for _, word := range words {
		match, _ := regexp.MatchString("\\b"+regexp.QuoteMeta(strings.ToLower(word))+"\\b", filter)
		if match {
			cleaned = append(cleaned, "****")
			continue
		}
		cleaned = append(cleaned, word)
	}
	chirp.Body = strings.Join(cleaned, " ")
	chirp.UserId = id

	cfg.saveChirp(chirp, r, w)
}

func (cfg *apiConfig) validateRefreshToken(w http.ResponseWriter, r *http.Request) {
	Rtkn, err := auth.GetRefreshToken(r.Header)
	if err != nil {
		fmt.Println(err)
		return
	}

	Rdata, err := cfg.db.GetRefreshToken(r.Context(), Rtkn)
	if err != nil {
		respondWithError(w, fmt.Sprintf("unable to retrieve refresh token from database: %v", err), 401)
		return
	}
	if Rdata.ExpiresAt.Before(time.Now()) {
		respondWithError(w, "refresh token has expired", 401)
		return
	}
	if Rdata.RevokedAt.Valid {
		respondWithError(w, "refresh token has been revoked", 401)
		return
	}

	type tkn struct {
		Token string `json:"token"`
	}

	jwt, err := auth.MakeJWT(Rdata.UserID, cfg.Secret)
	if err != nil {
		fmt.Printf("unable to make JWT: %v", err)
		return
	}

	Tkn := tkn{
		Token: jwt,
	}

	respondWithJSON(w, Tkn, 200)
}
