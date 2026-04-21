package service

import (
	"io"
	"strconv"
	"testing"
	"time"

	"github.com/qesterrx/gofermart/internal/logger"
	"github.com/qesterrx/gofermart/internal/model"
	"github.com/qesterrx/gofermart/internal/status"
	"github.com/qesterrx/gofermart/mocks"
	"github.com/stretchr/testify/assert"
)

func TestCheckOrderNumber(t *testing.T) {
	llog := logger.NewLogger("debug", io.Discard)
	gm, err := NewGofermart(llog, nil)
	assert.NoError(t, err)

	err = gm.CheckOrderNumber("")
	assert.EqualError(t, err, "пустой order")

	err = gm.CheckOrderNumber("159qwe")
	assert.EqualError(t, err, "order: ошибка формата")

	err = gm.CheckOrderNumber("5482145236587445")
	assert.EqualError(t, err, "order: ошибка контрольной суммы")

	err = gm.CheckOrderNumber("548214523658744")
	assert.NoError(t, err)
}

func TestNewOrder(t *testing.T) {
	//Подготовка
	mockStorage := new(mocks.GofermartStorage)
	llog := logger.NewLogger("debug", io.Discard)

	gm, err := NewGofermart(llog, mockStorage)
	assert.NoError(t, err)

	user1 := 2001
	order1 := 1001
	order2 := 1002

	user2 := 2002
	order3 := 1003
	order4 := 1004

	//Ожидания моков
	mockStorage.On("CheckOrderExist", order1, user1).Return(status.StOrderNotExists)
	mockStorage.On("CheckOrderExist", order2, user1).Return(status.StOrderDuplicated)
	mockStorage.On("CheckOrderExist", order2, user2).Return(status.StOrderAnotherUser)

	mockStorage.On("CheckOrderExist", order3, user2).Return(status.StOrderNotExists)
	mockStorage.On("CheckOrderExist", order4, user2).Return(status.StOrderDuplicated)
	mockStorage.On("CheckOrderExist", order4, user1).Return(status.StOrderAnotherUser)

	//нюанс - тут надо по-честному повторить логику ограничений на БД
	mockStorage.On("NewOrder", &model.DBOrder{Order: order1, User: user1, Status: model.OrderStNew}).Return(status.StOk)
	mockStorage.On("NewOrder", &model.DBOrder{Order: order2, User: user1, Status: model.OrderStNew}).Return(status.StGeneralError)
	mockStorage.On("NewOrder", &model.DBOrder{Order: order2, User: user2, Status: model.OrderStNew}).Return(status.StGeneralError)

	mockStorage.On("NewOrder", &model.DBOrder{Order: order3, User: user2, Status: model.OrderStNew}).Return(status.StOk)
	mockStorage.On("NewOrder", &model.DBOrder{Order: order4, User: user2, Status: model.OrderStNew}).Return(status.StGeneralError)
	mockStorage.On("NewOrder", &model.DBOrder{Order: order4, User: user1, Status: model.OrderStNew}).Return(status.StGeneralError)

	//Тесты
	st := gm.NewOrder(1, "123qwe")
	assert.Equal(t, status.StGeneralError, st)

	st = gm.NewOrder(user1, strconv.Itoa(order1))
	assert.Equal(t, status.StOk, st)
	st = gm.NewOrder(user1, strconv.Itoa(order2))
	assert.Equal(t, status.StOrderDuplicated, st)
	st = gm.NewOrder(user2, strconv.Itoa(order2))
	assert.Equal(t, status.StOrderAnotherUser, st)

	st = gm.NewOrder(user2, strconv.Itoa(order3))
	assert.Equal(t, status.StOk, st)
	st = gm.NewOrder(user2, strconv.Itoa(order4))
	assert.Equal(t, status.StOrderDuplicated, st)
	st = gm.NewOrder(user1, strconv.Itoa(order4))
	assert.Equal(t, status.StOrderAnotherUser, st)

	//Тут нужна дополнительная проверка того что при одновременной вставке одного пользователя работает защелка

	orderAdd1 := 10011
	orderAdd2 := 10012
	orderAdd3 := 10013

	mockStorage.On("CheckOrderExist", orderAdd1, user1).Return(status.StOrderNotExists)
	mockStorage.On("CheckOrderExist", orderAdd2, user1).Return(status.StOrderNotExists)
	mockStorage.On("CheckOrderExist", orderAdd3, user2).Return(status.StOrderNotExists)

	mockStorage.On("NewOrder", &model.DBOrder{Order: orderAdd1, User: user1, Status: model.OrderStNew}).After(1 * time.Second).Return(status.StOk)
	mockStorage.On("NewOrder", &model.DBOrder{Order: orderAdd2, User: user1, Status: model.OrderStNew}).Return(status.StOk)
	mockStorage.On("NewOrder", &model.DBOrder{Order: orderAdd3, User: user2, Status: model.OrderStNew}).Return(status.StOk)

	// Эти тесты будут выполняться параллельно
	t.Run("parallel_A", func(t *testing.T) {
		t.Parallel()
		start := time.Now()
		st = gm.NewOrder(user1, strconv.Itoa(orderAdd1)) //поставлена задержка ответа 1 секунду
		assert.Equal(t, status.StOk, st)
		assert.InDelta(t, 1*time.Second, time.Since(start), float64(10*time.Millisecond))
	})

	t.Run("parallel_B", func(t *testing.T) {
		t.Parallel()
		start := time.Now()
		st = gm.NewOrder(user1, strconv.Itoa(orderAdd2)) //т.к. выполнение параллельное а предыдущий тест выполняется 1 секунду, то этот не может выполниться быстрее из-за блокировки
		assert.Equal(t, status.StOk, st)
		assert.InDelta(t, 1*time.Second, time.Since(start), float64(10*time.Millisecond))
	})

	t.Run("parallel_C", func(t *testing.T) {
		t.Parallel()
		start := time.Now()
		st = gm.NewOrder(user2, strconv.Itoa(orderAdd3)) //т.к это другой пользователь тут должно быть моментальное выполнение
		assert.Equal(t, status.StOk, st)
		assert.LessOrEqual(t, time.Since(start), 100*time.Millisecond)
	})

}

