package service

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/qesterrx/gofermart/internal/model"
	"github.com/qesterrx/gofermart/internal/status"
)

// CheckOrderNumber функция проверки номера заявки
// Проверяет что переданная строка это число, подходящее под алгоритм Луна, в противном случае возвращает ошибку
func (gm *Gofermart) CheckOrderNumber(order string) error {

	if len(order) == 0 {
		return fmt.Errorf("пустой order")
	}

	sum := 0
	digits := len(order)
	parity := digits % 2

	for i := 0; i < digits; i++ {
		// Преобразуем символ в цифру
		digit := int(order[i] - '0')

		// Проверка, что это действительно цифра
		if digit < 0 || digit > 9 {
			return fmt.Errorf("order: ошибка формата")
		}

		if i%2 == parity {
			digit = digit * 2
			if digit > 9 {
				digit = digit - 9
			}
		}
		sum = sum + digit
	}

	if (sum % 10) != 0 {
		return fmt.Errorf("order: ошибка контрольной суммы")
	}

	return nil
}

// NewOrder - функция регистрации нового заказа на расчет бонусов
// Входящие параметры:
// user int - ИД пользователя
// order string - номер заказа
// При успехе возвращает статус status.StOk
// При не успехе возвращает один из статусов status.St*
func (gm *Gofermart) NewOrder(user int, order string) status.Status {

	ord, err := strconv.Atoi(order)
	if err != nil {
		return status.StGeneralError
	}

	//Нужно сделать контроль что одновременно не будут создаваться заявки одного пользователя
	gm.mxLocks.Lock()
	userLock, exists := gm.locks[user]

	if !exists {
		userLock = &sync.Mutex{}
		gm.locks[user] = userLock
	}

	gm.mxLocks.Unlock() //освобождаем общий мьютекс
	userLock.Lock()     //занимаем мьютекс пользователя
	defer userLock.Unlock()

	st := gm.storage.CheckOrderExist(ord, user)
	if st != status.StOrderNotExists {
		return st
	}

	st = gm.storage.NewOrder(&model.DBOrder{Order: ord, User: user, Status: model.OrderStNew})

	return st
}

// GetOrders - функция возвращающая заказы пользователя на расчет бонусов
// Входящие параметры:
// user int - ИД пользователя
// При успехе возвращает массив заказов в виде model.Order и статус status.StOk
// При не успехе возвращает один из статусов status.St*
func (gm *Gofermart) GetOrders(user int) ([]model.Order, status.Status) {
	dborders, st := gm.storage.GetOrders(user)

	if st != status.StOk {
		return nil, status.StGeneralError
	}

	orders := []model.Order{}
	for _, dbord := range *dborders {
		var accrual *float32
		if dbord.Accrual != nil {
			tmp := float32(*dbord.Accrual) / 100
			accrual = &tmp
		}
		orders = append(orders, model.Order{Order: strconv.Itoa(dbord.Order), Status: dbord.Status, Accrual: accrual, Uploaded: dbord.Uploaded})
	}

	return orders, status.StOk
}

// GetBalance - функция возвращающая баланс пользоватлея
// Входящие параметры:
// user int - ИД пользователя
// При успехе возвращает объект model.Balance и статус status.StOk
// При не успехе возвращает объект model.Balance (со значениями 0) и один из статусов status.St*
func (gm *Gofermart) GetBalance(user int) (model.Balance, status.Status) {

	amount, withdraw, st := gm.storage.GetBalance(user)

	return model.Balance{Amount: float32(amount) / 100, Withdrawn: float32(withdraw) / 100}, st
}

// NewWithdraw - функция создает новое списание балов в счет заказа
// Входящие параметры:
// user int - ИД пользователя
// wd *model.NewWithdraw - объект модели нового списания
// При успехе возвращает статус status.StOk
// При не успехе возвращает один из статусов status.St*
func (gm *Gofermart) NewWithdraw(user int, wd *model.NewWithdraw) status.Status {

	ord, err := strconv.Atoi(wd.Order)
	if err != nil {
		return status.StGeneralError
	}

	//Переводим в копейки
	tmpSum := int(wd.Sum * 100)

	//Нужно сделать контроль что одновременно не будут выполняться списания одного пользователя
	gm.mxLocks.Lock()
	userLock, exists := gm.locks[user]

	if !exists {
		userLock = &sync.Mutex{}
		gm.locks[user] = userLock
	}

	gm.mxLocks.Unlock() //освобождаем общий мьютекс
	userLock.Lock()     //занимаем мьютекс пользователя
	defer userLock.Unlock()

	//Проверяем заявку
	st := gm.storage.CheckOrderExist(ord, user)
	if st != status.StOrderNotExists {
		return st
	}

	//Проверяем баланс
	amount, _, st := gm.storage.GetBalance(user)
	if st != status.StOk {
		return status.StGeneralError
	}

	//Проверяем достаточность суммы
	if tmpSum > amount {
		return status.StWithdrawInsufficientFunds
	}

	withdraw := model.DBWithdraw{Order: ord, User: user, Sum: tmpSum}
	st = gm.storage.NewWithdraw(&withdraw)

	return st
}

// GetWithdrawals - функция возвращающая заказы пользователя на списание бонусов
// Входящие параметры:
// user int - ИД пользователя
// При успехе возвращает массив заказов в виде model.Withdraw и статус status.StOk
// При не успехе возвращает один из статусов status.St*
func (gm *Gofermart) GetWithdrawals(user int) ([]model.Withdraw, status.Status) {
	dbwds, st := gm.storage.GetWithdrawals(user)

	if st != status.StOk {
		return nil, status.StGeneralError
	}

	wds := []model.Withdraw{}
	for _, wd := range *dbwds {
		sum := float32(wd.Sum) / 100
		wds = append(wds, model.Withdraw{Order: strconv.Itoa(wd.Order), Sum: sum, Uploaded: wd.Uploaded})
	}

	return wds, st
}
