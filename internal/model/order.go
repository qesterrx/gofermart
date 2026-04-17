package model

import (
	"time"
)

type OrderStatus string

const (
	OrderStNew        OrderStatus = "NEW"
	OrderStInvalid    OrderStatus = "INVALID"
	OrderStProcessing OrderStatus = "PROCESSING"
	OrderStProcessed  OrderStatus = "PROCESSED"
)

type DBOrder struct {
	Order    int
	User     int
	Status   OrderStatus
	Accrual  *int
	Uploaded time.Time
	Updated  time.Time
}

type Order struct {
	Order    string      `json:"number"`
	Status   OrderStatus `json:"status"`
	Accrual  *float32    `json:"accrual"`
	Uploaded time.Time   `json:"uploaded_at"`
}
