package accrualservice

import (
	"encoding/json"
	"fmt"
	"gophermart/cmd/internal/config"
	"gophermart/cmd/internal/constants"
	"gophermart/cmd/internal/models"
	"net/http"
	"strconv"

	"go.uber.org/zap"
)

type AccrualService struct {
	config *config.Config
	logger *zap.Logger
}

func NewAccrualSystem(cfg *config.Config, logger *zap.Logger) (*AccrualService, error) {
	if cfg.AccrualSystemAddress == "" {
		err := fmt.Errorf("адрес системы расчёта начислений не заполнен")
		return nil, err
	}
	return &AccrualService{
		config: cfg,
		logger: logger,
	}, nil
}

func (a *AccrualService) GetAccrualFromService(orderNum int64) (response models.AccrualSystemResponce, retryAfter string, err error) {

	url := fmt.Sprintf("%s/api/orders/%s", a.config.AccrualSystemAddress, strconv.FormatInt(orderNum, 10))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		a.logger.Error(err.Error())
		return response, retryAfter, err
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		a.logger.Error(err.Error())
		return response, retryAfter, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&response)
		if err != nil {
			err = fmt.Errorf("ошибка при десериализации JSON: %w", err)
			a.logger.Error(err.Error())
			return response, retryAfter, err
		}

	case http.StatusNoContent:
		response = models.AccrualSystemResponce{Order: strconv.FormatInt(orderNum, 10), Status: constants.New}

	case http.StatusTooManyRequests:
		response = models.AccrualSystemResponce{Order: strconv.FormatInt(orderNum, 10), Status: constants.New}
		retryAfter = resp.Header.Get("Retry-After")

	case http.StatusInternalServerError:
		err = fmt.Errorf("ошибка при обращении к системе расчёта начислений баллов лояльности")
		a.logger.Error(err.Error())
		return

	default:
		err = fmt.Errorf("невозможно обработать ответ от системы расчёта начислений баллов лояльности (неизвестный статус)")
		a.logger.Error(err.Error())
		return
	}

	return response, retryAfter, err
}
