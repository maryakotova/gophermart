package authutils

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

const TokenExp = time.Hour * 24
const SecretKey = "SecretKeyForGophermart"

func SetAuthCookie(w http.ResponseWriter, userID int) error {

	w.Header().Set("Authorization", strconv.Itoa(userID))

	// expiresAt := time.Now().Add(TokenExp)
	// tokenString, err := buildJWTString(userID, expiresAt)
	// if err != nil {
	// 	slog.Error(fmt.Sprintf("ошибка при генерации токена: %s", err))
	// 	return err
	// }

	// http.SetCookie(w, &http.Cookie{
	// 	Name:     "auth_token",
	// 	Value:    tokenString,
	// 	Expires:  expiresAt,
	// 	HttpOnly: true,
	// })

	// slog.Info(fmt.Sprintf("Токен сгенерирован: %s", tokenString))

	return nil
}

// func buildJWTString(userID int, expiresAt time.Time) (string, error) {

// 	slog.Info(fmt.Sprintf("данные для создания токена: %v, %v", userID, expiresAt))

// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
// 		RegisteredClaims: jwt.RegisteredClaims{
// 			Issuer:    "Gophermart",
// 			ExpiresAt: jwt.NewNumericDate(expiresAt),
// 			IssuedAt:  jwt.NewNumericDate(time.Now()),
// 			Subject:   fmt.Sprintf("%d", userID),
// 		},
// 		UserID: userID,
// 	})

// 	if token == nil {
// 		slog.Info("объект token не создан")
// 	}

// 	tokenString, err := token.SignedString([]byte(SecretKey))
// 	if err != nil {
// 		slog.Error(fmt.Sprintf("ошибка при подписании токеном: %s", err))
// 		return "", err
// 	}
// 	slog.Info(fmt.Sprintf("token string: %s", tokenString))

// 	return tokenString, nil
// }

// func getUserID(tokenString string) int {

// 	claims := &Claims{}

// 	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
// 		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
// 			slog.Error(fmt.Sprintf("unexpected signing method: %v", t.Header["alg"]))
// 			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
// 		}

// 		return []byte(SecretKey), nil
// 	})

// 	if err != nil {
// 		slog.Error(fmt.Sprintf("ошибка при чтении токена: %s", err))
// 		return -1
// 	}

// 	if !token.Valid {
// 		slog.Error("токен не вальдный")
// 		return -1
// 	}

// 	return claims.UserID
// }

func ReadAuthCookie(r *http.Request) (userID int, err error) {

	userIDstring := r.Header.Get("Authorization")

	slog.Error(fmt.Sprintf("UserID in request: %s", userIDstring))

	userID, err = strconv.Atoi(userIDstring)

	// cookie, err := r.Cookie("auth_token")
	// if err != nil {
	// 	slog.Error(fmt.Sprintf("ошибка чтении токена из request: %s", err))
	// 	return -1, err
	// }

	// slog.Info(fmt.Sprintf("значение токена в request: %s", cookie.Value))

	// userID = getUserID(cookie.Value)

	// if userID == -1 {
	// 	return userID, err
	// }

	return userID, err

	// return 2, nil
}
