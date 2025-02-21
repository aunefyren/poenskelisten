package auth

import (
	"aunefyren/poenskelisten/config"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTClaim struct {
	Firstname string    `json:"first_name"`
	Lastname  string    `json:"last_name"`
	Email     string    `json:"email"`
	Admin     bool      `json:"admin"`
	Verified  bool      `json:"verified"`
	UserID    uuid.UUID `json:"id"`
	jwt.RegisteredClaims
}

func GenerateJWT(firstname string, lastname string, email string, userid uuid.UUID, admin bool, verified bool) (tokenString string, err error) {
	expirationTime := time.Now().Add(1 * time.Hour * 24 * 7)
	claims := &JWTClaim{
		Firstname: firstname,
		Lastname:  lastname,
		Email:     email,
		Admin:     admin,
		UserID:    userid,
		Verified:  verified,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "Pønskelisten",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtKey := config.GetPrivateKey(1)
	tokenString, err = token.SignedString(jwtKey)
	return
}

func GenerateJWTFromClaims(claims *JWTClaim) (tokenString string, err error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtKey := config.GetPrivateKey(1)
	tokenString, err = token.SignedString(jwtKey)
	return
}

func ValidateToken(signedToken string, admin bool) (err error) {
	jwtKey := config.GetPrivateKey(1)
	token, err := jwt.ParseWithClaims(
		signedToken,
		&JWTClaim{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtKey), nil
		},
	)
	if err != nil {
		return
	}
	claims, ok := token.Claims.(*JWTClaim)
	if !ok {
		err = errors.New("Couldn't parse claims.")
		return
	} else if claims.ExpiresAt == nil || claims.NotBefore == nil {
		err = errors.New("Claims not present.")
		return
	}
	now := time.Now()
	if claims.ExpiresAt.Time.Before(now) {
		err = errors.New("Token has expired.")
		return
	}
	if claims.NotBefore.Time.After(now) {
		err = errors.New("Token has not begun.")
		return
	}
	if admin && !claims.Admin {
		err = errors.New("Token not an admin session.")
		return
	}
	return
}

func ParseToken(signedToken string) (*JWTClaim, error) {
	jwtKey := config.GetPrivateKey(1)
	token, err := jwt.ParseWithClaims(
		signedToken,
		&JWTClaim{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtKey), nil
		},
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaim)
	if !ok {
		err = errors.New("couldn't parse claims")
		return nil, err
	}
	return claims, nil
}
