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
	storage storage.Storage
	logger  *zap.Logger
	accrual *accrualservice.AccrualService
}

func NewService(storage *storage.Storage, logger *zap.Logger, accrual *accrualservice.AccrualService) *Service {
	return &Service{
		storage: *storage,
		logger:  logger,
		accrual: accrual,
	}
}

func (s *Service) CreateUser(ctx context.Context, login string, password string) (userID int, err error) {
	exists, err := s.checkUserExists(ctx, login)
	if err != nil {
		s.logger.Error(err.Error())
		return
	}

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

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		s.logger.Error(err.Error())
		return
	}

	userID, dbPassword, err := s.storage.GetUserAuthData(ctx, login)
	if err != nil {
		s.logger.Error(err.Error())
		return
	}

	if dbPassword != hashedPassword || userID == 0 {
		err = fmt.Errorf("неверная пара логин/пароль")
		s.logger.Error(err.Error())
		return
	}

	return
}

func (s *Service) LoadOrderNumber(ctx context.Context, orderNumber int64, userID int) error {

	err := s.checkOrderLoaded(ctx, orderNumber, userID)
	if err != nil {
		s.logger.Error(err.Error())
		return err
	}

	accrualResponce, err := s.accrual.GetAccrualFromService(orderNumber)
	if err != nil {
		s.logger.Error(err.Error())
		return err
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
		s.logger.Error(err.Error())
		return
	}

	WithdrawalSum, err := s.storage.GetWithdrawalSum(ctx, userID)
	if err != nil {
		s.logger.Error(err.Error())
		return
	}

	balance.Balance = currentBalance
	balance.Withdrawn = WithdrawalSum

	return balance, nil
}

func (s *Service) WithdrawalRequest(ctx context.Context, userID int, orderNumber int64, sum float64) (err error) {

	currentBalance, err := s.storage.GetCurrentBalance(ctx, userID)
	if err != nil {
		s.logger.Error(err.Error())
		return
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

func (s *Service) checkUserExists(ctx context.Context, login string) (exists bool, err error) {

	userID, err := s.storage.GetUserID(ctx, login)
	return userID != -1, err

}

func (s *Service) createUser(ctx context.Context, login string, hashedPassword string) (userID int, err error) {
	userID, err = s.storage.CreateUser(ctx, login, hashedPassword)
	if userID == 0 {
		err = fmt.Errorf("ошибка при создании пользователя")
	}
	return
}

func (s *Service) checkOrderLoaded(ctx context.Context, orderNumber int64, userID int) (err error) {
	dbUserID, err := s.storage.GetUserByOrderNum(ctx, orderNumber)
	if err != nil {
		return err
	}

	if dbUserID != 0 {
		if dbUserID == userID {
			err = customerrors.ErrOrderLoadedByUser
		} else {
			err = customerrors.ErrOrderLoadedByAnotherUser
		}
		return err
	}

	return nil
}
