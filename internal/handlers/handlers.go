package handlers

import (
	"context"
	"encoding/json"
	"log"
	"marketplace/internal/sqlite"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var secretKey = []byte("my_secret_key")

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

type CreateAdRequest struct {
	Title    string `json:"title"`
	Text     string `json:"text"`
	ImageURL string `json:"image_url"`
	Price    int64  `json:"price"`
}

func (authReq *AuthRequest) hashPassword() (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(authReq.Password), bcrypt.DefaultCost)
	return string(bytes), err
}

func RegisterHandler(storage *sqlite.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Println(err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.Password == "" {
			http.Error(w, "Username or/and password are not specified", http.StatusNotFound)
			return
		}

		hashedPassword, err := req.hashPassword()
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		newUser := sqlite.User{
			ID:        uuid.New().String(),
			Username:  req.Username,
			Password:  hashedPassword,
			CreatedAt: time.Now(),
		}

		if err := storage.CreateUser(newUser); err != nil {
			log.Fatalf("Failed to create user in database: %s", err)
			http.Error(w, "Internal server error", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newUser)
		w.WriteHeader(http.StatusCreated)

	}
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateToken(user sqlite.User) (string, error) {
	claims := jwt.MapClaims{
		"userID": user.ID,
		"exp":    time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

func LoginHandler(storage *sqlite.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Println(err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		user, err := storage.GetUserByUsername(req.Username)
		if err != nil {
			log.Printf("Failed to find user: %s", err)
			http.Error(w, "Invalid login or password", http.StatusUnauthorized)
			return
		}

		if !checkPasswordHash(req.Password, user.Password) {
			log.Printf("Failed to check password: %s", err)
			http.Error(w, "Invalid login or password", http.StatusUnauthorized)
			return
		}

		token, err := generateToken(user)
		if err != nil {
			log.Fatal("Failed to generate token")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthResponse{Token: token})
		w.WriteHeader(http.StatusOK)
	}
}

func parseToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			if r.Method == http.MethodGet {
				next(w, r)
			} else {
				http.Error(w, "Token is empty", http.StatusUnauthorized)
				return
			}
		} else {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			token, err := parseToken(tokenString)

			if err != nil || !token.Valid {
				log.Println(err)
				http.Error(w, "Token is not valid", http.StatusUnauthorized)
				return
			}

			claims, _ := token.Claims.(jwt.MapClaims)
			userID := claims["userID"]

			ctx := context.WithValue(r.Context(), "userID", userID)
			next(w, r.WithContext(ctx))
		}
	}
}

func AddCardHandler(storage *sqlite.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateAdRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		userID := r.Context().Value("userID").(string)
		newAd := sqlite.Ad{
			ID:        uuid.New().String(),
			UserID:    userID,
			Title:     req.Title,
			Text:      req.Text,
			ImageURL:  req.ImageURL,
			Price:     req.Price,
			CreatedAt: time.Now(),
		}

		err := storage.CreateAd(newAd)
		if err != nil {
			log.Printf("Error creating ad: %s", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newAd)
		w.WriteHeader(http.StatusCreated)
		log.Printf("Ad %s created by user %s", newAd.ID, userID)
	}
}

func GetAdsParams(urlQuery url.Values) (*sqlite.AdsParamsRequest, error) {
	params := &sqlite.AdsParamsRequest{}

	var err error
	params.Page, err = strconv.Atoi(urlQuery.Get("page"))
	if err != nil && urlQuery.Get("page") != "" {
		return nil, err
	}
	if params.Page == 0 {
		params.Page = 1
	}

	params.SortBy = urlQuery.Get("sort_by")
	if params.SortBy == "created_at" || params.SortBy == "" {
		params.SortBy = "ads.created_at"
	}

	params.SortDir = urlQuery.Get("sort_dir")
	if params.SortDir == "" {
		params.SortDir = "asc"
	}

	priceMinStr := urlQuery.Get("price_min")
	if priceMinStr != "" {
		params.PriceMin, err = strconv.Atoi(priceMinStr)
		if err != nil {
			return nil, err
		}
	}

	priceMaxStr := urlQuery.Get("price_max")
	if priceMaxStr != "" {
		params.PriceMax, err = strconv.Atoi(priceMaxStr)
		if err != nil {
			return nil, err
		}
	}

	return params, nil
}

func GetAllCardsHandler(storage *sqlite.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var userID string
		if r.Context().Value("userID") != nil {
			userID = r.Context().Value("userID").(string)
		}

		params, err := GetAdsParams(r.URL.Query())
		if err != nil {
			log.Printf("Error to get ads params: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ads, err := storage.GetListAds(userID, *params)
		if err != nil {
			log.Printf("Error listing ads: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ads)
		w.WriteHeader(http.StatusOK)
	}
}
