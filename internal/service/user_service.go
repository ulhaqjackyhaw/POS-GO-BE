package service

import (
	"context"
	"strings"

	"go-backend/internal/model"
	"go-backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	userRepo *repository.UserRepository
	orgRepo  *repository.OrgRepository
}

func NewUserService(userRepo *repository.UserRepository, orgRepo *repository.OrgRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
		orgRepo:  orgRepo,
	}
}

func (s *UserService) ListRoles(ctx context.Context) ([]model.Role, error) {
	return s.userRepo.ListRoles(ctx)
}

func (s *UserService) ListUsers(ctx context.Context, roleCode, storeID, search string) ([]model.User, error) {
	return s.userRepo.ListUsers(ctx, roleCode, storeID, strings.TrimSpace(search))
}

func (s *UserService) GetUser(ctx context.Context, id string) (*model.User, error) {
	if id == "" {
		return nil, model.ErrValidation("ID pengguna tidak valid")
	}
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, model.ErrInternal("Gagal mengambil data pengguna")
	}
	if user == nil {
		return nil, model.ErrNotFound("Pengguna tidak ditemukan")
	}

	// Load store IDs
	storeIDs, err := s.userRepo.GetUserStoreIDs(ctx, id)
	if err == nil {
		user.StoreIDs = storeIDs
	}
	return user, nil
}

func (s *UserService) CreateUser(ctx context.Context, req model.CreateUserRequest) (*model.User, error) {
	req.Username = strings.TrimSpace(strings.ToLower(req.Username))
	req.FullName = strings.TrimSpace(req.FullName)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.Username == "" {
		return nil, model.ErrValidation("Username wajib diisi")
	}
	if req.FullName == "" {
		return nil, model.ErrValidation("Nama lengkap wajib diisi")
	}
	if req.Email == "" {
		return nil, model.ErrValidation("Email wajib diisi")
	}
	if len(req.Password) < 6 {
		return nil, model.ErrValidation("Password minimal 6 karakter")
	}

	// Check username duplicate
	exists, err := s.userRepo.CheckUsernameExists(ctx, req.Username, "")
	if err != nil {
		return nil, model.ErrInternal("Gagal memeriksa ketersediaan username")
	}
	if exists {
		return nil, model.ErrValidation("Username sudah digunakan oleh akun lain")
	}

	// Check email duplicate
	existsEmail, err := s.userRepo.CheckEmailExists(ctx, req.Email, "")
	if err != nil {
		return nil, model.ErrInternal("Gagal memeriksa ketersediaan email")
	}
	if existsEmail {
		return nil, model.ErrValidation("Email sudah terdaftar pada akun lain")
	}

	// Resolve role
	var role *model.Role
	if req.RoleID != "" {
		role, err = s.userRepo.GetRoleByID(ctx, req.RoleID)
	} else if req.RoleCode != "" {
		role, err = s.userRepo.GetRoleByCode(ctx, req.RoleCode)
	} else {
		// Default to cashier if not specified
		role, err = s.userRepo.GetRoleByCode(ctx, "cashier")
	}

	if err != nil || role == nil {
		return nil, model.ErrValidation("Role / jabatan yang dipilih tidak valid")
	}

	// Hash password
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, model.ErrInternal("Gagal memproses enkripsi password")
	}

	newUser := &model.User{
		Username: req.Username,
		Email:    req.Email,
		FullName: req.FullName,
		RoleID:   role.ID,
		RoleCode: role.Code,
		RoleName: role.Name,
		IsActive: true,
		StoreIDs: req.StoreIDs,
	}

	if err := s.userRepo.CreateUser(ctx, newUser, string(hashed), req.StoreIDs); err != nil {
		return nil, model.ErrInternal("Gagal menyimpan akun pengguna: " + err.Error())
	}

	return newUser, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id string, req model.UpdateUserRequest) (*model.User, error) {
	if id == "" {
		return nil, model.ErrValidation("ID pengguna tidak valid")
	}

	existing, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, model.ErrInternal("Gagal memeriksa pengguna")
	}
	if existing == nil {
		return nil, model.ErrNotFound("Pengguna tidak ditemukan")
	}

	req.FullName = strings.TrimSpace(req.FullName)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if req.FullName == "" {
		req.FullName = existing.FullName
	}
	if req.Email == "" {
		req.Email = existing.Email
	}

	// Check email uniqueness if modified
	if req.Email != existing.Email {
		exists, err := s.userRepo.CheckEmailExists(ctx, req.Email, id)
		if err != nil {
			return nil, model.ErrInternal("Gagal memeriksa ketersediaan email")
		}
		if exists {
			return nil, model.ErrValidation("Email sudah digunakan oleh akun lain")
		}
	}

	// Resolve role
	if req.RoleID == "" && req.RoleCode != "" {
		role, err := s.userRepo.GetRoleByCode(ctx, req.RoleCode)
		if err == nil && role != nil {
			req.RoleID = role.ID
		}
	}
	if req.RoleID == "" {
		req.RoleID = existing.RoleID
	}

	var newHash string
	if req.Password != "" {
		if len(req.Password) < 6 {
			return nil, model.ErrValidation("Password baru minimal 6 karakter")
		}
		h, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, model.ErrInternal("Gagal mengenkripsi password baru")
		}
		newHash = string(h)
	}

	if err := s.userRepo.UpdateUser(ctx, id, req, newHash); err != nil {
		return nil, model.ErrInternal("Gagal memperbarui data pengguna: " + err.Error())
	}

	return s.GetUser(ctx, id)
}

func (s *UserService) SetUserActive(ctx context.Context, id string, isActive bool) error {
	if id == "" {
		return model.ErrValidation("ID pengguna tidak valid")
	}
	return s.userRepo.SetUserActive(ctx, id, isActive)
}
