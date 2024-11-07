package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
)

type Chirp struct {
	Body         string `json:"body"`
	Cleaned_body string `json:"cleaned_body"`
}

type respVal struct {
	Error string `json:"error"`
	Valid bool   `json:"valid"`
}

func respondWithError(w http.ResponseWriter, er string, code int) {
	resp := respVal{
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

func respondWithJSON(w http.ResponseWriter, payload interface{}, code int) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("marshal error: %v", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(dat)
}

func (cfg *apiConfig) validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	chirp := Chirp{}
	err := decoder.Decode(&chirp)

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
		fmt.Printf(" -%v\n", word)
		match, _ := regexp.MatchString("\\b"+regexp.QuoteMeta(strings.ToLower(word))+"\\b", filter)
		if match {
			cleaned = append(cleaned, "****")
			continue
		}
		cleaned = append(cleaned, word)
	}
	chirp.Cleaned_body = strings.Join(cleaned, " ")

	respondWithJSON(w, chirp, 200)
}
