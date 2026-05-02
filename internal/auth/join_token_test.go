package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestJWTService() *JWTService {
	return NewJWTService("test-secret", time.Hour, 24*time.Hour, "konsul-test")
}

func TestGenerateJoinToken_ReturnsNonEmptyToken(t *testing.T) {
	svc := newTestJWTService()
	token, err := svc.GenerateJoinToken(time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestGenerateJoinToken_ZeroTTLDefaultsToOneHour(t *testing.T) {
	svc := newTestJWTService()
	token, err := svc.GenerateJoinToken(0)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateJoinToken_ValidToken(t *testing.T) {
	svc := newTestJWTService()
	token, err := svc.GenerateJoinToken(time.Hour)
	require.NoError(t, err)

	err = svc.ValidateJoinToken(token)
	assert.NoError(t, err)
}

func TestValidateJoinToken_ExpiredToken(t *testing.T) {
	svc := newTestJWTService()
	// Build an expired token directly (bypass GenerateJoinToken's TTL guard).
	claims := JoinClaims{
		Purpose: "join",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "konsul-test",
		},
	}
	raw := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err := raw.SignedString([]byte("test-secret"))
	require.NoError(t, err)

	err = svc.ValidateJoinToken(token)
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestValidateJoinToken_WrongSecret(t *testing.T) {
	svc1 := newTestJWTService()
	svc2 := NewJWTService("other-secret", time.Hour, 24*time.Hour, "konsul-test")

	token, err := svc1.GenerateJoinToken(time.Hour)
	require.NoError(t, err)

	err = svc2.ValidateJoinToken(token)
	assert.Error(t, err)
}

func TestValidateJoinToken_RegularUserTokenIsRejected(t *testing.T) {
	svc := newTestJWTService()
	// Generate a regular user token (not a join token).
	userToken, err := svc.GenerateToken("user1", "alice", []string{"admin"})
	require.NoError(t, err)

	err = svc.ValidateJoinToken(userToken)
	assert.ErrorIs(t, err, ErrNotJoinToken)
}

func TestValidateJoinToken_EmptyString(t *testing.T) {
	svc := newTestJWTService()
	err := svc.ValidateJoinToken("")
	assert.Error(t, err)
}

func TestValidateJoinToken_Garbage(t *testing.T) {
	svc := newTestJWTService()
	err := svc.ValidateJoinToken("not.a.jwt")
	assert.Error(t, err)
}
