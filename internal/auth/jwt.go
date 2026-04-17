package auth

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var JWTSecretKey []byte = []byte("hello")
var JWTExpire time.Duration = 60 * time.Minute
var CookieName string = "access_token"

type JWTC struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(userId int, username string) (string, error) {

	jwtс := JWTC{
		UserID:   userId,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(JWTExpire)),
			Issuer:    "gofermart",
			Subject:   strconv.Itoa(userId),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtс)
	return token.SignedString(JWTSecretKey)

}

func ValidateToken(tokenString string) (*JWTC, error) {

	fn := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("нераспознанный метод подписи JWT")
		}
		return JWTSecretKey, nil
	}

	tkn, err := jwt.ParseWithClaims(tokenString, &JWTC{}, fn)
	if err != nil {
		return nil, err
	}

	if jwtc, ok := tkn.Claims.(*JWTC); ok && tkn.Valid {
		return jwtc, nil
	}

	return nil, fmt.Errorf("некорректный JWT токен")
}
