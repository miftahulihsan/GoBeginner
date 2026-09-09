package model

import (
	"errors"
	"fmt"
	"net/mail"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidEmail      = errors.New("invalid email format")
	ErrInvalidPassword   = errors.New("password must contain at least one uppercase letter, one lowercase letter, one number, and one symbol")
	ErrInvalidID         = errors.New("ID must be alphanumeric and start with 'P'")
	ErrDuplicateUser     = errors.New("duplicate user")
	ErrAllFieldsRequired = errors.New("all fields are required")
)

// func (u *User) IsValidationErr(err error) bool {
// 	return errors.Is(err, ErrInvalidEmail) ||
// 		errors.Is(err, ErrInvalidPassword) ||
// 		errors.Is(err, ErrInvalidID) ||
// 		errors.Is(err, ErrAllFieldsRequired) ||
// 		errors.Is(err, ErrDuplicateUser)
// }

type User struct {
	ID       int    `json:"id"`  // e.g., 1
	NIK      string `json:"nik"` // e.g., P12345
	Email    string `json:"email"`
	Name     string `json:"name"`     // required
	Division string `json:"division"` // required
	Password string `json:"password"` // plain text on input, hashed on storage
}

func (u *User) Validate() (User, error) {
	return *u, nil
}

type UserResponse struct {
	ID       int    `json:"id"`
	NIK      string `json:"nik"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Division string `json:"division"`
}

// type Response struct {
// 	User      UserResponse   `json:"user"`
// 	UserArray []UserResponse `json:"users"`
// }

type UserPatch struct {
	Email    *string `json:"email"`
	Name     *string `json:"name"`
	Division *string `json:"division"`
	Password *string `json:"password"`
}

func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

var (
	passwordLower  = regexp.MustCompile(`[a-z]`)
	passwordUpper  = regexp.MustCompile(`[A-Z]`)
	passwordNumber = regexp.MustCompile(`[0-9]`)
	passwordSymbol = regexp.MustCompile(`[^a-zA-Z0-9]`)
	uidPattern     = regexp.MustCompile(`^P[A-Za-z0-9]{4,6}$`)
)

// ValidateInput checks the User struct's fields for required constraints
func (u *User) ValidateInput() error {
	// Check for required fields
	if u.NIK == "" || u.Email == "" || u.Name == "" || u.Division == "" || u.Password == "" {
		return ErrAllFieldsRequired
	}

	// Validate NIK format
	if !uidPattern.MatchString(u.NIK) {
		return ErrInvalidID
	}

	// Validate Email format
	if !IsValidEmail(u.Email) {
		return ErrInvalidEmail
	}

	// Validate Password strength
	if !isStrongPassword(u.Password) {
		return ErrInvalidPassword
	}

	return nil
}

func (u *User) ValidatePatch() error {
	if u.Email != "" && !IsValidEmail(u.Email) {
		return ErrInvalidEmail
	}

	if u.NIK != "" && !uidPattern.MatchString(u.NIK) {
		return ErrInvalidID
	}

	if u.Password != "" && !isStrongPassword(u.Password) {
		return ErrInvalidPassword
	}

	return nil
}

func isStrongPassword(p string) bool {
	if len(p) < 8 {
		return false
	}
	if !passwordLower.MatchString(p) {
		return false
	}
	if !passwordUpper.MatchString(p) {
		return false
	}
	if !passwordNumber.MatchString(p) {
		return false
	}
	if !passwordSymbol.MatchString(p) {
		return false
	}
	return true
}

// HashPassword securely hashes the password using bcrypt.
func (u *User) HashPassword() error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	u.Password = string(hashed)
	return nil
}

func (u User) Response() UserResponse {
	return UserResponse{
		ID:       u.ID,
		NIK:      u.NIK,
		Email:    u.Email,
		Name:     u.Name,
		Division: u.Division,
	}
}
