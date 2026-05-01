package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenExpired = errors.New("token has expired")
	ErrTokenInvalid = errors.New("token is invalid")
	ErrTokenMissing = errors.New("token is missing")
	ErrNotJoinToken = errors.New("token is not a join token")
)

// JoinClaims are embedded in short-lived join tokens issued by the leader.
// They authorise exactly one new node to join the cluster within the TTL.
type JoinClaims struct {
	Purpose string `json:"purpose"` // always "join"
	jwt.RegisteredClaims
}

type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	Policies []string `json:"policies,omitempty"` // ACL policies attached to this token
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	Policies []string `json:"policies,omitempty"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secretKey     []byte
	tokenExpiry   time.Duration
	refreshExpiry time.Duration
	issuer        string
}

func NewJWTService(secretKey string, tokenExpiry, refreshExpiry time.Duration, issuer string) *JWTService {
	return &JWTService{
		secretKey:     []byte(secretKey),
		tokenExpiry:   tokenExpiry,
		refreshExpiry: refreshExpiry,
		issuer:        issuer,
	}
}

func (j *JWTService) GenerateToken(userID, username string, roles []string) (string, error) {
	return j.GenerateTokenWithPolicies(userID, username, roles, nil)
}

// GenerateTokenWithPolicies generates a JWT token with policies
func (j *JWTService) GenerateTokenWithPolicies(userID, username string, roles, policies []string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		Policies: policies,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    j.issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

func (j *JWTService) GenerateRefreshToken(userID, username string, roles []string) (string, error) {
	return j.GenerateRefreshTokenWithPolicies(userID, username, roles, nil)
}

// GenerateRefreshTokenWithPolicies generates a refresh token with user claims.
func (j *JWTService) GenerateRefreshTokenWithPolicies(userID, username string, roles, policies []string) (string, error) {
	claims := RefreshClaims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		Policies: policies,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    j.issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

func (j *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrTokenMissing
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return j.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

func (j *JWTService) ValidateRefreshToken(tokenString string) (*RefreshClaims, error) {
	if tokenString == "" {
		return nil, ErrTokenMissing
	}

	token, err := jwt.ParseWithClaims(tokenString, &RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return j.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	if claims, ok := token.Claims.(*RefreshClaims); ok && token.Valid {
		if claims.UserID == "" && claims.Subject != "" {
			claims.UserID = claims.Subject
		}
		if claims.UserID == "" {
			return nil, ErrTokenInvalid
		}
		return claims, nil
	}

	return nil, ErrTokenInvalid
}

// RefreshToken refreshes an access and refresh token pair from refresh token claims.
func (j *JWTService) RefreshToken(refreshTokenString string) (string, string, error) {
	claims, err := j.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return "", "", err
	}

	newToken, err := j.GenerateTokenWithPolicies(claims.UserID, claims.Username, claims.Roles, claims.Policies)
	if err != nil {
		return "", "", err
	}

	newRefreshToken, err := j.GenerateRefreshTokenWithPolicies(claims.UserID, claims.Username, claims.Roles, claims.Policies)
	if err != nil {
		return "", "", err
	}

	return newToken, newRefreshToken, nil
}

// GenerateJoinToken creates a short-lived JWT authorising a node to join the cluster.
// ttl must be positive; if zero, defaults to 1 hour.
func (j *JWTService) GenerateJoinToken(ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = time.Hour
	}
	claims := JoinClaims{
		Purpose: "join",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    j.issuer,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

// ValidateJoinToken parses and validates a join token.
// Returns ErrNotJoinToken when the purpose claim is not "join".
func (j *JWTService) ValidateJoinToken(tokenString string) error {
	token, err := jwt.ParseWithClaims(tokenString, &JoinClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return j.secretKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return ErrTokenExpired
		}
		return ErrTokenInvalid
	}

	claims, ok := token.Claims.(*JoinClaims)
	if !ok || !token.Valid {
		return ErrTokenInvalid
	}
	if claims.Purpose != "join" {
		return ErrNotJoinToken
	}
	return nil
}
