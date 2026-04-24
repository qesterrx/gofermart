package model

import (
	"time"
)

// OrderStatus - Статус заявки в системе
type OrderStatus string

const (
	OrderStNew        OrderStatus = "NEW"
	OrderStInvalid    OrderStatus = "INVALID"
	OrderStProcessing OrderStatus = "PROCESSING"
	OrderStProcessed  OrderStatus = "PROCESSED"
)

// DBOrder Модель БД описывающая заявки на начисление бонусных балов
type DBOrder struct {
	Order    int
	User     int
	Status   OrderStatus
	Accrual  *int
	Uploaded time.Time
	Updated  time.Time
}

// Order Модель для Handlerов, описывающая заявки на начисление бонусных балов
type Order struct {
	Order    string      `json:"number"`
	Status   OrderStatus `json:"status"`
	Accrual  *float32    `json:"accrual"`
	Uploaded time.Time   `json:"uploaded_at"`
}
