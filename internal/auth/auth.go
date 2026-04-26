package auth

import (
	"database/sql"
	"errors"
	"strings"

	"git.romanzipp.net/romanzipp/news/internal/models"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserExists     = errors.New("user already exists")
	ErrInvalidLogin   = errors.New("invalid email or password")
	ErrEmailRequired  = errors.New("email is required")
	ErrUsernameRequired = errors.New("username is required")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Register(email, username, password string) (*models.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	username = strings.TrimSpace(username)

	if email == "" {
		return nil, ErrEmailRequired
	}
	if username == "" {
		return nil, ErrUsernameRequired
	}
	if len(password) < 8 {
		return nil, ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}

	result, err := s.db.Exec(
		"INSERT INTO users (email, username, password_hash) VALUES (?, ?, ?)",
		email, username, string(hash),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, ErrUserExists
		}
		return nil, err
	}

	id, _ := result.LastInsertId()
	return &models.User{ID: id, Email: email, Username: username}, nil
}

func (s *Service) Login(email, password string) (*models.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	var user models.User
	err := s.db.QueryRow(
		"SELECT id, email, username, password_hash FROM users WHERE email = ?",
		email,
	).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidLogin
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidLogin
	}

	return &user, nil
}

func (s *Service) GetUser(id int64) (*models.User, error) {
	var user models.User
	err := s.db.QueryRow(
		"SELECT id, email, username FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Email, &user.Username)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Service) HasInterests(userID int64) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM interests WHERE user_id = ?", userID).Scan(&count)
	return count > 0, err
}
