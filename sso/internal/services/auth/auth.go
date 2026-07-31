package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sso/internal/domain/models"
	"sso/internal/lib/jwt"
	"sso/internal/lib/logger/sl"
	"sso/internal/storage"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type UserSaver interface {
	SaveUser(ctx context.Context, email string, wallet string, hashedPassword []byte) (id int64, err error)
}

type UserProvider interface {
	GetUser(ctx context.Context, email string) (models.User, error)
}

type BlackListProvider interface {
	SetToken(ctx context.Context, jti string, expiresAt time.Time) error
	GetToken(ctx context.Context, jti string) error
}

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUserExists          = errors.New("user already exists")
	ErrRefreshTokenRevoked = errors.New("token is revoked")
	ErrRefreshTokenExpired = errors.New("token is expired")
	ErrUserNotFound        = errors.New("user not found")
)

type TokenResponse struct {
	AccessToken  string
	RefreshToken string
}

type Auth struct {
	log               *slog.Logger
	userSaver         UserSaver
	userProvider      UserProvider
	blackListProvider BlackListProvider
	publicKeyPath     string
	privateKeyPath    string
	accessTTL         time.Duration
	refreshTTL        time.Duration
}

func New(
	log *slog.Logger,
	userSaver UserSaver,
	userProvider UserProvider,
	blackListProvider BlackListProvider,
	publicKeyPath string,
	privateKeyPath string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) *Auth {
	return &Auth{
		log:               log,
		userSaver:         userSaver,
		userProvider:      userProvider,
		blackListProvider: blackListProvider,
		publicKeyPath:     publicKeyPath,
		privateKeyPath:    privateKeyPath,
		accessTTL:         accessTTL,
		refreshTTL:        refreshTTL,
	}
}

func (a *Auth) Register(
	ctx context.Context, email string, wallet string, password1 string, password2 string,
) (userID int64, err error) {
	const op = "Auth.Register"

	log := a.log.With(slog.String("op", op), slog.String("email", email))

	log.Info("registering user")

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password1), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to hash password", sl.Err(err))

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := a.userSaver.SaveUser(ctx, email, wallet, hashedPassword)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			log.Warn("user already exists", sl.Err(err))

			return 0, fmt.Errorf("%s: %w", op, ErrUserExists)
		}

		log.Error("failed to save user", sl.Err(err))

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user registered")

	return id, nil
}

func (a *Auth) Login(ctx context.Context, email string, password string) (TokenResponse, error) {
	const op = "Auth.Login"

	log := a.log.With(slog.String("op", op), slog.String("email", email))

	log.Info("attempting to login user")

	user, err := a.userProvider.GetUser(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Warn("user not found", sl.Err(err))

			return TokenResponse{}, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}

		log.Error("failed to get user", sl.Err(err))

		return TokenResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)); err != nil {
		log.Info("invalid credentials", sl.Err(err))

		return TokenResponse{}, fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	accessToken, err := jwt.EncodeJWT(user, a.accessTTL, a.privateKeyPath, "access")
	if err != nil {
		log.Error("failed to encode access token", sl.Err(err))

		return TokenResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	refreshToken, err := jwt.EncodeJWT(user, a.refreshTTL, a.privateKeyPath, "refresh")
	if err != nil {
		log.Error("failed to encode refresh token", sl.Err(err))

		return TokenResponse{}, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("user logged in successfully")

	return TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (a *Auth) Refresh(ctx context.Context, refreshToken string) (string, error) {
	const op = "Auth.Refresh"

	log := a.log.With(slog.String("op", op))

	log.Info("attempting to refresh and access tokens")

	payload, err := jwt.DecodeJWT(refreshToken, a.publicKeyPath)
	if err != nil {
		if errors.Is(err, jwt.ErrRefreshTokenExpired) {
			log.Warn("token is expired", sl.Err(err))

			return "", fmt.Errorf("%s: %w", op, ErrRefreshTokenExpired)
		}

		log.Error("failed to generate tokens", sl.Err(err))

		return "", fmt.Errorf("%s: %w", op, err)
	}

	err = a.blackListProvider.GetToken(ctx, payload["jti"].(string))
	if err != nil {
		if errors.Is(err, storage.ErrRefreshTokenRevoked) {
			log.Warn("token is revoked", sl.Err(err))

			return "", fmt.Errorf("%s: %w", op, ErrRefreshTokenRevoked)
		}

		log.Error("failed to get token from blacklist", sl.Err(err))

		return "", fmt.Errorf("%s: %w", op, err)
	}

	user := models.User{
		ID:           int64(payload["uid"].(float64)),
		Email:        payload["email"].(string),
		Wallet:       payload["wallet"].(string),
		PasswordHash: nil,
	}

	accessToken, err := jwt.EncodeJWT(user, a.accessTTL, a.privateKeyPath, "access")
	if err != nil {
		log.Error("failed to endode access token", sl.Err(err))

		return "", fmt.Errorf("%s: %w", op, err)
	}

	log.Info("access token successfully received")

	return accessToken, nil
}

func (a *Auth) Logout(ctx context.Context, refreshToken string) error {
	const op = "Auth.Logout"

	log := a.log.With(slog.String("op", op))

	log.Info("attempting to logout")

	payload, err := jwt.DecodeJWT(refreshToken, a.publicKeyPath)
	if err != nil {
		if errors.Is(err, jwt.ErrRefreshTokenExpired) {
			return nil
		}

		log.Error("failed decode JWT refresh", sl.Err(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	expiresAt := time.Unix(int64(payload["exp"].(float64)), 0)

	err = a.blackListProvider.SetToken(ctx, payload["jti"].(string), expiresAt)
	if err != nil {
		if errors.Is(err, storage.ErrRefreshTokenRevoked) {
			return nil
		}

		log.Error("failed to set token access", sl.Err(err))

		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("logout is successfully")

	return nil
}
