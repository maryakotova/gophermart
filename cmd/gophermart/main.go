package main

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/maryakotova/gophermart/cmd/internal/accrualservice"
	"github.com/maryakotova/gophermart/cmd/internal/config"
	"github.com/maryakotova/gophermart/cmd/internal/handlers"
	"github.com/maryakotova/gophermart/cmd/internal/logger"
	"github.com/maryakotova/gophermart/cmd/internal/service"
	"github.com/maryakotova/gophermart/cmd/internal/storage"
)

func main() {

	log, err := logger.Initialize("")
	if err != nil {
		panic(err)
	}

	config := config.NewConfig()

	factory := &storage.StorageFactory{}

	storage, err := factory.NewStorage(config, log)
	if err != nil {
		panic(err)
	}

	accrual, err := accrualservice.NewAccrualSystem(config, log)
	if err != nil {
		panic(err)
	}

	service := service.NewService(&storage, log, accrual)

	handler := handlers.NewHandler(config, log, service)

	router := chi.NewRouter()
	router.Use()

	router.Post("/api/user/register", logger.WithLogging(handler.Register))
	router.Post("/api/user/login", logger.WithLogging(handler.Login))
	router.Post("/api/user/orders", logger.WithLogging(handler.LoadOrder))
	router.Get("/api/user/orders", logger.WithLogging(handler.GetOrderList))
	router.Get("/api/user/balance", logger.WithLogging(handler.GetBalance))
	router.Post("/api/user/balance/withdraw", logger.WithLogging(handler.Withdraw))
	router.Get("/api/user/withdrawals", logger.WithLogging(handler.GetWithdraws))

	err = http.ListenAndServe(config.RunAddress, router)
	if err != nil {
		panic(err)
	}

}
