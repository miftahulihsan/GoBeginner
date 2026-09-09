package repository

import (
	"context"
	"errors"
	"os"
	"testing"

	"example/httpservice/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://shop:shoppass@localhost:5432/shop"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		os.Exit(1)
	}

	testPool = pool
	os.Exit(m.Run())
}

func setup(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(context.Background(), "TRUNCATE users RESTART IDENTITY")
	if err != nil {
		t.Fatalf("failed to truncate: %v", err)
	}
}

func TestUserRepository_CRUD(t *testing.T) {
	setup(t)
	repo := NewUserRepository(testPool)
	ctx := context.Background()

	user := model.User{
		NIK:      "P95912",
		Email:    "john.doe@example.com",
		Name:     "John Doe",
		Division: "Engineering",
		Password: "Passw0rd!",
	}

	// 1. Create
	created, err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.ID <= 0 {
		t.Errorf("Expected ID > 0, got %d", created.ID)
	}

	// 2. Get
	found, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if found.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, found.Email)
	}

	// 3. Update
	found.Name = "John Smith"
	found.Division = "Marketing"
	err = repo.Patch(ctx, found.ID, found)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}

	updated, _ := repo.GetByID(ctx, found.ID)
	if updated.Name != "John Smith" {
		t.Errorf("Expected name 'John Smith', got '%s'", updated.Name)
	}

	// 4. Delete
	err = repo.Delete(ctx, found.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = repo.GetByID(ctx, found.ID)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	setup(t)
	repo := NewUserRepository(testPool)

	_, err := repo.GetByID(context.Background(), 999)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}
}
