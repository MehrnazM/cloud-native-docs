package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MehrnazM/cloud-native-docs/internal/repository"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrEmailTaken = errors.New("email already taken")

type AuthService struct {
	repo      *repository.UsersRepository
	jwtSecret []byte
	tokenTTL  time.Duration
}

func NewAuthService(repo *repository.UsersRepository, jwtSecret string) *AuthService {
	return &AuthService{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  24 * time.Hour,
	}
}

func (s *AuthService) Register(ctx context.Context, email, password string) error {

	id := uuid.New()
	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return fmt.Errorf("Get password hash failed: %w", err)
	}

	err = s.repo.RegisterUser(ctx, id, email, string(passHash))
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrEmailTaken
		}
		return fmt.Errorf("failed to register user: %w", err)
	}
	return nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("GetUserByEmail failed: %w", err)
	}

	if user == nil {
		return "", ErrInvalidCredentials
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", ErrInvalidCredentials
	}

	claims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"exp":     time.Now().Add(s.tokenTTL).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, err
}
