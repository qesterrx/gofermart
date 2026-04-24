package model

// AccrualStatus - статус заявки в сервисе расчета бонусных балов
type AccrualStatus string

const (
	AccrualStRegistered AccrualStatus = "REGISTERED"
	AccrualStInvalid    AccrualStatus = "INVALID"
	AccrualStProcessing AccrualStatus = "PROCESSING"
	AccrualStProcessed  AccrualStatus = "PROCESSED"
)

// Accrual - модель для десериализации данных из сервиса расчета бонусных балов
type Accrual struct {
	Order  string        `json:"order"`
	Status AccrualStatus `json:"status"`
	Sum    *float32      `json:"accrual"`
}
