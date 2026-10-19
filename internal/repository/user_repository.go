package repository

import (
	"context"
	"errors"

	"go-backend/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.password_hash, u.full_name, u.role_id, r.code, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.username = $1 AND u.is_active = true
	`
	var user model.User
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.FullName, &user.RoleID, &user.RoleCode, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	if id == "" {
		return nil, nil
	}
	query := `
		SELECT u.id, u.username, u.email, u.password_hash, u.full_name, u.role_id, r.code, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id::text = $1
	`
	var user model.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.FullName, &user.RoleID, &user.RoleCode, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetUserPermissions(ctx context.Context, roleID string) ([]string, error) {
	if roleID == "" {
		return nil, nil
	}
	query := `
		SELECT p.code
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id::text = $1
	`
	rows, err := r.pool.Query(ctx, query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		permissions = append(permissions, p)
	}
	return permissions, nil
}

func (r *UserRepository) GetUserStoreIDs(ctx context.Context, userID string) ([]string, error) {
	if userID == "" {
		return nil, nil
	}
	query := `SELECT store_id FROM user_stores WHERE user_id::text = $1`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var storeIDs []string
	for rows.Next() {
		var sid string
		if err := rows.Scan(&sid); err != nil {
			return nil, err
		}
		storeIDs = append(storeIDs, sid)
	}
	return storeIDs, nil
}

func (r *UserRepository) ListRoles(ctx context.Context) ([]model.Role, error) {
	query := `SELECT id, code, name, COALESCE(description, ''), created_at FROM roles ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []model.Role
	for rows.Next() {
		var role model.Role
		if err := rows.Scan(&role.ID, &role.Code, &role.Name, &role.Description, &role.CreatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *UserRepository) GetRoleByCode(ctx context.Context, code string) (*model.Role, error) {
	query := `SELECT id, code, name, COALESCE(description, ''), created_at FROM roles WHERE code = $1`
	var role model.Role
	err := r.pool.QueryRow(ctx, query, code).Scan(&role.ID, &role.Code, &role.Name, &role.Description, &role.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *UserRepository) GetRoleByID(ctx context.Context, id string) (*model.Role, error) {
	query := `SELECT id, code, name, COALESCE(description, ''), created_at FROM roles WHERE id::text = $1`
	var role model.Role
	err := r.pool.QueryRow(ctx, query, id).Scan(&role.ID, &role.Code, &role.Name, &role.Description, &role.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

func (r *UserRepository) ListUsers(ctx context.Context, roleCode, storeID, search string) ([]model.User, error) {
	query := `
		SELECT 
			u.id, u.username, u.email, u.full_name, u.role_id, r.code, r.name, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE ($1 = '' OR r.code = $1)
		  AND ($2 = '' OR EXISTS (SELECT 1 FROM user_stores us WHERE us.user_id = u.id AND us.store_id::text = $2))
		  AND ($3 = '' OR u.username ILIKE '%' || $3 || '%' OR u.full_name ILIKE '%' || $3 || '%' OR u.email ILIKE '%' || $3 || '%')
		ORDER BY u.created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, roleCode, storeID, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	userMap := make(map[string]int)

	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.Email, &u.FullName, &u.RoleID,
			&u.RoleCode, &u.RoleName, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		userMap[u.ID] = len(users)
		users = append(users, u)
	}

	if len(users) == 0 {
		return users, nil
	}

	// Fetch all user_stores with store info
	storeQuery := `
		SELECT us.user_id, s.id, s.code, s.name
		FROM user_stores us
		JOIN stores s ON us.store_id = s.id
		ORDER BY s.name ASC
	`
	srows, err := r.pool.Query(ctx, storeQuery)
	if err == nil {
		defer srows.Close()
		for srows.Next() {
			var uid string
			var s model.Store
			if err := srows.Scan(&uid, &s.ID, &s.Code, &s.Name); err == nil {
				if idx, ok := userMap[uid]; ok {
					users[idx].StoreIDs = append(users[idx].StoreIDs, s.ID)
					users[idx].Stores = append(users[idx].Stores, s)
				}
			}
		}
	}

	return users, nil
}

func (r *UserRepository) CreateUser(ctx context.Context, u *model.User, passwordHash string, storeIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO users (username, email, password_hash, full_name, role_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	if err := tx.QueryRow(ctx, query, u.Username, u.Email, passwordHash, u.FullName, u.RoleID, u.IsActive).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return err
	}

	for _, sid := range storeIDs {
		if sid == "" {
			continue
		}
		_, err := tx.Exec(ctx, `INSERT INTO user_stores (user_id, store_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, u.ID, sid)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *UserRepository) UpdateUser(ctx context.Context, id string, req model.UpdateUserRequest, newPasswordHash string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if newPasswordHash != "" {
		query := `
			UPDATE users
			SET full_name = $1, email = $2, role_id = $3, password_hash = $4, is_active = COALESCE($5, is_active), updated_at = NOW()
			WHERE id::text = $6
		`
		if _, err := tx.Exec(ctx, query, req.FullName, req.Email, req.RoleID, newPasswordHash, req.IsActive, id); err != nil {
			return err
		}
	} else {
		query := `
			UPDATE users
			SET full_name = $1, email = $2, role_id = $3, is_active = COALESCE($4, is_active), updated_at = NOW()
			WHERE id::text = $5
		`
		if _, err := tx.Exec(ctx, query, req.FullName, req.Email, req.RoleID, req.IsActive, id); err != nil {
			return err
		}
	}

	// Replace store assignments
	if _, err := tx.Exec(ctx, `DELETE FROM user_stores WHERE user_id::text = $1`, id); err != nil {
		return err
	}

	for _, sid := range req.StoreIDs {
		if sid == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO user_stores (user_id, store_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, id, sid); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *UserRepository) SetUserActive(ctx context.Context, id string, isActive bool) error {
	query := `UPDATE users SET is_active = $1, updated_at = NOW() WHERE id::text = $2`
	res, err := r.pool.Exec(ctx, query, isActive, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *UserRepository) CheckUsernameExists(ctx context.Context, username, excludeID string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE username = $1 AND ($2 = '' OR id::text != $2)`
	err := r.pool.QueryRow(ctx, query, username, excludeID).Scan(&count)
	return count > 0, err
}

func (r *UserRepository) CheckEmailExists(ctx context.Context, email, excludeID string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE email = $1 AND ($2 = '' OR id::text != $2)`
	err := r.pool.QueryRow(ctx, query, email, excludeID).Scan(&count)
	return count > 0, err
}
