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

// AccrualStorage - интерфейс системы хранения для возможности запрос данных о начисленных бонусах
// AccrualStorage.UpdateOrder - Метод для обновления данных заказа по данным системы начисления бонусов
// GetOrdersWOaccrual - метод получения списка незавершенных заказов для запроса данных о начисленных бонусах
type AccrualStorage interface {
	UpdateOrder(order *model.DBOrder) status.Status
	GetOrdersWOaccrual(limit int) (*[]model.DBOrder, status.Status)
}

// Accrual - часть сервисного слоя, работающая с системой начисления бонусов
// Для создания использовать NewAccrual
type Accrual struct {
	log     *logger.Logger
	storage AccrualStorage
	url     string
}

// NewAccrual - создает новый Accrual
// Входящие параметры:
// logger *logger.Logger - ссылку на логгер
// storage AccrualStorage - реализацию интерфейса AccrualStorage
// URL string - Адрес сервиса начисления бонусов
func NewAccrual(logger *logger.Logger, storage AccrualStorage, URL string) (*Accrual, error) {
	llog := logger.With("accrual")
	return &Accrual{log: llog, storage: storage, url: URL}, nil
}

// GetaccrualData - метод получения данных от системы начисления бонусов
// Входящий параметр:
// order int - номер заказа
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

// RunCheckaccrualAsync - процесс получения информации от сервиса начисления бонусов по незавершенным в системе заказам предпологается запуск в отдельной горутине
// Входящие параметры:
// ctx context.Context - контекст выполнения
// limit int - Количество записей вычитываемых из БД для запроса начисленных бонусов
// interval time.Duration - Интервал вычитки из БД
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