func TestGetOrders(t *testing.T) {

	//Подготовка
	mockStorage := new(mocks.GofermartStorage)
	llog := logger.NewLogger("debug", io.Discard)

	gm, err := NewGofermart(llog, mockStorage)
	assert.NoError(t, err)

	//Ожидания моков
	tmp := 5000
	tm := time.Now()
	dborders := []model.DBOrder{}
	dborders = append(dborders, model.DBOrder{Order: 1, User: 1, Status: model.OrderStNew, Uploaded: tm, Updated: tm})
	dborders = append(dborders, model.DBOrder{Order: 2, User: 1, Status: model.OrderStProcessed, Accrual: &tmp, Uploaded: tm, Updated: tm})
	dborders = append(dborders, model.DBOrder{Order: 3, User: 1, Status: model.OrderStInvalid, Uploaded: tm, Updated: tm})

	mockStorage.On("GetOrders", 1).Return(&dborders, status.StOk)
	mockStorage.On("GetOrders", 2).Return(nil, status.StGeneralError)

	//Тесты
	orders, st := gm.GetOrders(1)
	assert.Equal(t, status.StOk, st)
	assert.Equal(t, len(dborders), len(orders))
	for idx, dborder := range dborders {
		var accrual *float32
		if dborder.Accrual != nil {
			tmp := float32(*dborder.Accrual) / 100
			accrual = &tmp
		}

		order := model.Order{Order: strconv.Itoa(dborder.Order), Status: dborder.Status, Accrual: accrual, Uploaded: dborder.Updated}
		assert.Equal(t, order, orders[idx])
	}

	orders, st = gm.GetOrders(2)
	assert.Equal(t, status.StGeneralError, st)

}

func TestGetBalance(t *testing.T) {

	//Подготовка
	mockStorage := new(mocks.GofermartStorage)
	llog := logger.NewLogger("debug", io.Discard)

	gm, err := NewGofermart(llog, mockStorage)
	assert.NoError(t, err)

	//Ожидания моков
	mockStorage.On("GetBalance", 1).Return(10033, 5055, status.StOk)
	mockStorage.On("GetBalance", 2).Return(0, 0, status.StGeneralError)

	//Тесты
	balance, st := gm.GetBalance(1)
	assert.Equal(t, status.StOk, st)
	assert.Equal(t, model.Balance{Amount: float32(100.33), Withdrawn: float32(50.55)}, balance)

	balance, st = gm.GetBalance(2)
	assert.Equal(t, status.StGeneralError, st)
	assert.Equal(t, model.Balance{Amount: float32(0), Withdrawn: float32(0)}, balance)

}

