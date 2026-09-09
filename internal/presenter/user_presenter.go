package presenter

import (
	"errors"
	"example/httpservice/internal/model"
	"example/httpservice/internal/repository"
	"example/httpservice/internal/usecase"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type (
	UserPresenter struct {
		usecase *usecase.UserUsecase
	}
)

func NewUserPresenter(usecase *usecase.UserUsecase) *UserPresenter {
	return &UserPresenter{usecase: usecase}
}

func (u *UserPresenter) List(c *gin.Context) {
	users, err := u.usecase.List(c.Request.Context())
	if err != nil {
		log.Printf("list users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		return
	}
	resp := make([]model.UserResponse, len(users))
	for i, user := range users {
		resp[i] = user.Response()
	}
	c.JSON(http.StatusOK, resp)
}

func (u *UserPresenter) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}
	user, err := u.usecase.GetByID(c.Request.Context(), id)
	if errors.Is(err, repository.ErrUserNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err != nil {
		log.Printf("get user %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		return
	}
	c.JSON(http.StatusOK, user.Response())
}

func (u *UserPresenter) Create(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	createdUser, err := u.usecase.Create(c.Request.Context(), user)
	if errors.Is(err, repository.ErrDuplicateUser) {
		c.JSON(http.StatusConflict, gin.H{"error": "duplicate user "})
		return
	}

	if errors.Is(err, model.ErrAllFieldsRequired) ||
		errors.Is(err, model.ErrInvalidEmail) ||
		errors.Is(err, model.ErrInvalidID) ||
		errors.Is(err, model.ErrInvalidPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err != nil {
		log.Printf("create user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		return
	}
	c.JSON(http.StatusCreated, createdUser.Response())
}

func (u *UserPresenter) Patch(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	updatedUser, err := u.usecase.Patch(c.Request.Context(), id, user)
	if errors.Is(err, repository.ErrUserNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err != nil {
		log.Printf("patch user %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		return
	}
	c.JSON(http.StatusOK, updatedUser.Response())
}

func (u *UserPresenter) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}
	err = u.usecase.Delete(c.Request.Context(), id)
	if errors.Is(err, repository.ErrUserNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err != nil {
		log.Printf("delete user %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		return
	}
	c.Status(http.StatusNoContent)
}
