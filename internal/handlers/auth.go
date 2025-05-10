package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/auth"
	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/models"
	"github.com/gtrmalay/LMS.Sprint1.HTTP-Calculator/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

func RegisterHandler(s *storage.PostgresStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		user := models.User{
			Login:        req.Login,
			PasswordHash: string(hashedPassword),
		}

		if err := s.CreateUser(r.Context(), &user); err != nil {
			respondWithError(w, http.StatusConflict, "User already exists")
			return
		}

		respondWithJSON(w, http.StatusOK, map[string]string{"status": "OK"})
	}
}

func LoginHandler(s *storage.PostgresStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		user, err := s.GetUserByLogin(r.Context(), req.Login)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}

		token, err := auth.GenerateToken(user.ID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		respondWithJSON(w, http.StatusOK, models.LoginResponse{Token: token})
	}
}
