package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/qesterrx/gofermart/internal/auth"
	"github.com/qesterrx/gofermart/internal/logger"
	"github.com/qesterrx/gofermart/internal/model"
	"github.com/qesterrx/gofermart/internal/status"
)

type GofermartService interface {
	Login(login, password string) (string, status.Status)
	Register(login, password string) status.Status
	CheckOrderNumber(order string) error
	NewOrder(user int, order string) status.Status
	GetOrders(user int) ([]model.Order, status.Status)
	GetBalance(user int) (model.Balance, status.Status)
	NewWithdraw(user int, wd *model.NewWithdraw) status.Status
	GetWithdrawals(user int) ([]model.Withdraw, status.Status)
}

// HandlerContainer - контейнер с хендлерами сервиса
// Новый контейнер создается через функцию NewHandlerContainer
type HandlerContainer struct {
	log *logger.Logger
	gms GofermartService
}

// NewHandlerContainer - Создает новый контейнер хендлеров
// Входные параметры:
// logger *logger.Logger - ссылка на логгер
// service *service.Gofermart - ссылка на реализацию сервисного слоя
func NewHandlerContainer(logger *logger.Logger, service GofermartService) (*HandlerContainer, error) {
	return &HandlerContainer{log: logger, gms: service}, nil
}

// PostUserRegister - Регистрация нового пользователя
func (hc *HandlerContainer) PostUserRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ul := model.AuthUser{}

	err := json.NewDecoder(r.Body).Decode(&ul)
	if err != nil {
		hc.log.Error("Ошибка десериализации в методе UserRegister")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	st := hc.gms.Register(ul.Login, ul.Password)

	var accessToken string
	if st == status.StOk {
		accessToken, st = hc.gms.Login(ul.Login, ul.Password)
	}

	switch st {
	case status.StGeneralError:
		w.WriteHeader(http.StatusInternalServerError)
		return
	case status.StUserAlreadyExists:
		w.WriteHeader(http.StatusConflict)
		return
	case status.StUserWrongPassword:
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if st != status.StOk {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.JWTCookieName,
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(auth.JWTExpire.Seconds()),
	})

	hc.log.Debug("Call PostUserRegister")

	w.WriteHeader(http.StatusOK)

}

// PostUserLogin - Авторизация пользователя
func (hc *HandlerContainer) PostUserLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ul := model.AuthUser{}

	err := json.NewDecoder(r.Body).Decode(&ul)
	if err != nil {
		hc.log.Error("Ошибка десериализации в методе UserLogin")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	accessToken, st := hc.gms.Login(ul.Login, ul.Password)
	switch st {
	case status.StGeneralError:
		w.WriteHeader(http.StatusInternalServerError)
		return
	case status.StUserWrongPassword:
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if st != status.StOk {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.JWTCookieName,
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(auth.JWTExpire.Seconds()),
	})

	w.WriteHeader(http.StatusOK)
}

// PostUserOrders - новый заказ для добавления в расчет бонусных балов
func (hc *HandlerContainer) PostUserOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "text/plain" || r.ContentLength == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	jwtc, ok := r.Context().Value("user").(*auth.JWTC)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	hc.log.Debug(fmt.Sprintf("Call PostUserOrder, User=%d name=%s", jwtc.UserID, jwtc.Username))

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	order := string(body)

	err = hc.gms.CheckOrderNumber(order)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	st := hc.gms.NewOrder(jwtc.UserID, order)

	switch st {
	case status.StOrderDuplicated:
		w.WriteHeader(http.StatusOK)
	case status.StOrderAnotherUser:
		w.WriteHeader(http.StatusConflict)
	case status.StOk:
		w.WriteHeader(http.StatusAccepted)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

}

// GetUserOrders - Получение списка заказов пользователя
func (hc *HandlerContainer) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	jwtc, ok := r.Context().Value("user").(*auth.JWTC)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	hc.log.Debug(fmt.Sprintf("Call GetUserOrder, User=%d name=%s", jwtc.UserID, jwtc.Username))

	orders, st := hc.gms.GetOrders(jwtc.UserID)
	if st != status.StOk {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	body, err := json.MarshalIndent(&orders, "", " ")
	if err != nil {
		hc.log.Error(fmt.Sprintf("Ошибка сериализации данных GetUserOrders UserID=%d", jwtc.UserID))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)

}

// GetUserBalance - Получение баланса пользователя
func (hc *HandlerContainer) GetUserBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	jwtc, ok := r.Context().Value("user").(*auth.JWTC)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	hc.log.Debug(fmt.Sprintf("Call GetUserBalance, User=%d name=%s", jwtc.UserID, jwtc.Username))

	balance, st := hc.gms.GetBalance(jwtc.UserID)

	if st != status.StOk {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	body, err := json.MarshalIndent(&balance, "", " ")
	if err != nil {
		hc.log.Error(fmt.Sprintf("Ошибка сериализации данных GetUserOrders UserID=%d", jwtc.UserID))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

// PostUserBalanceWithdraw - Новое списание бонусных балов
func (hc *HandlerContainer) PostUserBalanceWithdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	jwtc, ok := r.Context().Value("user").(*auth.JWTC)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	hc.log.Debug(fmt.Sprintf("Call PostUserBalanceWithdraw, User=%d name=%s", jwtc.UserID, jwtc.Username))

	withdraw := model.NewWithdraw{}

	err := json.NewDecoder(r.Body).Decode(&withdraw)
	if err != nil {
		hc.log.Error(fmt.Sprintf("Ошибка сериализации данных GetUserBalance UserID=%d", jwtc.UserID))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	err = hc.gms.CheckOrderNumber(withdraw.Order)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	st := hc.gms.NewWithdraw(jwtc.UserID, &withdraw)
	if st == status.StWithdrawInsufficientFunds {
		w.WriteHeader(http.StatusPaymentRequired)
		return
	}

	if st != status.StOk {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetUserWithdrawals - Список списаний бонусных балов
func (hc *HandlerContainer) GetUserWithdrawals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	jwtc, ok := r.Context().Value("user").(*auth.JWTC)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	hc.log.Debug(fmt.Sprintf("Call GetUserWithdrawals, User=%d name=%s", jwtc.UserID, jwtc.Username))

	withdrawals, st := hc.gms.GetWithdrawals(jwtc.UserID)
	if st != status.StOk {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	body, err := json.MarshalIndent(&withdrawals, "", " ")
	if err != nil {
		hc.log.Error(fmt.Sprintf("Ошибка сериализации данных GetUserWithdrawals UserID=%d", jwtc.UserID))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}
