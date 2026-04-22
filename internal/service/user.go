package service

import (
	"context"
	"errors"
	"strings"

	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/auth"
	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/models"
	"github.com/DimitryShR/go-ya-practicum-gophermart/internal/repository"
)

type UserService struct {
	repo      UserRepository
	passwords *auth.PasswordManager
}

// Cоздает сервис пользователей.
func NewUserService(repo UserRepository, passwords *auth.PasswordManager) *UserService {
	return &UserService{
		repo:      repo,
		passwords: passwords,
	}
}

// Создает нового пользователя
func (s *UserService) Register(ctx context.Context, credentials models.Credentials) (models.User, error) {
	login := strings.TrimSpace(credentials.Login)
	password := strings.TrimSpace(credentials.Password)
	if login == "" || password == "" {
		return models.User{}, ErrInvalidCredentials
	}

	passwordHash, err := s.passwords.Hash(password)
	if err != nil {
		return models.User{}, err
	}

	user, err := s.repo.CreateUser(ctx, login, passwordHash)
	if err != nil {
		if errors.Is(err, repository.ErrUserAlreadyExists) {
			return models.User{}, ErrLoginAlreadyExists
		}
		return models.User{}, err
	}

	return user, nil
}

// Аутентифицирует пользователя по логину и паролю
func (s *UserService) Login(ctx context.Context, credentials models.Credentials) (models.User, error) {
	login := strings.TrimSpace(credentials.Login)
	password := strings.TrimSpace(credentials.Password)
	if login == "" || password == "" {
		return models.User{}, ErrInvalidCredentials
	}

	user, err := s.repo.UserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return models.User{}, ErrInvalidCredentials
		}
		return models.User{}, err
	}

	match, err := s.passwords.Compare(user.PasswordHash, password)
	if err != nil {
		return models.User{}, err
	}
	if !match {
		return models.User{}, ErrInvalidCredentials
	}

	return user, nil
}
