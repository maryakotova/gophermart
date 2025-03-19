package authutils

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

const TokenExp = time.Hour * 3
const SecretKey = "SecretKeyForGophermart"

func SetAuthCookie(w http.ResponseWriter, userID int) error {

	expiresAt := time.Now().Add(TokenExp)
	tokenString, err := buildJWTString(userID, expiresAt)
	if err != nil {
		slog.Error(fmt.Sprintf("ошибка при генерации токена: %s", err))
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Expires:  expiresAt,
		HttpOnly: true,
	})

	slog.Info(fmt.Sprintf("Токен сгенерирован: %s", tokenString))

	return nil
}

func buildJWTString(userID int, expiresAt time.Time) (string, error) {

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		UserID: userID,
	})

	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		slog.Error(fmt.Sprintf("ошибка при подписании токеном: %s", err))
		return "", err
	}

	return tokenString, nil
}

func getUserID(tokenString string) int {

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			slog.Error(fmt.Sprintf("unexpected signing method: %v", t.Header["alg"]))
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return []byte(SecretKey), nil
	})

	if err != nil {
		slog.Error(fmt.Sprintf("ошибка при чтении токена: %s", err))
		return -1
	}

	if !token.Valid {
		slog.Error("токен не вальдный")
		return -1
	}

	return claims.UserID
}

func ReadAuthCookie(r *http.Request) (userID int, err error) {

	cookie, err := r.Cookie("auth_token")
	if err != nil {
		slog.Error(fmt.Sprintf("ошибка чтении токена из request: %s", err))
		return -1, err
	}

	slog.Info(fmt.Sprintf("значение токена в request: %s", cookie.Value))

	userID = getUserID(cookie.Value)

	if userID == -1 {
		return userID, err
	}

	return userID, nil

	// return 2, nil
}
