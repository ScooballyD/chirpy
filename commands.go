package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ScooballyD/chirpy/internal/auth"
	"github.com/ScooballyD/chirpy/internal/database"
	"github.com/google/uuid"
)

type User struct {
	Id        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	RefToken  string    `json:"refresh_token"`
	IsRed     bool      `json:"is_chirpy_red"`
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type Usr struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	usr := Usr{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&usr)
	if err != nil {
		fmt.Printf("unable to decoder request: %v", err)
		return
	}

	hPass, err := auth.HashPassword(usr.Password)
	if err != nil {
		fmt.Println(err)
	}

	user, err := cfg.db.CreateUser(
		r.Context(),
		database.CreateUserParams{
			Email:          usr.Email,
			HashedPassword: hPass,
		})
	if err != nil {
		fmt.Printf("unable to create user: %v", err)
	}
	resp := User{
		Id:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		IsRed:     user.IsChirpyRed,
	}
	respondWithJSON(w, resp, 201)
}

func (cfg *apiConfig) deleteChirp(w http.ResponseWriter, r *http.Request) {
	tkn, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, fmt.Sprintf("unauthorized: %v", err), 401)
		return
	}
	uid, err := auth.ValidateJWT(tkn, cfg.Secret)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, fmt.Sprintf("unauthorized: %v", err), 401)
		return
	}

	idStr := r.PathValue("chirpID")
	if idStr == "" {
		respondWithError(w, "to delete chirp matching id is required", 400)
		return
	}
	cid, err := uuid.Parse(idStr)
	if err != nil {
		fmt.Printf("unable to parse id: %v", err)
		return
	}

	chirp, err := cfg.db.GetChirp(r.Context(), cid)
	if err != nil {
		respondWithError(w, fmt.Sprintf("unable to find chirp: %v", err), 404)
		return
	}
	if uid != chirp.UserID {
		respondWithError(w, fmt.Sprintf("you are not the registered author of chirp %v", cid), 403)
		return
	}

	err = cfg.db.DeleteChirp(r.Context(), chirp.ID)
	if err != nil {
		fmt.Printf("unable to delete chirp: %v", err)
		return
	}
	respondWithJSON(w, nil, 204)
}

func (cfg *apiConfig) getChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.GetChirps(r.Context())
	if err != nil {
		fmt.Printf("unable to retrieve chirps: %v", err)
		return
	}
	if r.URL.Query().Get("sort") == "desc" {
		chirps, err = cfg.db.GetChirpsDesc(r.Context())
		if err != nil {
			fmt.Printf("unable to retrieve chirps: %v", err)
			return
		}
	}

	idStr := r.PathValue("chirpID")
	if idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			fmt.Printf("unable to parse id: %v", err)
			return
		}
		resp, err := cfg.db.GetChirp(r.Context(), id)
		if err != nil {
			er := fmt.Sprintf("unable to retrieve chirp: %v", err)
			respondWithError(w, er, 404)
			return
		}
		if resp.ID == uuid.Nil {
			respondWithError(w, "chirp does not exist", 404)
			return
		}
		chirp := Chirp{
			Id:        resp.ID,
			CreatedAt: resp.CreatedAt,
			UpdatedAt: resp.UpdatedAt,
			Body:      resp.Body,
			UserId:    resp.UserID,
		}
		respondWithJSON(w, chirp, 200)
		return
	}
	var resp []Chirp

	Aid := r.URL.Query().Get("author_id")
	if Aid != "" {
		id, err := uuid.Parse(Aid)
		if err != nil {
			fmt.Printf("unable to parse id: %v", err)
			return
		}

		for _, chirp := range chirps {
			chrp := Chirp{
				Id:        chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserId:    chirp.UserID,
			}
			if chrp.UserId == id {
				resp = append(resp, chrp)
			}
		}
		respondWithJSON(w, resp, 200)
		return
	}

	for _, chirp := range chirps {
		chrp := Chirp{
			Id:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserId:    chirp.UserID,
		}
		resp = append(resp, chrp)
	}
	respondWithJSON(w, resp, 200)
}

func (cfg *apiConfig) loginUser(w http.ResponseWriter, r *http.Request) {
	type Usr struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	usr := Usr{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&usr)
	if err != nil {
		fmt.Printf("unable to decoder request: %v", err)
		return
	}

	user, err := cfg.db.GetUser(r.Context(), usr.Email)
	if err != nil {
		fmt.Printf("error retrieving user info: %v", err)
		return
	}
	err = auth.CheckPasswordHash(usr.Password, user.HashedPassword)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, "Incorrect email or password", 401)
		return
	}

	tkn, err := auth.MakeJWT(user.ID, cfg.Secret)
	if err != nil {
		fmt.Printf("unable to make JWT: %v", err)
		return
	}

	Rtkn, err := auth.MakeRefreshToken()
	if err != nil {
		fmt.Printf("unable to make refresh token: %v", err)
		return
	}
	_, err = cfg.db.CreateRefToken(
		r.Context(),
		database.CreateRefTokenParams{
			Token:     Rtkn,
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(1440 * time.Hour),
		})
	if err != nil {
		fmt.Printf("unable to register refresh token: %v", err)
	}

	Rusr := User{
		Id:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
		Token:     tkn,
		RefToken:  Rtkn,
		IsRed:     user.IsChirpyRed,
	}

	respondWithJSON(w, Rusr, 200)
}

func (cfg *apiConfig) updateUser(w http.ResponseWriter, r *http.Request) {
	tkn, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, fmt.Sprintf("unauthorized: %v", err), 401)
		return
	}
	id, err := auth.ValidateJWT(tkn, cfg.Secret)
	if err != nil {
		fmt.Println(err)
		respondWithError(w, fmt.Sprintf("unauthorized: %v", err), 401)
		return
	}

	type Usr struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	usr := Usr{}

	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&usr)
	if err != nil {
		fmt.Printf("unable to decoder request: %v", err)
		return
	}

	hPass, err := auth.HashPassword(usr.Password)
	if err != nil {
		fmt.Println(err)
	}

	usrData, err := cfg.db.UpdateUserCredentials(
		r.Context(),
		database.UpdateUserCredentialsParams{
			ID:             id,
			Email:          usr.Email,
			HashedPassword: hPass,
		})
	if err != nil {
		respondWithError(w, fmt.Sprintf("unable to update credentials: %v", err), 401)
		return
	}

	pl := User{
		Id:        usrData.ID,
		UpdatedAt: usrData.UpdatedAt,
		CreatedAt: usrData.CreatedAt,
		Email:     usrData.Email,
		IsRed:     usrData.IsChirpyRed,
	}

	respondWithJSON(w, pl, 200)
}
