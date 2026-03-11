package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charan/url-shortener/internal/domain"
	"github.com/charan/url-shortener/internal/repository/postgres"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/argon2"
	"google.golang.org/api/idtoken"
)

const (
	argonTime    = 1
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	saltLen      = 16
	accessTTL    = 15 * time.Minute
	refreshTTL   = 30 * 24 * time.Hour
)

type AuthService struct {
	userRepo           *postgres.UserRepo
	refreshSessionRepo *postgres.RefreshSessionRepo
	jwtSecret          string
	googleClientID     string
}

func NewAuthService(userRepo *postgres.UserRepo, refreshSessionRepo *postgres.RefreshSessionRepo, jwtSecret, googleClientID string) *AuthService {
	return &AuthService{
		userRepo:           userRepo,
		refreshSessionRepo: refreshSessionRepo,
		jwtSecret:          jwtSecret,
		googleClientID:     strings.TrimSpace(googleClientID),
	}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*domain.User, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	return s.userRepo.Create(ctx, email, hash)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, *domain.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", nil, errors.New("invalid credentials")
	}

	if !verifyPassword(password, user.PasswordHash) {
		return "", "", nil, errors.New("invalid credentials")
	}

	accessToken, err := s.issueToken(user.ID, "access", accessTTL)
	if err != nil {
		return "", "", nil, err
	}

	familyID, err := s.generateTokenID()
	if err != nil {
		return "", "", nil, fmt.Errorf("generate refresh family id: %w", err)
	}

	refreshJTI, err := s.generateTokenID()
	if err != nil {
		return "", "", nil, fmt.Errorf("generate refresh jti: %w", err)
	}

	refreshToken, err := s.issueRefreshToken(user.ID, familyID, refreshJTI)
	if err != nil {
		return "", "", nil, err
	}

	if err := s.refreshSessionRepo.Create(ctx, user.ID, familyID, refreshJTI, time.Now().Add(refreshTTL), nil); err != nil {
		return "", "", nil, err
	}

	return accessToken, refreshToken, &domain.User{
		ID:    user.ID,
		Email: user.Email,
	}, nil
}

func (s *AuthService) LoginWithGoogle(ctx context.Context, credential string) (string, string, *domain.User, error) {
	if s.googleClientID == "" {
		return "", "", nil, errors.New("google oauth is not configured")
	}

	payload, err := idtoken.Validate(ctx, credential, s.googleClientID)
	if err != nil {
		return "", "", nil, errors.New("invalid google credential")
	}

	emailRaw, ok := payload.Claims["email"].(string)
	if !ok || strings.TrimSpace(emailRaw) == "" {
		return "", "", nil, errors.New("google account email is unavailable")
	}

	email := strings.ToLower(strings.TrimSpace(emailRaw))
	user, err := s.getOrCreateGoogleUser(ctx, email)
	if err != nil {
		return "", "", nil, err
	}

	accessToken, err := s.issueToken(user.ID, "access", accessTTL)
	if err != nil {
		return "", "", nil, err
	}

	familyID, err := s.generateTokenID()
	if err != nil {
		return "", "", nil, fmt.Errorf("generate refresh family id: %w", err)
	}

	refreshJTI, err := s.generateTokenID()
	if err != nil {
		return "", "", nil, fmt.Errorf("generate refresh jti: %w", err)
	}

	refreshToken, err := s.issueRefreshToken(user.ID, familyID, refreshJTI)
	if err != nil {
		return "", "", nil, err
	}

	if err := s.refreshSessionRepo.Create(ctx, user.ID, familyID, refreshJTI, time.Now().Add(refreshTTL), nil); err != nil {
		return "", "", nil, err
	}

	return accessToken, refreshToken, user, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (string, string, *domain.User, error) {
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return "", "", nil, errors.New("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", nil, errors.New("invalid refresh token")
	}

	if tokenType, _ := claims["typ"].(string); tokenType != "refresh" {
		return "", "", nil, errors.New("invalid refresh token")
	}

	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return "", "", nil, errors.New("invalid refresh token")
	}

	refreshJTI, ok := claims["jti"].(string)
	if !ok || refreshJTI == "" {
		return "", "", nil, errors.New("invalid refresh token")
	}

	familyID, ok := claims["fid"].(string)
	if !ok || familyID == "" {
		return "", "", nil, errors.New("invalid refresh token")
	}

	nextRefreshJTI, err := s.generateTokenID()
	if err != nil {
		return "", "", nil, fmt.Errorf("generate next refresh jti: %w", err)
	}

	rotated, err := s.refreshSessionRepo.Rotate(ctx, userID, familyID, refreshJTI, nextRefreshJTI, time.Now().Add(refreshTTL))
	if err != nil {
		return "", "", nil, err
	}
	if !rotated {
		return "", "", nil, errors.New("invalid refresh token")
	}

	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return "", "", nil, errors.New("invalid refresh token")
	}

	accessToken, err := s.issueToken(user.ID, "access", accessTTL)
	if err != nil {
		return "", "", nil, err
	}

	newRefreshToken, err := s.issueRefreshToken(user.ID, familyID, nextRefreshJTI)
	if err != nil {
		return "", "", nil, err
	}

	return accessToken, newRefreshToken, user, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return errors.New("invalid refresh token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return errors.New("invalid refresh token")
	}

	if tokenType, _ := claims["typ"].(string); tokenType != "refresh" {
		return errors.New("invalid refresh token")
	}

	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return errors.New("invalid refresh token")
	}

	familyID, ok := claims["fid"].(string)
	if !ok || familyID == "" {
		return errors.New("invalid refresh token")
	}

	if err := s.refreshSessionRepo.RevokeFamily(ctx, userID, familyID, "logout"); err != nil {
		return err
	}

	return nil
}

func (s *AuthService) issueToken(userID, tokenType string, ttl time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"typ": tokenType,
		"exp": time.Now().Add(ttl).Unix(),
		"iat": time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

func (s *AuthService) issueRefreshToken(userID, familyID, tokenJTI string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"typ": "refresh",
		"fid": familyID,
		"jti": tokenJTI,
		"exp": time.Now().Add(refreshTTL).Unix(),
		"iat": time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return tokenString, nil
}

func (s *AuthService) generateTokenID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *AuthService) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

func (s *AuthService) getOrCreateGoogleUser(ctx context.Context, email string) (*domain.User, error) {
	authUser, err := s.userRepo.GetByEmail(ctx, email)
	if err == nil {
		return s.userRepo.GetByID(ctx, authUser.ID)
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("lookup google user: %w", err)
	}

	placeholderPassword, hashErr := hashPassword("google-oauth-no-password")
	if hashErr != nil {
		return nil, fmt.Errorf("hash placeholder password: %w", hashErr)
	}

	created, createErr := s.userRepo.Create(ctx, email, placeholderPassword)
	if createErr == nil {
		return created, nil
	}

	if !postgres.IsUniqueViolation(createErr) {
		return nil, createErr
	}

	authUser, err = s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("lookup google user after conflict: %w", err)
	}

	return s.userRepo.GetByID(ctx, authUser.ID)
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	keyB64 := base64.RawStdEncoding.EncodeToString(key)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, saltB64, keyB64), nil
}

func verifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return false
	}

	var memory uint32
	var iterations uint32
	var threads uint8
	_, _ = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &threads)

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expectedKey, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	key := argon2.IDKey([]byte(password), salt, iterations, memory, threads, uint32(len(expectedKey)))

	return subtle.ConstantTimeCompare(key, expectedKey) == 1
}