func TestNewWithdraw(t *testing.T) {
	//Подготовка
	mockStorage := new(mocks.GofermartStorage)
	llog := logger.NewLogger("debug", io.Discard)

	gm, err := NewGofermart(llog, mockStorage)
	assert.NoError(t, err)

	//Тесты
	st := gm.NewOrder(1, "123qwe")
	assert.Equal(t, status.StGeneralError, st)

	sum := float32(100.11)

	//Успех, заявка отсуствует, Суммы хватает
	user1, order1 := 1001, 1001
	mockStorage.On("CheckOrderExist", order1, user1).Return(status.StOrderNotExists)
	mockStorage.On("GetBalance", user1).Return(11000, 0, status.StOk)
	mockStorage.On("NewWithdraw", &model.DBWithdraw{Order: order1, User: user1, Sum: int(sum * 100)}).Return(status.StOk)
	st = gm.NewWithdraw(user1, &model.NewWithdraw{Order: strconv.Itoa(order1), Sum: sum})
	assert.Equal(t, status.StOk, st)

	//Заявка уже заведена, Суммы хватает
	user1, order1 = 1002, 1002
	mockStorage.On("CheckOrderExist", order1, user1).Return(status.StOrderDuplicated)
	mockStorage.On("GetBalance", user1).Return(11000, 0, status.StOk)
	mockStorage.On("NewWithdraw", &model.DBWithdraw{Order: order1, User: user1, Sum: int(sum * 100)}).Return(status.StOk)
	st = gm.NewWithdraw(user1, &model.NewWithdraw{Order: strconv.Itoa(order1), Sum: sum})
	assert.Equal(t, status.StOrderDuplicated, st)

	//Заявка заведена другим пользователем, Суммы хватает
	user1, order1 = 1003, 1003
	mockStorage.On("CheckOrderExist", order1, user1).Return(status.StOrderAnotherUser)
	mockStorage.On("GetBalance", user1).Return(11000, 0, status.StOk)
	mockStorage.On("NewWithdraw", &model.DBWithdraw{Order: order1, User: user1, Sum: int(sum * 100)}).Return(status.StOk)
	st = gm.NewWithdraw(user1, &model.NewWithdraw{Order: strconv.Itoa(order1), Sum: sum})
	assert.Equal(t, status.StOrderAnotherUser, st)

	//Заявка заявка отсуствует, Суммы НЕ хватает
	user1, order1 = 1004, 1004
	mockStorage.On("CheckOrderExist", order1, user1).Return(status.StOrderNotExists)
	mockStorage.On("GetBalance", user1).Return(5000, 0, status.StOk)
	mockStorage.On("NewWithdraw", &model.DBWithdraw{Order: order1, User: user1, Sum: int(sum * 100)}).Return(status.StOk)
	st = gm.NewWithdraw(user1, &model.NewWithdraw{Order: strconv.Itoa(order1), Sum: sum})
	assert.Equal(t, status.StWithdrawInsufficientFunds, st)

	//Ошибка CheckOrderExist
	user1, order1 = 1005, 1005
	mockStorage.On("CheckOrderExist", order1, user1).Return(status.StGeneralError)
	mockStorage.On("GetBalance", user1).Return(5000, 0, status.StOk)
	mockStorage.On("NewWithdraw", &model.DBWithdraw{Order: order1, User: user1, Sum: int(sum * 100)}).Return(status.StOk)
	st = gm.NewWithdraw(user1, &model.NewWithdraw{Order: strconv.Itoa(order1), Sum: sum})
	assert.Equal(t, status.StGeneralError, st)

	//Ошибка GetBalance
	user1, order1 = 1006, 1006
	mockStorage.On("CheckOrderExist", order1, user1).Return(status.StOrderNotExists)
	mockStorage.On("GetBalance", user1).Return(5000, 0, status.StGeneralError)
	mockStorage.On("NewWithdraw", &model.DBWithdraw{Order: order1, User: user1, Sum: int(sum * 100)}).Return(status.StOk)
	st = gm.NewWithdraw(user1, &model.NewWithdraw{Order: strconv.Itoa(order1), Sum: sum})
	assert.Equal(t, status.StGeneralError, st)

	//Ошибка NewWithdraw
	user1, order1 = 1007, 1007
	mockStorage.On("CheckOrderExist", order1, user1).Return(status.StOrderNotExists)
	mockStorage.On("GetBalance", user1).Return(5000, 0, status.StGeneralError)
	mockStorage.On("NewWithdraw", &model.DBWithdraw{Order: order1, User: user1, Sum: int(sum * 100)}).Return(status.StGeneralError)
	st = gm.NewWithdraw(user1, &model.NewWithdraw{Order: strconv.Itoa(order1), Sum: sum})
	assert.Equal(t, status.StGeneralError, st)

	//Тут нужна дополнительная проверка того что при одновременной вставке одного пользователя работает защелка

	user1 = 2001
	user2 := 2002

	orderAdd1 := 10011
	orderAdd2 := 10012
	orderAdd3 := 10013

	mockStorage.On("CheckOrderExist", orderAdd1, user1).Return(status.StOrderNotExists)
	mockStorage.On("GetBalance", user1).Return(11000, 0, status.StOk)
	mockStorage.On("NewWithdraw", &model.DBWithdraw{Order: orderAdd1, User: user1, Sum: int(sum * 100)}).After(1 * time.Second).Return(status.StOk)

	mockStorage.On("CheckOrderExist", orderAdd2, user1).Return(status.StOrderNotExists)
	mockStorage.On("GetBalance", user1).Return(11000, 0, status.StOk)
	mockStorage.On("NewWithdraw", &model.DBWithdraw{Order: orderAdd2, User: user1, Sum: int(sum * 100)}).Return(status.StOk)

	mockStorage.On("CheckOrderExist", orderAdd3, user2).Return(status.StOrderNotExists)
	mockStorage.On("GetBalance", user2).Return(11000, 0, status.StOk)
	mockStorage.On("NewWithdraw", &model.DBWithdraw{Order: orderAdd3, User: user2, Sum: int(sum * 100)}).Return(status.StOk)

	// Эти тесты будут выполняться параллельно
	t.Run("simultaneous_NewWithdraw_User1_A", func(t *testing.T) {
		t.Parallel()
		start := time.Now()
		st = gm.NewWithdraw(user1, &model.NewWithdraw{Order: strconv.Itoa(orderAdd1), Sum: sum})
		assert.Equal(t, status.StOk, st)
		//поставлена задержка ответа 1 секунду
		assert.InDelta(t, 1*time.Second, time.Since(start), float64(10*time.Millisecond))
	})

	t.Run("simultaneous_NewWithdraw_User1_B", func(t *testing.T) {
		t.Parallel()
		start := time.Now()
		st = gm.NewWithdraw(user1, &model.NewWithdraw{Order: strconv.Itoa(orderAdd2), Sum: sum})
		assert.Equal(t, status.StOk, st)
		//т.к. выполнение параллельное а предыдущий тест выполняется 1 секунду, то этот не может выполниться быстрее из-за блокировки
		assert.InDelta(t, 1*time.Second, time.Since(start), float64(10*time.Millisecond))
	})

	t.Run("simultaneous_NewWithdraw_User2", func(t *testing.T) {
		t.Parallel()
		start := time.Now()
		st = gm.NewWithdraw(user2, &model.NewWithdraw{Order: strconv.Itoa(orderAdd3), Sum: sum})
		assert.Equal(t, status.StOk, st)
		//т.к это другой пользователь тут должно быть моментальное выполнение
		assert.LessOrEqual(t, time.Since(start), 10*time.Millisecond)
	})

}

