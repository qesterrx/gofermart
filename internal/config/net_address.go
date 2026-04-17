package config

import (
	"fmt"
	"strconv"
	"strings"
)

type NetAddress struct {
	Host string
	Port int
}

func (na *NetAddress) String() string {
	return na.Host + ":" + strconv.Itoa(na.Port)
}

func (na *NetAddress) Set(value string) error {

	url := value

	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "ftp://")
	url = strings.TrimPrefix(url, "ws://")
	url = strings.TrimPrefix(url, "wss://")

	split := strings.Split(url, ":")
	if len(split) != 2 {
		return fmt.Errorf("параметр должен иметь формат хост:порт")
	}

	port, err := strconv.Atoi(split[1])
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("порт должен быть задан числом до 65535")
	}

	//	if net.ParseIP(split[0]) == nil {
	//		return fmt.Errorf("хост должен быть задан в виде IP")
	//	}

	na.Port = port
	na.Host = split[0]

	return nil
}
