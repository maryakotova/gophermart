package accrualservice

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/maryakotova/gophermart/cmd/internal/config"
	"github.com/maryakotova/gophermart/cmd/internal/constants"
	"github.com/maryakotova/gophermart/cmd/internal/models"
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

func (a *AccrualService) GetAccrualFromService(orderNum int64) (response models.AccrualSystemResponce, err error) {

	url := fmt.Sprintf("http://%s/api/orders/%s", a.config.AccrualSystemAddress, strconv.FormatInt(orderNum, 10))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return response, err
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return response, err
	}

	switch resp.StatusCode {
	case http.StatusOK:
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&response)
		if err != nil {
			err = fmt.Errorf("ошибка при десериализации JSON: %w", err)
			return response, err
		}

	case http.StatusNoContent:
		response = models.AccrualSystemResponce{Order: strconv.FormatInt(orderNum, 10), Status: constants.NotRelevant}

	case http.StatusTooManyRequests:
		response = models.AccrualSystemResponce{Order: strconv.FormatInt(orderNum, 10), Status: constants.New}

	case http.StatusInternalServerError:
		err = fmt.Errorf("ошибка при обращении к системе расчёта начислений баллов лояльности")
		return

	default:
		err = fmt.Errorf("невозможно обработать ответ от системы расчёта начислений баллов лояльности (неизвестный статус)")
		return
	}

	if response.Status != constants.Invalid && response.Status != constants.Processed && response.Status != constants.NotRelevant {
		//добавить в поток на обработку воркерами
	}

	return response, err
}
