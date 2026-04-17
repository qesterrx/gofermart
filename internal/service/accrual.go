package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/qesterrx/gofermart/internal/logger"
	"github.com/qesterrx/gofermart/internal/model"
	"github.com/qesterrx/gofermart/internal/status"
)

type AccrualStorage interface {
	UpdateOrder(order *model.DBOrder) status.Status
	GetOrdersWOaccrual(limit int) (*[]model.DBOrder, status.Status)
}

type Accrual struct {
	log     *logger.Logger
	storage AccrualStorage
	url     string
}

func NewAccrual(logger *logger.Logger, storage AccrualStorage, URL string) (*Accrual, error) {
	llog := logger.With("accrual")
	return &Accrual{log: llog, storage: storage, url: URL}, nil
}

func (a *Accrual) GetaccrualData(order int) (*model.Accrual, error) {
	uri := "http://" + a.url + "/api/orders/" + strconv.Itoa(order)

	resp, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Читаем тело ответа в буфер
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Показываем исходный body
	a.log.Debug("GetaccrualData: responce.body от accrual:" + string(bodyBytes))

	accrual := model.Accrual{}
	err = json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&accrual)
	if err != nil {
		return nil, err
	}

	return &accrual, nil
}

func (a *Accrual) RunCheckaccrualAsync(ctx context.Context, limit int, interval time.Duration) {

	a.log.Info("Запущен RunCheckaccrualAsync")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.log.Debug("Заверешени RunCheckaccrualAsync по контексту")
			return
		case <-ticker.C:
			a.log.Debug("Ticker RunCheckaccrualAsync")
			orders, st := a.storage.GetOrdersWOaccrual(limit)
			if st != status.StOk {
				a.log.Error("RunCheckaccrualAsync: Ошибка в GetOrdersWOaccrual")
				continue
			}

			for _, ord := range *orders {
				acc, err := a.GetaccrualData(ord.Order)
				if err != nil {
					a.log.Error("RunCheckaccrualAsync: Ошибка в GetaccrualData " + err.Error())
					continue
				}

				if acc == nil {
					continue
				}

				order := model.DBOrder{Order: ord.Order}

				if acc.Sum != nil {
					tmp := int(math.Round(float64(*acc.Sum * 100)))
					order.Accrual = &tmp
				}

				switch acc.Status {
				case model.AccrualStInvalid:
					order.Status = model.OrderStInvalid
				case model.AccrualStProcessed:
					order.Status = model.OrderStProcessed
				default:
					order.Status = model.OrderStProcessing
				}

				st := a.storage.UpdateOrder(&order)
				if st != status.StOk {
					a.log.Error("RunCheckaccrualAsync: Ошибка в SetaccrualStatus")
					continue
				}
			}

		}
	}
}
