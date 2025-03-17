package utils

import (
	"fmt"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

func CheckOrderNumber(orderNumberS string) (orderNumberI int64, err error) {

	// проверить что не пустой
	if orderNumberS == "" {
		err = fmt.Errorf("номер заказа должен быть заполнен")
		return 0, err
	}

	// проверить что тип int
	orderNumberI, err = strconv.ParseInt(orderNumberS, 10, 64)
	if err != nil {
		err = fmt.Errorf("номер заказа может содержать только цифры")
		return 0, err
	}

	//проверить через Luhn
	if !isValidLuhn(orderNumberS) {
		err = fmt.Errorf("номер заказа некорректен, проверьте правильность ввода номера")
		return 0, err
	}

	return orderNumberI, nil
}

func isValidLuhn(number string) bool {
	// Проходим по всем цифрам строки с конца
	sum := 0
	shouldDouble := false
	for i := len(number) - 1; i >= 0; i-- {
		// Получаем цифру по индексу
		digit, err := strconv.Atoi(string(number[i]))
		if err != nil {
			// Если символ не является цифрой, возвращаем false
			return false
		}

		// Если нужно удвоить цифру
		if shouldDouble {
			digit *= 2
			if digit > 9 {
				digit -= 9 // Если результат больше 9, вычитаем 9
			}
		}

		// Добавляем цифру в сумму
		sum += digit

		// Переключаем флаг для удвоения цифры на следующем шаге
		shouldDouble = !shouldDouble
	}

	// Если сумма кратна 10, то номер корректен
	return sum%10 == 0
}

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}