func TestGetWithdrawals(t *testing.T) {
	//Подготовка
	mockStorage := new(mocks.GofermartStorage)
	llog := logger.NewLogger("debug", io.Discard)

	gm, err := NewGofermart(llog, mockStorage)
	assert.NoError(t, err)

	//Ожидания моков
	tm := time.Now()
	dbwithds := []model.DBWithdraw{}
	dbwithds = append(dbwithds, model.DBWithdraw{Order: 1, User: 1, Uploaded: tm, Sum: 1033})
	dbwithds = append(dbwithds, model.DBWithdraw{Order: 2, User: 1, Uploaded: tm, Sum: 1055})
	dbwithds = append(dbwithds, model.DBWithdraw{Order: 3, User: 1, Uploaded: tm, Sum: 1011})

	mockStorage.On("GetWithdrawals", 1).Return(&dbwithds, status.StOk)
	mockStorage.On("GetWithdrawals", 2).Return(nil, status.StGeneralError)

	//Тесты
	withds, st := gm.GetWithdrawals(1)
	assert.Equal(t, status.StOk, st)
	assert.Equal(t, len(dbwithds), len(withds))
	for idx, dbwithd := range dbwithds {
		withd := model.Withdraw{Order: strconv.Itoa(dbwithd.Order), Sum: float32(dbwithd.Sum) / 100, Uploaded: dbwithd.Uploaded}
		assert.Equal(t, withd, withds[idx])
	}

	withds, st = gm.GetWithdrawals(2)
	assert.Equal(t, status.StGeneralError, st)

}
