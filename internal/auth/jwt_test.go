package auth

import (
	"testing"
	"time"
)

func TestJWTService_GenerateToken(t *testing.T) {
	service := NewJWTService("test-secret-key", 15*time.Minute, 7*24*time.Hour, "konsul-test")

	tests := []struct {
		name     string
		userID   string
		username string
		roles    []string
		wantErr  bool
	}{
		{
			name:     "valid token generation",
			userID:   "user123",
			username: "testuser",
			roles:    []string{"admin", "user"},
			wantErr:  false,
		},
		{
			name:     "empty roles",
			userID:   "user456",
			username: "anotheruser",
			roles:    []string{},
			wantErr:  false,
		},
		{
			name:     "nil roles",
			userID:   "user789",
			username: "thirduser",
			roles:    nil,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := service.GenerateToken(tt.userID, tt.username, tt.roles)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && token == "" {
				t.Error("GenerateToken() returned empty token")
			}
		})
	}
}

func TestJWTService_ValidateToken(t *testing.T) {
	service := NewJWTService("test-secret-key", 15*time.Minute, 7*24*time.Hour, "konsul-test")

	// Generate a valid token
	userID := "user123"
	username := "testuser"
	roles := []string{"admin", "user"}
	token, err := service.GenerateToken(userID, username, roles)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	tests := []struct {
		name      string
		token     string
		wantErr   error
		checkData bool
	}{
		{
			name:      "valid token",
			token:     token,
			wantErr:   nil,
			checkData: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: ErrTokenMissing,
		},
		{
			name:    "invalid token",
			token:   "invalid.token.here",
			wantErr: ErrTokenInvalid,
		},
		{
			name:    "malformed token",
			token:   "not-a-jwt-token",
			wantErr: ErrTokenInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := service.ValidateToken(tt.token)
			if err != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkData {
				if claims == nil {
					t.Error("ValidateToken() returned nil claims")
					return
				}
				if claims.UserID != userID {
					t.Errorf("ValidateToken() userID = %v, want %v", claims.UserID, userID)
				}
				if claims.Username != username {
					t.Errorf("ValidateToken() username = %v, want %v", claims.Username, username)
				}
				if len(claims.Roles) != len(roles) {
					t.Errorf("ValidateToken() roles length = %v, want %v", len(claims.Roles), len(roles))
				}
			}
		})
	}
}

func TestJWTService_TokenExpiration(t *testing.T) {
	// Create service with very short expiry
	service := NewJWTService("test-secret-key", 1*time.Millisecond, 1*time.Millisecond, "konsul-test")

	token, err := service.GenerateToken("user123", "testuser", []string{"admin"})
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Wait for token to expire
	time.Sleep(100 * time.Millisecond)

	_, err = service.ValidateToken(token)
	if err != ErrTokenExpired && err != ErrTokenInvalid {
		t.Errorf("ValidateToken() error = %v, want ErrTokenExpired or ErrTokenInvalid", err)
	}
}

func TestJWTService_GenerateRefreshToken(t *testing.T) {
	service := NewJWTService("test-secret-key", 15*time.Minute, 7*24*time.Hour, "konsul-test")

	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{
			name:    "valid refresh token",
			userID:  "user123",
			wantErr: false,
		},
		{
			name:    "empty user id",
			userID:  "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := service.GenerateRefreshToken(tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateRefreshToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && token == "" {
				t.Error("GenerateRefreshToken() returned empty token")
			}
		})
	}
}

func TestJWTService_ValidateRefreshToken(t *testing.T) {
	service := NewJWTService("test-secret-key", 15*time.Minute, 7*24*time.Hour, "konsul-test")

	// Generate a valid refresh token
	userID := "user123"
	refreshToken, err := service.GenerateRefreshToken(userID)
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	tests := []struct {
		name      string
		token     string
		wantErr   error
		wantID    string
		checkData bool
	}{
		{
			name:      "valid refresh token",
			token:     refreshToken,
			wantErr:   nil,
			wantID:    userID,
			checkData: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: ErrTokenMissing,
		},
		{
			name:    "invalid token",
			token:   "invalid.token.here",
			wantErr: ErrTokenInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := service.ValidateRefreshToken(tt.token)
			if err != tt.wantErr {
				t.Errorf("ValidateRefreshToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkData && id != tt.wantID {
				t.Errorf("ValidateRefreshToken() id = %v, want %v", id, tt.wantID)
			}
		})
	}
}

func TestJWTService_RefreshToken(t *testing.T) {
	service := NewJWTService("test-secret-key", 15*time.Minute, 7*24*time.Hour, "konsul-test")

	// Generate initial tokens
	userID := "user123"
	username := "testuser"
	roles := []string{"admin"}

	refreshToken, err := service.GenerateRefreshToken(userID)
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	tests := []struct {
		name         string
		refreshToken string
		username     string
		roles        []string
		wantErr      bool
	}{
		{
			name:         "valid refresh",
			refreshToken: refreshToken,
			username:     username,
			roles:        roles,
			wantErr:      false,
		},
		{
			name:         "invalid refresh token",
			refreshToken: "invalid.token",
			username:     username,
			roles:        roles,
			wantErr:      true,
		},
		{
			name:         "empty refresh token",
			refreshToken: "",
			username:     username,
			roles:        roles,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newToken, newRefreshToken, err := service.RefreshToken(tt.refreshToken, tt.username, tt.roles)
			if (err != nil) != tt.wantErr {
				t.Errorf("RefreshToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if newToken == "" {
					t.Error("RefreshToken() returned empty new token")
				}
				if newRefreshToken == "" {
					t.Error("RefreshToken() returned empty new refresh token")
				}

				// Validate the new tokens
				claims, err := service.ValidateToken(newToken)
				if err != nil {
					t.Errorf("New token validation failed: %v", err)
				}
				if claims.Username != username {
					t.Errorf("New token username = %v, want %v", claims.Username, username)
				}

				newUserID, err := service.ValidateRefreshToken(newRefreshToken)
				if err != nil {
					t.Errorf("New refresh token validation failed: %v", err)
				}
				if newUserID != userID {
					t.Errorf("New refresh token userID = %v, want %v", newUserID, userID)
				}
			}
		})
	}
}

func TestJWTService_DifferentSecrets(t *testing.T) {
	service1 := NewJWTService("secret-1", 15*time.Minute, 7*24*time.Hour, "konsul-test")
	service2 := NewJWTService("secret-2", 15*time.Minute, 7*24*time.Hour, "konsul-test")

	// Generate token with service1
	token, err := service1.GenerateToken("user123", "testuser", []string{"admin"})
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Try to validate with service2 (different secret)
	_, err = service2.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken() should fail with different secret")
	}
}
