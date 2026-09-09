package repository

import (
	"context"
	"errors"

	"example/httpservice/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserNotFound = errors.New("user not found")
var ErrDuplicateUser = errors.New("duplicate user")

type (
	UserRepoInterface interface {
		List(ctx context.Context) ([]model.User, error)
		GetByID(ctx context.Context, id int) (model.User, error)
		Create(ctx context.Context, u model.User) (model.User, error)
		Patch(ctx context.Context, id int, u model.User) error
		Delete(ctx context.Context, id int) error
	}

	UserRepository struct {
		db *pgxpool.Pool
	}
)

func NewUserRepository(db *pgxpool.Pool) UserRepoInterface {
	return &UserRepository{db: db}
}

func (r *UserRepository) List(ctx context.Context) ([]model.User, error) {
	rows, err := r.db.Query(ctx, `SELECT id, nik, email, name, division, password FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]model.User, 0)
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.NIK, &u.Email, &u.Name, &u.Division, &u.Password); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int) (model.User, error) {
	var u model.User
	err := r.db.QueryRow(ctx,
		`SELECT id, nik, email, name, division, password FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.NIK, &u.Email, &u.Name, &u.Division, &u.Password)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, ErrUserNotFound
	}
	if err != nil {
		return u.Validate()
	}
	return u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (model.User, error) {
	var u model.User
	err := r.db.QueryRow(ctx,
		`SELECT id, nik, email, name, division, password FROM users WHERE email = $1`, email).
		Scan(&u.ID, &u.NIK, &u.Email, &u.Name, &u.Division, &u.Password)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrUserNotFound
	}
	if err != nil {
		return u.Validate()
	}
	return u, nil
}

func (r *UserRepository) Create(ctx context.Context, u model.User) (model.User, error) {
	var pgErr *pgconn.PgError
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (nik, email, name, division, password) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		u.NIK, u.Email, u.Name, u.Division, u.Password).Scan(&u.ID)
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return model.User{}, ErrDuplicateUser
	}
	if err != nil {
		return model.User{}, err
	}
	return u, nil
}

func (r *UserRepository) Patch(ctx context.Context, id int, u model.User) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE users
		 SET
			nik = COALESCE(NULLIF($1, ''), nik),
			email = COALESCE(NULLIF($2, ''), email),
			name = COALESCE(NULLIF($3, ''), name),
			division = COALESCE(NULLIF($4, ''), division),
			password = COALESCE(NULLIF($5, ''), password)
		 WHERE id = $6`,
		u.NIK,
		u.Email,
		u.Name,
		u.Division,
		u.Password,
		id,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id int) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}
