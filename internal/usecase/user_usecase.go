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
		Update(ctx context.Context, u model.User) error
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
func (u *UserUsecase) Patch(ctx context.Context, id int, patch model.UserPatch) (model.User, error) {
	user, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return model.User{}, err
	}

	if patch.Email != nil {
		if !model.IsValidEmail(*patch.Email) {
			return user, model.ErrInvalidEmail
		}
		user.Email = *patch.Email
	}
	if patch.Name != nil {
		user.Name = *patch.Name
	}
	if patch.Division != nil {
		user.Division = *patch.Division
	}
	if patch.Password != nil {
		user.Password = *patch.Password
		if err := user.ValidateInput(); err != nil {
			return user, err
		}
		if err := user.HashPassword(); err != nil {
			return user, err
		}
	}

	if err := u.repo.Update(ctx, user); err != nil {
		return model.User{}, err
	}
	return user, nil
}
