package model

import (
	"time"
)

// DBWithdraw - Модель БД описывающая заявки на списание бонусных балов
type DBWithdraw struct {
	Order    int
	User     int
	Sum      int
	Uploaded time.Time
}

// NewWithdraw - Модель для Handlerа заведения нового списания
type NewWithdraw struct {
	Order string  `json:"order"`
	Sum   float32 `json:"sum"`
}

// Withdraw - Модель для Handlerов отображения списаний
type Withdraw struct {
	Order    string    `json:"order"`
	Sum      float32   `json:"sum"`
	Uploaded time.Time `json:"processed_at"`
}
