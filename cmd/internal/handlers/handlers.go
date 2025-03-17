package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/maryakotova/gophermart/cmd/internal/authutils"
	"github.com/maryakotova/gophermart/cmd/internal/config"
	"github.com/maryakotova/gophermart/cmd/internal/customerrors"
	"github.com/maryakotova/gophermart/cmd/internal/models"
	"github.com/maryakotova/gophermart/cmd/internal/service"
	"github.com/maryakotova/gophermart/cmd/internal/utils"
	"go.uber.org/zap"
)

type Handler struct {
	config  *config.Config
	logger  *zap.Logger
	service *service.Service
}

func NewHandler(cfg *config.Config, logger *zap.Logger, service *service.Service) *Handler {
	return &Handler{
		config:  cfg,
		logger:  logger,
		service: service,
	}
}

func (handler *Handler) Register(res http.ResponseWriter, req *http.Request) {

	var request models.RegisterRequest

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&request)
	if err != nil || request.Login == "" || request.Password == "" {
		err = fmt.Errorf("ошибка при десериализации JSON: %w", err)
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if request.Login == "" || request.Password == "" {
		err = fmt.Errorf("логин и пароль должны быть заполнены")
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	userID, err := handler.service.CreateUser(req.Context(), request.Login, request.Password)
	if err != nil {
		if errors.Is(err, customerrors.ErrUsernameTaken) {
			http.Error(res, err.Error(), http.StatusConflict)
		} else {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if userID == -1 {
		err = fmt.Errorf("неизвестная ошибка при создании пользователя")
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = authutils.SetAuthCookie(res, userID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)

}

func (handler *Handler) Login(res http.ResponseWriter, req *http.Request) {

	var request models.RegisterRequest

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&request)
	if err != nil || request.Login == "" || request.Password == "" {
		err = fmt.Errorf("ошибка при десериализации JSON: %w", err)
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	if request.Login == "" || request.Password == "" {
		err = fmt.Errorf("логин и пароль должны быть заполнены")
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	userID, err := handler.service.CheckLoginData(req.Context(), request.Login, request.Password)
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}

	if userID == -1 {
		err = fmt.Errorf("неизвестная ошибка при регистрации пользователя")
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	err = authutils.SetAuthCookie(res, userID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)

}

func (handler *Handler) LoadOrder(res http.ResponseWriter, req *http.Request) {
	userID, err := authutils.ReadAuthCookie(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}

	orderNum, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "ошибка при чтении тела запроса", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	orderNumber, err := utils.CheckOrderNumber(string(orderNum))
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	err = handler.service.LoadOrderNumber(req.Context(), orderNumber, userID)
	if err != nil {
		if errors.Is(err, customerrors.ErrOrderLoadedByAnotherUser) {
			http.Error(res, err.Error(), http.StatusConflict)
		} else if errors.Is(err, customerrors.ErrOrderLoadedByUser) {
			res.WriteHeader(http.StatusOK)
		} else {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusAccepted)

}

func (handler *Handler) GetOrderList(res http.ResponseWriter, req *http.Request) {

	userID, err := authutils.ReadAuthCookie(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}

	orders, err := handler.service.GetOrders(req.Context(), userID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	enc := json.NewEncoder(res)
	if err := enc.Encode(orders); err != nil {
		err = fmt.Errorf("ошибка при заполнении ответа: %w", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

}

func (handler *Handler) GetBalance(res http.ResponseWriter, req *http.Request) {

	userID, err := authutils.ReadAuthCookie(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}

	balance, err := handler.service.GetBalance(req.Context(), userID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	enc := json.NewEncoder(res)
	if err := enc.Encode(balance); err != nil {
		err = fmt.Errorf("ошибка при заполнении ответа: %w", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

}

func (handler *Handler) Withdraw(res http.ResponseWriter, req *http.Request) {
	userID, err := authutils.ReadAuthCookie(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}

	var request models.WithdrawRequest
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&request); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError) //должен быть http.StatusBadRequest
		return
	}

	orderNumber, err := utils.CheckOrderNumber(request.OrderNumber)
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	err = handler.service.WithdrawalRequest(req.Context(), userID, orderNumber, request.Sum)
	if err != nil {
		if errors.Is(err, customerrors.ErrLowBalance) {
			http.Error(res, err.Error(), http.StatusPaymentRequired)
		} else {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusOK)
}

func (handler *Handler) GetWithdraws(res http.ResponseWriter, req *http.Request) {

	userID, err := authutils.ReadAuthCookie(req)
	if err != nil {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}

	withdraws, err := handler.service.GetWithdraws(req.Context(), userID)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(withdraws) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	enc := json.NewEncoder(res)
	if err := enc.Encode(withdraws); err != nil {
		err = fmt.Errorf("ошибка при заполнении ответа: %w", err)
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

}
