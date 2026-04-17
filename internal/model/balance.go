package model

type Balance struct {
	Amount    float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}
