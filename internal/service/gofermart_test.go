package service

import (
	"io"
	"testing"

	"github.com/qesterrx/gofermart/internal/logger"
	"github.com/qesterrx/gofermart/internal/model"
	"github.com/qesterrx/gofermart/internal/status"
	"github.com/qesterrx/gofermart/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func TestLogin(t *testing.T) {

	//Подготовка
	mockStorage := new(mocks.GofermartStorage)
	llog := logger.NewLogger("debug", io.Discard)

	gm, err := NewGofermart(llog, mockStorage)
	assert.NoError(t, err)

	pswd, err := bcrypt.GenerateFromPassword([]byte("CorrectPassword"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	//Ожидания моков
	mockStorage.On("GetUser", "ExistedUser").Return(&model.DBUser{ID: 100, Login: "ExistedUser", Password: string(pswd)}, status.StOk)
	mockStorage.On("GetUser", "UnExistedUser").Return(nil, status.StUserNotFound)
	mockStorage.On("GetUser", "InternalError").Return(nil, status.StGeneralError)

	//Тесты
	_, st := gm.Login("ExistedUser", "CorrectPassword")
	assert.Equal(t, status.StOk, st)

	_, st = gm.Login("ExistedUser", "InCorrectPassword")
	assert.Equal(t, status.StUserWrongPassword, st)

	_, st = gm.Login("UnExistedUser", "DUMP")
	assert.Equal(t, status.StUserWrongPassword, st)

	_, st = gm.Login("InternalError", "DUMP")
	assert.Equal(t, status.StGeneralError, st)

}

func TestRegister(t *testing.T) {

	//Подготовка
	mockStorage := new(mocks.GofermartStorage)
	llog := logger.NewLogger("debug", io.Discard)

	gm, err := NewGofermart(llog, mockStorage)
	assert.NoError(t, err)

	//Ожидания моков
	mockStorage.On("NewUser", "UnExistedUser", mock.Anything).Return(status.StOk)
	mockStorage.On("NewUser", "ExistedUser", mock.Anything).Return(status.StUserAlreadyExists)
	mockStorage.On("NewUser", "InternalError", mock.Anything).Return(status.StGeneralError)

	//Тесты
	st := gm.Register("UnExistedUser", "Password")
	assert.Equal(t, status.StOk, st)

	st = gm.Register("ExistedUser", "Password")
	assert.Equal(t, status.StUserAlreadyExists, st)

	st = gm.Register("InternalError", "DUMP")
	assert.Equal(t, status.StGeneralError, st)

}
