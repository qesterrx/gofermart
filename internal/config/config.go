package config

import (
	"flag"
	"fmt"
	"os"
)

// Config - структура для хранения параметров запуска приложения
type Config struct {
	ServerHost  NetAddress
	AccrualHost NetAddress
	DatabaseDSN string
	LogMode     string
}

// ParseParamsServer - функция заполнения Config параметрами, переданными при запуске приложения
// а так же анализом ожидаемых переменных окружения
func ParseParamsServer() (*Config, error) {

	fs := flag.NewFlagSet("", flag.PanicOnError)

	var cfg Config

	cfg.ServerHost = NetAddress{Host: "localhost", Port: 8081}
	cfg.AccrualHost = NetAddress{Host: "localhost", Port: 8080}

	fs.Var(&cfg.ServerHost, "a", "адрес запуска приложения (host:port)")
	fs.Var(&cfg.AccrualHost, "r", "адрес приложения интеграции с accrual (host:port)")
	fs.StringVar(&cfg.DatabaseDSN, "d", "postgres://user:pswd@localhost:5432/db?sslmode=disable", "строка соединения с БД")
	fs.StringVar(&cfg.LogMode, "m", "info", "уровень логирования debug/info/error")

	fs.Parse(os.Args[1:])

	//Переопределение через переменные окружения
	if envServerHost := os.Getenv("RUN_ADDRESS"); envServerHost != "" {
		newServerHost := NetAddress{}
		err := newServerHost.Set(envServerHost)
		if err != nil {
			return nil, fmt.Errorf("переменная окружения RUN_ADDRESS: %s", err.Error())
		}
		cfg.ServerHost = newServerHost
	}

	if envAccrualHost := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrualHost != "" {
		newAccrualHost := NetAddress{}
		err := newAccrualHost.Set(envAccrualHost)
		if err != nil {
			return nil, fmt.Errorf("переменная окружения ACCRUAL_SYSTEM_ADDRESS: %s", err.Error())
		}
		cfg.AccrualHost = newAccrualHost
	}

	if envDatabaseDNS := os.Getenv("DATABASE_URI"); envDatabaseDNS != "" {
		cfg.DatabaseDSN = envDatabaseDNS
	}

	if !(cfg.LogMode == "debug" || cfg.LogMode == "info" || cfg.LogMode == "error") {
		return nil, fmt.Errorf("режим логирования может принимать значения debug/info/error")
	}

	return &cfg, nil

}
