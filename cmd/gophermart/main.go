package main

import (
	"gophermart/cmd/internal/accrualservice"
	"gophermart/cmd/internal/config"
	"gophermart/cmd/internal/handlers"
	"gophermart/cmd/internal/logger"
	"gophermart/cmd/internal/models"
	"gophermart/cmd/internal/service"
	"gophermart/cmd/internal/storage"
	"gophermart/cmd/internal/workerpool"
	"net/http"

	"github.com/go-chi/chi"
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

	orderQueue := make(chan models.OrderQueue, config.RateLimit)

	service := service.NewService(&storage, log, accrual, orderQueue)

	handler := handlers.NewHandler(config, log, service)

	workerpool := workerpool.NewWorkerPool(orderQueue, service, accrual, log)
	for w := range config.RateLimit {
		go workerpool.Worker(w)
	}

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
