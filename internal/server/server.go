package server

import (
	"context"
	"net/http"
	"time"

	"github.com/qesterrx/gofermart/internal/handler"
	"github.com/qesterrx/gofermart/internal/logger"
	"github.com/qesterrx/gofermart/internal/middleware"
)

// ServerGofermatr - контейнер http сервера
// Для создания использовать NewServer
type ServerGofermatr struct {
	log    *logger.Logger
	server *http.Server
}

// NewServer - создает новый ServerGofermatr
// Входящие параметры:
// log *logger.Logger - ссылка на логгер
// address string - адрес запуска http сервера в формате host:port
// handlers *handler.HandlerContainer - ссылка на объект HandlerContainer
func NewServer(log *logger.Logger, address string, handlers *handler.HandlerContainer) *ServerGofermatr {

	mux := http.NewServeMux()

	//Middleware
	logging := middleware.LoggingHandler(log)
	JWT := middleware.JWTAccess
	JSON := middleware.JsonContentType

	UserRegister := JSON(logging(http.HandlerFunc(handlers.PostUserRegister)))
	UserLogin := JSON(logging(http.HandlerFunc(handlers.PostUserLogin)))
	UserOrders := JWT(logging(http.HandlerFunc(handlers.MethodUserOrders)))
	UserBalance := JWT(logging(http.HandlerFunc(handlers.GetUserBalance)))
	UserBalanceWithdraw := JSON(JWT(logging(http.HandlerFunc(handlers.PostUserBalanceWithdraw))))
	UserWithdrawals := JWT(logging(http.HandlerFunc(handlers.GetUserWithdrawals)))

	//Route
	mux.Handle("/api/user/register", UserRegister)
	mux.Handle("/api/user/login", UserLogin)
	mux.Handle("/api/user/orders", UserOrders)
	mux.Handle("/api/user/balance", UserBalance)
	mux.Handle("/api/user/balance/withdraw", UserBalanceWithdraw)
	mux.Handle("/api/user/withdrawals", UserWithdrawals)

	//Server
	server := &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadTimeout:       2 * time.Second,  // Максимальное время на чтение запроса - запросы короткие, данных мало
		ReadHeaderTimeout: 1 * time.Second,  // Время чтения заголовка - меньше чем ReadTimeout
		WriteTimeout:      5 * time.Second,  // Максимальное время на запись ответа
		IdleTimeout:       60 * time.Second, // Таймаут для keep-alive соединений
	}

	return &ServerGofermatr{server: server, log: log}
}

// ListenAndServe - обертка http.Server.ListenAndServe
func (sg *ServerGofermatr) ListenAndServe() error {
	return sg.server.ListenAndServe()
}

// Shutdown - обертка http.Server.Shutdown
func (sg *ServerGofermatr) Shutdown(ctx context.Context) error {
	return sg.server.Shutdown(ctx)
}
