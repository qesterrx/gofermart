package model

type AccrualStatus string

const (
	AccrualStRegistered AccrualStatus = "REGISTERED"
	AccrualStInvalid    AccrualStatus = "INVALID"
	AccrualStProcessing AccrualStatus = "PROCESSING"
	AccrualStProcessed  AccrualStatus = "PROCESSED"
)

type Accrual struct {
	Order  string        `json:"order"`
	Status AccrualStatus `json:"status"`
	Sum    *float32      `json:"accrual"`
}
