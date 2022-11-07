package middlewares

import (
	"errors"
	"poenskelisten/auth"

	"github.com/gin-gonic/gin"
)

func Auth(admin bool) gin.HandlerFunc {
	return func(context *gin.Context) {
		tokenString := context.GetHeader("Authorization")
		if tokenString == "" {
			context.JSON(401, gin.H{"error": "request does not contain an access token"})
			context.Abort()
			return
		}
		err := auth.ValidateToken(tokenString, admin)
		if err != nil {
			context.JSON(401, gin.H{"error": err.Error()})
			context.Abort()
			return
		}
		context.Next()
	}
}

func GetAuthUsername(tokenString string) (int, error) {

	if tokenString == "" {
		return 0, errors.New("No Auhtorization header given.")
	}
	claims, err := auth.ParseToken(tokenString)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}
