package usecase

import (
	"context"
	"example/httpservice/internal/model"
)

type (
	UserRepoInterface interface {
		List(ctx context.Context) ([]model.User, error)
		GetByID(ctx context.Context, id int) (model.User, error)
		Create(ctx context.Context, u model.User) (model.User, error)
		Patch(ctx context.Context, id int, u model.User) error
		Delete(ctx context.Context, id int) error
	}

	UserUsecase struct {
		repo UserRepoInterface
	}
)

func NewUserUsecase(repo UserRepoInterface) *UserUsecase {
	return &UserUsecase{repo: repo}
}

// Create implements [UserUsecaseInterface].
func (u *UserUsecase) Create(ctx context.Context, user model.User) (model.User, error) {
	if err := user.ValidateInput(); err != nil {
		return user, err
	}
	if err := user.HashPassword(); err != nil {
		return user, err
	}
	return u.repo.Create(ctx, user)
}

// Delete implements [UserUsecaseInterface].
func (u *UserUsecase) Delete(ctx context.Context, id int) error {
	return u.repo.Delete(ctx, id)
}

// GetByID implements [UserUsecaseInterface].
func (u *UserUsecase) GetByID(ctx context.Context, id int) (model.User, error) {
	return u.repo.GetByID(ctx, id)
}

// List implements [UserUsecaseInterface].
func (u *UserUsecase) List(ctx context.Context) ([]model.User, error) {
	return u.repo.List(ctx)
}

// Patch implements [UserUsecaseInterface].
func (u *UserUsecase) Patch(ctx context.Context, id int, user model.User) (model.User, error) {
	if err := user.ValidatePatch(); err != nil {
		return user, err
	}

	if err := u.repo.Patch(ctx, id, user); err != nil {
		return model.User{}, err
	}

	updatedUser, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return model.User{}, err
	}

	return updatedUser, nil
}
