package service

import (
	"context"
	"time"

	"go-backend/internal/model"
	"go-backend/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo           *repository.UserRepository
	jwtSecret          string
	jwtExpirationHours int
}

func NewAuthService(userRepo *repository.UserRepository, jwtSecret string, jwtExpirationHours int) *AuthService {
	return &AuthService{
		userRepo:           userRepo,
		jwtSecret:          jwtSecret,
		jwtExpirationHours: jwtExpirationHours,
	}
}

func (s *AuthService) Login(ctx context.Context, req model.LoginRequest) (*model.LoginResponse, error) {
	if req.Username == "" || req.Password == "" {
		return nil, model.ErrValidation("Username dan password wajib diisi")
	}

	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, model.ErrInternal("Gagal memproses autentikasi")
	}

	// PRD rule FR-01: "Login gagal memberikan pesan aman tanpa membocorkan apakah username atau password tertentu benar"
	if user == nil {
		return nil, model.ErrUnauthenticated("Kredensial login tidak valid")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, model.ErrUnauthenticated("Kredensial login tidak valid")
	}

	// Fetch permissions
	perms, err := s.userRepo.GetUserPermissions(ctx, user.RoleID)
	if err != nil {
		return nil, model.ErrInternal("Gagal memuat permission pengguna")
	}

	// Fetch user store IDs
	storeIDs, err := s.userRepo.GetUserStoreIDs(ctx, user.ID)
	if err != nil {
		return nil, model.ErrInternal("Gagal memuat cabang pengguna")
	}

	// Create JWT token
	expiresInSec := s.jwtExpirationHours * 3600
	expirationTime := time.Now().Add(time.Duration(s.jwtExpirationHours) * time.Hour)

	claims := jwt.MapClaims{
		"user_id":     user.ID,
		"username":    user.Username,
		"role_code":   user.RoleCode,
		"permissions": perms,
		"store_ids":   storeIDs,
		"exp":         expirationTime.Unix(),
		"iat":         time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, model.ErrInternal("Gagal membuat token autentikasi")
	}

	return &model.LoginResponse{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresIn:   expiresInSec,
		User: model.UserInfo{
			ID:          user.ID,
			Username:    user.Username,
			FullName:    user.FullName,
			Email:       user.Email,
			Role:        user.RoleCode,
			Permissions: perms,
			StoreIDs:    storeIDs,
		},
	}, nil
}
