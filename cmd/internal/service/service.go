package service

import (
	"context"
	"fmt"
	"gophermart/cmd/internal/accrualservice"
	"gophermart/cmd/internal/constants"
	"gophermart/cmd/internal/customerrors"
	"gophermart/cmd/internal/models"
	"gophermart/cmd/internal/storage"
	"gophermart/cmd/internal/utils"
	"time"

	"go.uber.org/zap"
)

// // как правильно использовать интерфейсы? сейчас у мен яобъявлены 2 одинаковый
// type DataStorage interface {
// 	GetUserID(ctx context.Context, userName string) (userID int)
// 	CreateUser(ctx context.Context, login string, hashedPassword string) (userID int64, err error)
// }

type Service struct {
	storage    storage.Storage
	logger     *zap.Logger
	accrual    *accrualservice.AccrualService
	orderQueue chan models.OrderQueue
}

func NewService(storage *storage.Storage, logger *zap.Logger, accrual *accrualservice.AccrualService, orderQueue chan models.OrderQueue) *Service {
	return &Service{
		storage:    *storage,
		logger:     logger,
		accrual:    accrual,
		orderQueue: orderQueue,
	}
}

func (s *Service) CreateUser(ctx context.Context, login string, password string) (userID int, err error) {
	exists := s.checkUserExists(ctx, login)

	if exists {
		err = customerrors.ErrUsernameTaken
		s.logger.Error(err.Error())
		return
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		s.logger.Error(err.Error())
		return
	}

	userID, err = s.createUser(ctx, login, hashedPassword)
	if err != nil {
		s.logger.Error(err.Error())
		return
	}

	return
}

func (s *Service) CheckLoginData(ctx context.Context, login string, password string) (userID int, err error) {

	userID, dbPassword, err := s.storage.GetUserAuthData(ctx, login)
	if err != nil {
		s.logger.Error(err.Error())
		return -1, err
	}

	err = utils.СheckPassword(dbPassword, password)
	if err != nil {
		s.logger.Error(err.Error())
		return -1, err
	}

	return
}

func (s *Service) LoadOrderNumber(ctx context.Context, orderNumber int64, userID int) error {

	err := s.checkOrderLoaded(ctx, orderNumber, userID)
	if err != nil {
		s.logger.Error(err.Error())
		return err
	}

	accrualResponce, _, err := s.accrual.GetAccrualFromService(orderNumber)
	if err != nil {
		s.logger.Error(err.Error())
		return err
	}

	if accrualResponce.Status != constants.Invalid &&
		accrualResponce.Status != constants.Processed &&
		accrualResponce.Status != constants.NotRelevant {
		s.orderQueue <- models.OrderQueue{OrderNum: orderNumber, Status: accrualResponce.Status}
	}

	err = s.storage.InsertOrder(ctx, userID, accrualResponce)
	if err != nil {
		s.logger.Error(err.Error())
		return err
	}

	if accrualResponce.Status == constants.Processed && accrualResponce.Accrual > 0 {
		err = s.storage.IncreaseBalance(ctx, userID, accrualResponce.Accrual)
		if err != nil {
			s.logger.Error(err.Error())
			return err
		}
	}

	return nil
}

func (s *Service) GetOrders(ctx context.Context, userID int) (orders []models.OrderListResponce, err error) {

	bdOrders, err := s.storage.GetOrdersForUser(ctx, userID)
	if err != nil {
		s.logger.Error(err.Error())
		return orders, err
	}

	for _, order := range bdOrders {
		if order.Status == constants.NotRelevant {
			continue
		}

		orders = append(orders, models.OrderListResponce{
			OrderNumber: order.OrderNumber,
			Status:      order.Status,
			Accrural:    order.Accrual,
			UploadedAt:  order.UploadedAt.Format(time.RFC3339),
		},
		)
	}

	return orders, nil
}

func (s *Service) GetBalance(ctx context.Context, userID int) (balance models.BalanceResponce, err error) {

	currentBalance, err := s.storage.GetCurrentBalance(ctx, userID)
	if err != nil {
		currentBalance = 0
	}

	WithdrawalSum, err := s.storage.GetWithdrawalSum(ctx, userID)
	if err != nil {
		WithdrawalSum = 0
	}

	balance.Balance = currentBalance
	balance.Withdrawn = WithdrawalSum

	return balance, nil
}

func (s *Service) WithdrawalRequest(ctx context.Context, userID int, orderNumber int64, sum float64) (err error) {

	currentBalance, err := s.storage.GetCurrentBalance(ctx, userID)
	if err != nil {
		currentBalance = 0
	}

	if currentBalance < sum {
		err = customerrors.ErrLowBalance
		s.logger.Error(err.Error())
		return
	}

	newBalance := currentBalance - sum

	err = s.storage.UpdateBalance(ctx, userID, newBalance)
	if err != nil {
		s.logger.Error(err.Error())
		return err
	}

	err = s.storage.InsertWithdrawal(ctx, userID, orderNumber, sum)
	if err != nil {
		s.logger.Error(err.Error())
		return err
	}

	return nil
}

func (s *Service) GetWithdraws(ctx context.Context, userID int) (withdrawals []models.WithdrawalsResponce, err error) {

	bdWithdrawals, err := s.storage.GetWithdrawalsForUser(ctx, userID)
	if err != nil {
		s.logger.Error(err.Error())
		return withdrawals, err
	}

	for _, withdrawal := range bdWithdrawals {
		withdrawals = append(withdrawals, models.WithdrawalsResponce{
			OrderNumber: withdrawal.OrderNumber,
			Sum:         withdrawal.Sum,
			ProcessedAt: withdrawal.ProcessedAt.Format(time.RFC3339),
		})
	}

	return withdrawals, nil
}

func (s *Service) checkUserExists(ctx context.Context, login string) (exists bool) {

	userID, _ := s.storage.GetUserID(ctx, login)
	return userID != -1

}

func (s *Service) createUser(ctx context.Context, login string, hashedPassword string) (userID int, err error) {
	userID, err = s.storage.CreateUser(ctx, login, hashedPassword)
	if userID == 0 {
		err = fmt.Errorf("ошибка при создании пользователя")
	}
	return
}

func (s *Service) checkOrderLoaded(ctx context.Context, orderNumber int64, userID int) (err error) {
	dbUserID, _ := s.storage.GetUserByOrderNum(ctx, orderNumber)
	// if err != nil {
	// 	return err
	// }

	if dbUserID != -1 {
		if dbUserID == userID {
			err = customerrors.ErrOrderLoadedByUser
		} else {
			err = customerrors.ErrOrderLoadedByAnotherUser
		}
		return err
	}

	return nil
}

func (s *Service) UpdateOrder(ctx context.Context, accrualResponce models.AccrualSystemResponce) error {
	err := s.storage.UpdateOrder(ctx, accrualResponce)
	if err != nil {
		s.logger.Error(err.Error())
		return err
	}
	return nil
}
