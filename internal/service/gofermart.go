package service

import (
	"sync"

	"github.com/qesterrx/gofermart/internal/auth"
	"github.com/qesterrx/gofermart/internal/logger"
	"github.com/qesterrx/gofermart/internal/model"
	"github.com/qesterrx/gofermart/internal/status"
	"golang.org/x/crypto/bcrypt"
)

// GofermartStorage - Интерфейс для работы с хранилищем данных
type GofermartStorage interface {
	NewUser(login, password string) status.Status
	GetUser(login string) (*model.DBUser, status.Status)
	CheckOrderExist(order int, user int) status.Status
	NewOrder(order *model.DBOrder) status.Status
	GetOrders(user int) (*[]model.DBOrder, status.Status)
	GetBalance(user int) (int, int, status.Status)
	NewWithdraw(withdraw *model.DBWithdraw) status.Status
	GetWithdrawals(user int) (*[]model.DBWithdraw, status.Status)
}

// Gofermart - основная часть сервисного слоя, обеспечивает связь Handler - DB
// Новый экземпляр создается функцией NewGofermart
type Gofermart struct {
	log     *logger.Logger
	storage GofermartStorage

	locks   map[int]*sync.Mutex
	mxLocks sync.Mutex
}

// NewGofermart - создает новый Gofermart
// Входящие параметры:
// logger *logger.Logger - ссылка на логгер
// storage GofermartStorage - реализация интерфейса для работы с хранилищем данных
func NewGofermart(logger *logger.Logger, storage GofermartStorage) (*Gofermart, error) {

	llog := logger.With("service")

	//Сделаем мапу защелок, ключем будет выступать ИД пользователя
	locks := make(map[int]*sync.Mutex)

	return &Gofermart{log: llog, storage: storage, locks: locks}, nil
}

// Login - функция авторизации пользователя по набору логин/пароль
// При успехе возвращает строку JWT авторизации (токен) и статус status.StOk
// При не успехе возвращает пустой токен и один из статусов:
// status.StUserWrongPassword - Пользователь не найден или пароль не совпал
// status.StGeneralError - Внутренняя ошибка генерации токена JWT
func (gm *Gofermart) Login(login, password string) (string, status.Status) {

	user, st := gm.storage.GetUser(login)

	if st == status.StUserNotFound || st == status.StOk && bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		gm.log.Info("Ошибка авторизации в методе UserLogin")
		return "", status.StUserWrongPassword
	}

	if st != status.StOk {
		return "", status.StGeneralError
	}

	accessToken, err := auth.GenerateAccessToken(user.ID, user.Login)
	if err != nil {
		gm.log.Error("Ошибка генерации токена JWT")
		return "", status.StGeneralError
	}

	return accessToken, status.StOk

}

// Register - функция создания нового пользователя по набору логин/пароль
// При успехе возвращает строку JWT авторизации (токен) и статус status.StOk
// При не успехе возвращает пустой токен и один из статусов:
// status.StUserAlreadyExists - Имя пользователя занято
// status.StGeneralError - Общая ошибка, может быть связана с генерацией хеша пароля или ошибкой работы с БД
func (gm *Gofermart) Register(login, password string) status.Status {

	pswd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		gm.log.Error("Ошибка генерации hash пароля")
		return status.StGeneralError
	}

	st := gm.storage.NewUser(login, string(pswd))

	return st

}
