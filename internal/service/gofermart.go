package service

import (
	"time"

	"github.com/qesterrx/gofermart/internal/auth"
	"github.com/qesterrx/gofermart/internal/logger"
	"github.com/qesterrx/gofermart/internal/model"
	"github.com/qesterrx/gofermart/internal/status"
	"golang.org/x/crypto/bcrypt"
)

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

type Gofermart struct {
	log     *logger.Logger
	storage GofermartStorage
	wul     map[int]time.Time
	wulttl  time.Duration
}

func NewGofermart(logger *logger.Logger, storage GofermartStorage, delayUserWithdraw time.Duration) (*Gofermart, error) {

	llog := logger.With("service")
	wul := map[int]time.Time{}

	return &Gofermart{log: llog, storage: storage, wul: wul, wulttl: delayUserWithdraw}, nil
}

func (gm *Gofermart) Login(login, password string) (string, status.Status) {

	user, st := gm.storage.GetUser(login)

	if st != status.StOk || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		gm.log.Info("Ошибка авторизации в методе UserLogin")
		return "", status.StUserWrongPassword
	}

	accessToken, err := auth.GenerateAccessToken(user.ID, user.Login)
	if err != nil {
		gm.log.Error("Ошибка генерации токена JWT")
		return "", status.StErrorGenerateJWT
	}

	return accessToken, status.StUserLogined

}

func (gm *Gofermart) Register(login, password string) (string, status.Status) {

	pswd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		gm.log.Error("Ошибка генерации hash пароля")
		return "", status.StGeneralError
	}

	st := gm.storage.NewUser(login, string(pswd))

	if st != status.StOk {
		return "", st
	}

	return gm.Login(login, password)

}
