package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var jwtKey = []byte("supersecretkey")

type JWTClaim struct {
	Firstname string `json:"first_name"`
	Lastname  string `json:"last_name"`
	Email     string `json:"email"`
	Admin     bool   `json:"admin"`
	Verified  bool   `json:"verified"`
	UserID    int    `json:"id"`
	jwt.StandardClaims
}

func SetPrivateKey(PrivateKey string) error {
	if len(PrivateKey) < 16 {
		return errors.New("Private key must be atleast 16 characters.")
	}

	jwtKey = []byte(PrivateKey)
	return nil
}

func GenerateJWT(firstname string, lastname string, email string, userid int, admin bool, verified bool) (tokenString string, err error) {
	expirationTime := time.Now().Add(1 * time.Hour * 24 * 7)
	claims := &JWTClaim{
		Firstname: firstname,
		Lastname:  lastname,
		Email:     email,
		Admin:     admin,
		UserID:    userid,
		Verified:  verified,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err = token.SignedString(jwtKey)
	return
}

func ValidateToken(signedToken string, admin bool) (err error) {
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
	}
	if claims.ExpiresAt < time.Now().Local().Unix() {
		err = errors.New("Token expired.")
		return
	}
	if admin && !claims.Admin {
		err = errors.New("Token not an admin session.")
		return
	}
	return
}

func ParseToken(signedToken string) (*JWTClaim, error) {
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
	if claims.ExpiresAt < time.Now().Local().Unix() {
		err = errors.New("token expired")
		return nil, err
	}
	return claims, nil
}
