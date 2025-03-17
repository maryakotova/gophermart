package storage

import (
	"context"

	"github.com/maryakotova/gophermart/internal/config"
	"github.com/maryakotova/gophermart/internal/models"
	"github.com/maryakotova/gophermart/internal/storage/postgres"
	"go.uber.org/zap"
)

type Storage interface {
	GetUserID(ctx context.Context, userName string) (userID int, err error)
	CreateUser(ctx context.Context, login string, hashedPassword string) (userID int, err error)
	GetUserAuthData(ctx context.Context, login string) (userID int, hashedPassword string, err error)
	GetUserByOrderNum(ctx context.Context, orderNumber int64) (userID int, err error)
	InsertOrder(ctx context.Context, userID int, accrualResponce models.AccrualSystemResponce) error
	GetOrdersForUser(ctx context.Context, userID int) (orders []models.OrderList, err error)
	UpdateBalance(ctx context.Context, userID int, points float64) error
	GetCurrentBalance(ctx context.Context, userID int) (balance float64, err error)
	GetWithdrawalSum(ctx context.Context, userID int) (withdrawalSum float64, err error)
	IncreaseBalance(ctx context.Context, userID int, points float64) error
	InsertWithdrawal(ctx context.Context, userID int, orderNumber int64, points float64) error
	GetWithdrawalsForUser(ctx context.Context, userID int) (withdrawals []models.Withdrawals, err error)
}

type StorageFactory struct{}

func (f *StorageFactory) NewStorage(cfg *config.Config, logger *zap.Logger) (Storage, error) {
	postgres, err := postgres.NewPostgresStorage(cfg, logger)
	if err != nil {
		return nil, err
	}
	err = postgres.Bootstrap(context.TODO())
	return postgres, err
}
