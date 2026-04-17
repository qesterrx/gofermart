package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/qesterrx/gofermart/internal/config"
	"github.com/qesterrx/gofermart/internal/handler"
	"github.com/qesterrx/gofermart/internal/logger"
	"github.com/qesterrx/gofermart/internal/server"
	"github.com/qesterrx/gofermart/internal/service"
	"github.com/qesterrx/gofermart/internal/storage"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//----------------Создаем необходимые сущности
	//Конфиг приложения
	cfg, err := config.ParseParamsServer()
	if err != nil {
		return err
	}

	//Логгер
	llog := logger.NewLogger(cfg.DebugMode, nil)

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
	gofermart, err := service.NewGofermart(llog, storage, 2*time.Second)
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
	var wg sync.WaitGroup

	wg.Add(1)
	llog.Debug("Запуск http сервера")
	go func() {
		defer wg.Done()
		err := server.ListenAndServe()
		if ctx.Err() == nil {
			//Ошибку отображаем только если контекст не завершен
			llog.Error("Ошибка в работе сервера ListenAndServe:" + err.Error())
		}
		cancel()
	}()
	llog.Info("Http сервер запущен по адресу " + cfg.ServerHost.String())

	//----------------Запускаем асинхронную обработку очереди сервисного слоя

	/*	wg.Add(1)
		go func() {
			defer wg.Done()
			storage.RunSaveData(ctx)
			cancel()
		}()*/

	wg.Add(1)
	go func() {
		defer wg.Done()
		accrual.RunCheckaccrualAsync(ctx, 20, 2*time.Second)
		cancel()
	}()

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
	wg.Wait()
	return nil
}
