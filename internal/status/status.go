package status

type Status int

const (
	StOk Status = iota
	StGeneralError
	StErrorGenerateJWT
	StUserLogined
	StUserWrongPassword
	StUserAlreadyExists
	StUserNotFound
	StOrderDuplicated
	StOrderAnotherUser
	StWithdrawInsufficientFunds
)
