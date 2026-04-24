package status

// Status - Статусы для связывания работы Handler - Service - DB
type Status int

const (
	//Общий статус успех
	StOk Status = iota

	//Общая ошибка без детализации
	StGeneralError

	//Некорректный пароль
	StUserWrongPassword

	//Попытка создать пользователя с зарегистрированным логином
	StUserAlreadyExists

	//Пользователь по переданному логину не найден
	StUserNotFound

	//Попытка зарегистрировать уже существующий заказ
	StOrderDuplicated

	//Попытка зарегистрировать заказ который уже был зарегистрирован другим ползователем
	StOrderAnotherUser

	//Заказ не существует
	StOrderNotExists

	//Недостаточно средств для списания бонусов в счет заказа
	StWithdrawInsufficientFunds
)
