package auth

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTSecretKey - Ключ для кодирования JWT токена
var JWTSecretKey []byte = []byte("xK9mP2nQ5rT8vW3yZ7bA1cD4eF6gH9jK2lM5oP8sU1vX4")

// JWTExpire - Срок действия токена
var JWTExpire time.Duration = 60 * time.Minute

// JWTCookieName Наименование cookie с JWT токеном
var JWTCookieName string = "access_token"

// JWTC - структура для данных передаваемых в JWT токене
type JWTC struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateAccessToken - функция геренации JWT токена авторизации
// На вход принимает ИД пользователя и логин
// На выходе отдает JWT токен или ошибку
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

// ValidateToken функция проверки и расшифровки JWT токена
// На вход принимает строку из cookie
// На выходе отдает ссылку на экземпляр JWTC или ошибку
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
