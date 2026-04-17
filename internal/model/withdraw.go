package model

import (
	"time"
)

type DBWithdraw struct {
	Order    int
	User     int
	Sum      int
	Uploaded time.Time
}

type NewWithdraw struct {
	Order string  `json:"order"`
	Sum   float32 `json:"sum"`
}

type Withdraw struct {
	Order    string    `json:"order"`
	Sum      float32   `json:"sum"`
	Uploaded time.Time `json:"processed_at"`
}
