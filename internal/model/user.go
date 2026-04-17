package model

type DBUser struct {
	ID       int
	Login    string
	Password string
}

type AuthUser struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
