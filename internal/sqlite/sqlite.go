package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type Ad struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Text      string    `json:"text"`
	ImageURL  string    `json:"image_url"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"created_at"`
}

type AdResponse struct {
	Username string  `json:"username"`
	Title    string  `json:"title"`
	Text     string  `json:"text"`
	ImageURL string  `json:"image_url"`
	Price    float64 `json:"price"`
	IsMine   bool    `json:"is_mine"`
}

type AdsParamsRequest struct {
	Page     int
	Limit    int
	SortBy   string
	SortDir  string
	PriceMin float64
	PriceMax float64
}

func (s *Storage) CreateUser(user User) error {
	_, err := s.db.Exec("INSERT INTO users (id, username, password, created_at) VALUES ($1, $2, $3, $4)",
		user.ID, user.Username, user.Password, user.CreatedAt)
	return err
}

func (s *Storage) GetUserByUsername(username string) (User, error) {
	var user User
	err := s.db.QueryRow("SELECT id, username, password FROM users WHERE username = $1", username).
		Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Storage) CreateAd(ad Ad) error {
	_, err := s.db.Exec("INSERT INTO ads (id, user_id, title, text, image_url, price, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		ad.ID, ad.UserID, ad.Title, ad.Text, ad.ImageURL, ad.Price, ad.CreatedAt)
	return err
}

func (s *Storage) GetListAds(userID_Current string, params AdsParamsRequest) ([]AdResponse, error) {
	query := `
		SELECT 
			ads.title, ads.text, ads.image_url, ads.price, users.username, users.id
		FROM ads
		JOIN users ON ads.user_id = users.id
		WHERE 1=1
	`
	if params.PriceMin > 0 {
		query += fmt.Sprintf(" AND ads.price >= $%f", params.PriceMin)
	}

	if params.PriceMax > 0 {
		query += fmt.Sprintf(" AND ads.price <= $%f", params.PriceMax)
	}

	if params.SortBy != "" && params.SortDir != "" {
		query += fmt.Sprintf(" ORDER BY %s %s", params.SortBy, params.SortDir)
	} else {
		query += " ORDER BY ads.created_at DESC"
	}

	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", params.Limit)
	}
	if params.Page > 0 {
		query += fmt.Sprintf(" OFFSET $%d", (params.Page+1)*10)
	}

	rows, err := s.db.Query(query)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ads []AdResponse
	for rows.Next() {
		var ad AdResponse
		var userID_Database, title, text, imageURL, username string
		var price float64

		if err := rows.Scan(&title, &text, &imageURL, &price, &username, &userID_Database); err != nil {
			return nil, err
		}

		ad.Title = title
		ad.Text = text
		ad.ImageURL = imageURL
		ad.Price = price
		ad.Username = username
		ad.IsMine = userID_Current != "" && userID_Current == userID_Database

		ads = append(ads, ad)
	}

	return ads, nil
}

func (s *Storage) createUsersTable() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL
		)
	`)
	return err
}

func (s *Storage) createAdsTable() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS ads (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			title VARCHAR(255) NOT NULL,
			text TEXT,
			image_url VARCHAR(255),
			price DECIMAL(10, 2) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	return err
}

func New(storagePath string) (*Storage, error) {
	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %s", err)
	}
	database := &Storage{db: db}

	if err = database.createUsersTable(); err != nil {
		log.Fatalf("Failed to create users table: %s", err)
	}

	if err = database.createAdsTable(); err != nil {
		log.Fatalf("Failed to create ads table: %s", err)
	}

	return database, nil
}
