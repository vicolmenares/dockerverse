package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// User represents a system user with authentication and authorization
type User struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	FirstName      string    `json:"firstName"`
	LastName       string    `json:"lastName"`
	Avatar         string    `json:"avatar,omitempty"` // Base64 encoded avatar or URL
	PasswordHash   string    `json:"passwordHash,omitempty"`
	Role           string    `json:"role"` // "admin" or "user"
	EmailConfirmed bool      `json:"emailConfirmed"`
	ConfirmToken   string    `json:"-"`
	TOTPSecret     string    `json:"-"`           // TOTP secret (not exposed to API)
	TOTPEnabled    bool      `json:"totpEnabled"` // Whether 2FA is enabled
	RecoveryCodes  []string  `json:"-"`           // Recovery codes (not exposed to API)
	CreatedAt      time.Time `json:"createdAt"`
	LastLogin      time.Time `json:"lastLogin,omitempty"`
}

// ResetCode represents a password reset code
type ResetCode struct {
	Code      string
	UserID    string
	ExpiresAt time.Time
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// LoginRequest represents a login attempt
type LoginRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	RememberMe   bool   `json:"rememberMe"`
	TOTPCode     string `json:"totpCode,omitempty"`
	RecoveryCode string `json:"recoveryCode,omitempty"`
}

// LoginResponse represents a successful login
type LoginResponse struct {
	User         *User      `json:"user"`
	Tokens       AuthTokens `json:"tokens"`
	RequiresTOTP bool       `json:"requiresTOTP,omitempty"`
	TempToken    string     `json:"tempToken,omitempty"` // Temporary token for 2FA verification
}

// AuthTokens represents authentication tokens
type AuthTokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
	TokenType    string `json:"tokenType"`
}

// CreateUserRequest represents a request to create a new user
type CreateUserRequest struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Role      string `json:"role"`
}

// UpdateUserRequest represents a request to update user details
type UpdateUserRequest struct {
	Email     string `json:"email,omitempty"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Password  string `json:"password,omitempty"`
	Role      string `json:"role,omitempty"`
}
