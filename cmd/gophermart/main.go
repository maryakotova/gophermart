package main

// Привет! Реализовала механизм сжатия, как в metrics. Может быть нужно по-другому?
// И не сделала retry для ошибок при подключении к БД, тоже доделаю, но хотела бы сначала обсудить
// Есть вопрос по правильному обновлению нескольких таблиц БД. Если по логике приложения должны быть обновлены несколько таблиц
// (например, order и balance) Нужно ли обновлять их через транзакцию?
// Ты обещал прислать пример использования транзакций, то так и не прислал, жду
// При возможности прошу прислать не только замечания, но и варианты, что можно улучшить и с чем еще потренироваться
// Заранее спасибо :)

import (
	"gophermart/cmd/internal/accrualservice"
	"gophermart/cmd/internal/config"
	"gophermart/cmd/internal/handlers"
	"gophermart/cmd/internal/logger"
	"gophermart/cmd/internal/middleware"
	"gophermart/cmd/internal/models"
	"gophermart/cmd/internal/service"
	"gophermart/cmd/internal/storage"
	"gophermart/cmd/internal/workerpool"
	"net/http"
	"strings"

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

func gzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ow := w

		supportsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
		supportsGzipJSON := strings.Contains(r.Header.Get("Accept"), "application/json")
		supportsGzipHTML := strings.Contains(r.Header.Get("Accept"), "text/html")
		if supportsGzip && (supportsGzipJSON || supportsGzipHTML) {
			cw := middleware.NewCompressWriter(w)
			ow = cw
			defer cw.Close()
			ow.Header().Set("Content-Encoding", "gzip")
		}

		sendsGzip := strings.Contains(r.Header.Get("Content-Encoding"), "gzip")
		if sendsGzip {
			cr, err := middleware.NewCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		h.ServeHTTP(ow, r)
	}
}
