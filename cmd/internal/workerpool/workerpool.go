package workerpool

import (
	"context"
	"fmt"
	"gophermart/cmd/internal/accrualservice"
	"gophermart/cmd/internal/models"
	"gophermart/cmd/internal/service"
	"strconv"
	"time"

	"go.uber.org/zap"
)

type WorkerPool struct {
	orderQueue     chan models.OrderQueue
	service        *service.Service
	accrualservice *accrualservice.AccrualService
	logger         *zap.Logger
}

func NewWorkerPool(orderQueue chan models.OrderQueue, service *service.Service, accrualservice *accrualservice.AccrualService, logger *zap.Logger) *WorkerPool {
	return &WorkerPool{
		orderQueue:     orderQueue,
		service:        service,
		accrualservice: accrualservice,
		logger:         logger,
	}
}

func (wp *WorkerPool) Worker(workerNum int) {
	wp.logger.Info(fmt.Sprintf("worker %v started", workerNum))
	for {
		order := <-wp.orderQueue
		response, retryAfter, err := wp.accrualservice.GetAccrualFromService(order.OrderNum)
		if err != nil {
			wp.orderQueue <- order
			err = fmt.Errorf("worker %v: error from the accrual system for the order %v: %w", workerNum, order.OrderNum, err)
			wp.logger.Error(err.Error())
			continue
		}

		if retryAfter != "" {
			seconds, err := strconv.Atoi(retryAfter)
			if err != nil {
				seconds = 60
			}
			time.Sleep(time.Duration(seconds) * time.Second)
		}

		if order.Status == response.Status {
			wp.orderQueue <- order
			wp.logger.Info(fmt.Sprintf("worker %v: status of order %v has not changed", workerNum, order.OrderNum))
			continue
		}

		err = wp.service.UpdateOrder(context.TODO(), response)
		if err != nil {
			wp.orderQueue <- order
			err = fmt.Errorf("worker %v: error during the update %v: %w", workerNum, order.OrderNum, err)
			wp.logger.Error(err.Error())
			continue
		}

	}
}
