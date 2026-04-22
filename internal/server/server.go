package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
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

	r := chi.NewRouter()

	r.Use(middleware.LoggingHandler(log))

	r.Group(func(r chi.Router) {
		r.Use(middleware.JsonContentType)
		r.Post("/api/user/register", handlers.PostUserRegister)
		r.Post("/api/user/login", handlers.PostUserLogin)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.JWTAccess)
		r.Post("/api/user/orders", handlers.PostUserOrders)
		r.Get("/api/user/orders", handlers.GetUserOrders)
		r.Get("/api/user/balance", handlers.GetUserBalance)
		r.With(middleware.JsonContentType).Post("/api/user/balance/withdraw", handlers.PostUserBalanceWithdraw)
		r.Get("/api/user/withdrawals", handlers.GetUserWithdrawals)
	})

	//Server
	server := &http.Server{
		Addr:              address,
		Handler:           r,
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
