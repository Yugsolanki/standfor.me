package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/Yugsolanki/standfor-me/internal/config"
	"github.com/Yugsolanki/standfor-me/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Sentinel errors returned from this package so callers can
// use errors.Is without depending on jwt library types.
var (
	ErrTokenExpired   = errors.New("token has expired")
	ErrTokenInvalid   = errors.New("token is invalid")
	ErrTokenMalformed = errors.New("token is malformed")
)

// accessClaims are the registered + custom claims embedded
// inside every access token we issue.
type accessClaims struct {
	UserID string `json:"uid"`
	Role   string `json:"role"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Service handles all JWT operations: issuance and validation.
// It holds no mutable state so it is safe for concurrent use.
type Service struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	issuer          string
}

// New constructs a JWT Service from the application JWT config.
func New(cfg config.JWTConfig) *Service {
	return &Service{
		secret:          []byte(cfg.Secret),
		accessTokenTTL:  cfg.AccessTokenTTL,
		refreshTokenTTL: cfg.RefreshTokenTTL,
		issuer:          cfg.Issuer,
	}
}

// AccessTokenTTL exposes the configured access token lifetime
// to callers that need to set cookie or cache expiry values.
func (s *Service) AccessTokenTTL() time.Duration { return s.accessTokenTTL }

// RefreshTokenTTL exposes the configured refresh token lifetime.
func (s *Service) RefreshTokenTTL() time.Duration { return s.refreshTokenTTL }

// IssueAccessToken creates and signs a short-lived JWT access
// token embedding the user's ID, role, and email as claims.
func (s *Service) IssueAccessToken(user *domain.User) (string, error) {
	const op = "jwt.Service.IssueAccessToken"

	now := time.Now().UTC()
	claims := accessClaims{
		UserID: user.ID.String(),
		Role:   user.Role,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return signed, nil
}

// ValidateAccessToken parses and validates a signed access token
// string, returning the extracted domain claims on success.
func (s *Service) ValidateAccessToken(raw string) (*domain.AccessTokenClaims, error) {
	const op = "jwt.Service.ValidateAccessToken"

	token, err := jwt.ParseWithClaims(
		raw,
		&accessClaims{},
		func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return s.secret, nil
		},
		jwt.WithIssuer(s.issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, s.classifyError(op, err)
	}

	claims, ok := token.Claims.(*accessClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("%s: malformed user id in claims: %w", op, ErrTokenInvalid)
	}

	return &domain.AccessTokenClaims{
		UserID: userID,
		Role:   claims.Role,
		Email:  claims.Email,
	}, nil
}

// classifyError maps jwt library errors onto our package-level
// sentinels so callers can use errors.Is cleanly.
func (s *Service) classifyError(op string, err error) error {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return fmt.Errorf("%s: %w", op, ErrTokenExpired)
	case errors.Is(err, jwt.ErrTokenMalformed):
		return fmt.Errorf("%s: %w", op, ErrTokenMalformed)
	default:
		return fmt.Errorf("%s: %w: %v", op, ErrTokenInvalid, err)
	}
}
