package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/qesterrx/gofermart/internal/config"
	"github.com/qesterrx/gofermart/internal/handler"
	"github.com/qesterrx/gofermart/internal/logger"
	"github.com/qesterrx/gofermart/internal/server"
	"github.com/qesterrx/gofermart/internal/service"
	"github.com/qesterrx/gofermart/internal/storage"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {

	//----------------Создаем необходимые сущности
	//Конфиг приложения
	cfg, err := config.ParseParamsServer()
	if err != nil {
		return err
	}

	//Логгер
	llog := logger.NewLogger(cfg.LogMode, nil)

	//Сторадж
	storage, err := storage.NewStoragePGSQL(llog, cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	llog.Debug("Создан storage")

	//Сервис accrual
	accrual, err := service.NewAccrual(llog, storage, cfg.AccrualHost.String())
	if err != nil {
		return err
	}
	llog.Debug("Создан accrual")

	//Сервис gofermart
	gofermart, err := service.NewGofermart(llog, storage)
	if err != nil {
		return err
	}
	llog.Debug("Создан gofermart")

	//Хендлеры
	hc, err := handler.NewHandlerContainer(llog, gofermart)
	if err != nil {
		return err
	}
	llog.Debug("Создан hc")

	//Сервер
	server := server.NewServer(llog, cfg.ServerHost.String(), hc)
	llog.Debug("Создан server")

	//----------------Запускаем сервер
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	llog.Debug("Запуск http сервера")
	g.Go(func() error {
		err := server.ListenAndServe()
		if ctx.Err() == nil {
			//Ошибку отображаем только если контекст не завершен
			llog.Error("Ошибка в работе сервера ListenAndServe:" + err.Error())
			return err
		}
		return nil
	})
	llog.Info("Http сервер запущен по адресу " + cfg.ServerHost.String())

	//----------------Запускаем асинхронную обработку очереди сервисного слоя
	g.Go(func() error {
		accrual.RunCheckaccrualAsync(ctx, 20, 2*time.Second)
		return nil
	})

	//----------------Остановка

	//Канал для сигналов ОС
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	//Ждем сигнала завершения
	<-sigChan
	cancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	//Остановка сервера
	if err := server.Shutdown(ctxShutdown); err != nil {
		llog.Error("Ошибка остановки работы сервера:" + err.Error())
	}

	llog.Info("Http сервер остановлен")

	return g.Wait()
}
