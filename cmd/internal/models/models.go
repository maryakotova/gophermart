package models

import "time"

type RegisterRequest struct {
	Login    string `json:"login"`    // Имя пользователя
	Password string `json:"password"` // Пароль
}

type OrderList struct {
	OrderNumber string
	Status      string
	Accrual     float64
	UploadedAt  time.Time
}

type OrderListResponce struct {
	OrderNumber string  `json:"number"`            // Номер заказа
	Status      string  `json:"status"`            // Статус заказа
	Accrural    float64 `json:"accrual,omitempty"` // Сумма начислений (опционально)
	UploadedAt  string  `json:"uploaded_at"`       // Время загрузки
}

type BalanceResponce struct {
	Balance   float64 `json:"current"`   // Текущая сумма баллов лояльности
	Withdrawn float64 `json:"withdrawn"` // Сумма использованных за весь период регистрации баллов
}

type WithdrawRequest struct {
	OrderNumber string  `json:"order"` // Номер заказа
	Sum         float64 `json:"sum"`   // Запрашиваемая сумма баллов для списания
}

type Withdrawals struct {
	OrderNumber string
	Sum         float64
	ProcessedAt time.Time
}

type WithdrawalsResponce struct {
	OrderNumber string  `json:"order"`        // Номер заказа
	Sum         float64 `json:"sum"`          // Списанное количество баллов
	ProcessedAt string  `json:"processed_at"` // Время вывода средств
}

type AccrualSystemResponce struct {
	Order   string  `json:"order"`             // Номер заказа
	Status  string  `json:"status"`            // Статус заказа
	Accrual float64 `json:"accrual,omitempty"` // Начисленные баллы
}
