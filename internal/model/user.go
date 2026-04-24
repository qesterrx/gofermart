package model

// DBUser - Модель БД описывающая пользователя
type DBUser struct {
	ID       int
	Login    string
	Password string
}

// AuthUser - Модель используемая для авторизации в сервисе
type AuthUser struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
