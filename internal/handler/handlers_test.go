package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/qesterrx/gofermart/internal/auth"
	"github.com/qesterrx/gofermart/internal/logger"
	"github.com/qesterrx/gofermart/internal/model"
	"github.com/qesterrx/gofermart/internal/status"
	"github.com/qesterrx/gofermart/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupTest(t *testing.T) (*HandlerContainer, *mocks.GofermartService) {
	log := logger.NewLogger("debug", io.Discard)
	mockService := mocks.NewGofermartService(t)

	container := &HandlerContainer{
		log: log,
		gms: mockService,
	}

	return container, mockService
}

// Вспомогательная функция для добавления JWT в контекст
func addUserToContext(r *http.Request, userID int, username string) *http.Request {
	jwtc := &auth.JWTC{
		UserID:   userID,
		Username: username,
	}
	ctx := context.WithValue(r.Context(), "user", jwtc)
	return r.WithContext(ctx)
}

// TestPostUserRegister - тестирование регистрации
func TestPostUserRegister(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*mocks.GofermartService)
		expectedStatus int
		expectedCookie bool
	}{
		{
			name: "Successful registration",
			requestBody: model.AuthUser{
				Login:    "testuser",
				Password: "testpass",
			},
			mockSetup: func(m *mocks.GofermartService) {
				m.On("Register", "testuser", "testpass").Return(status.StOk)
				m.On("Login", "testuser", "testpass").Return("valid.jwt.token", status.StOk)
			},
			expectedStatus: http.StatusOK,
			expectedCookie: true,
		},
		{
			name: "User already exists",
			requestBody: model.AuthUser{
				Login:    "existinguser",
				Password: "pass123",
			},
			mockSetup: func(m *mocks.GofermartService) {
				m.On("Register", "existinguser", "pass123").Return(status.StUserAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			expectedCookie: false,
		},
		{
			name:           "Invalid JSON body",
			requestBody:    "invalid json",
			mockSetup:      func(m *mocks.GofermartService) {},
			expectedStatus: http.StatusBadRequest,
			expectedCookie: false,
		},
		{
			name: "Wrong password after registration",
			requestBody: model.AuthUser{
				Login:    "user",
				Password: "wrong",
			},
			mockSetup: func(m *mocks.GofermartService) {
				m.On("Register", "user", "wrong").Return(status.StOk)
				m.On("Login", "user", "wrong").Return("", status.StUserWrongPassword)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCookie: false,
		},
		{
			name: "General error on register",
			requestBody: model.AuthUser{
				Login:    "user",
				Password: "pass",
			},
			mockSetup: func(m *mocks.GofermartService) {
				m.On("Register", "user", "pass").Return(status.StGeneralError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCookie: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container, mockService := setupTest(t)
			tt.mockSetup(mockService)

			var body []byte
			if data, ok := tt.requestBody.(model.AuthUser); ok {
				body, _ = json.Marshal(data)
			} else {
				body = []byte(tt.requestBody.(string))
			}

			req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			container.PostUserRegister(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedCookie {
				cookies := rr.Result().Cookies()
				assert.NotEmpty(t, cookies)
				found := false
				for _, c := range cookies {
					if c.Name == auth.JWTCookieName {
						found = true
						assert.Equal(t, "valid.jwt.token", c.Value)
						break
					}
				}
				assert.True(t, found, "JWT cookie should be set")
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestPostUserLogin - тестирование логина
func TestPostUserLogin(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func(*mocks.GofermartService)
		expectedStatus int
		expectedCookie bool
	}{
		{
			name: "Successful login",
			requestBody: model.AuthUser{
				Login:    "testuser",
				Password: "testpass",
			},
			mockSetup: func(m *mocks.GofermartService) {
				m.On("Login", "testuser", "testpass").Return("valid.jwt.token", status.StOk)
			},
			expectedStatus: http.StatusOK,
			expectedCookie: true,
		},
		{
			name: "Wrong password",
			requestBody: model.AuthUser{
				Login:    "testuser",
				Password: "wrongpass",
			},
			mockSetup: func(m *mocks.GofermartService) {
				m.On("Login", "testuser", "wrongpass").Return("", status.StUserWrongPassword)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCookie: false,
		},
		{
			name:           "Invalid JSON",
			requestBody:    "invalid",
			mockSetup:      func(m *mocks.GofermartService) {},
			expectedStatus: http.StatusBadRequest,
			expectedCookie: false,
		},
		{
			name: "General error",
			requestBody: model.AuthUser{
				Login:    "user",
				Password: "pass",
			},
			mockSetup: func(m *mocks.GofermartService) {
				m.On("Login", "user", "pass").Return("", status.StGeneralError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCookie: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container, mockService := setupTest(t)
			tt.mockSetup(mockService)

			var body []byte
			if data, ok := tt.requestBody.(model.AuthUser); ok {
				body, _ = json.Marshal(data)
			} else {
				body = []byte(tt.requestBody.(string))
			}

			req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			container.PostUserLogin(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedCookie {
				cookies := rr.Result().Cookies()
				assert.NotEmpty(t, cookies)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestPostUserOrders - тестирование создания заказа
func TestPostUserOrders(t *testing.T) {
	tests := []struct {
		name           string
		userID         int
		username       string
		orderNumber    string
		contentType    string
		mockSetup      func(*mocks.GofermartService)
		expectedStatus int
	}{
		{
			name:        "Successful order creation",
			userID:      1,
			username:    "testuser",
			orderNumber: "12345678903",
			contentType: "text/plain",
			mockSetup: func(m *mocks.GofermartService) {
				m.On("CheckOrderNumber", "12345678903").Return(nil)
				m.On("NewOrder", 1, "12345678903").Return(status.StOk)
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:        "Order already exists",
			userID:      1,
			username:    "testuser",
			orderNumber: "12345678903",
			contentType: "text/plain",
			mockSetup: func(m *mocks.GofermartService) {
				m.On("CheckOrderNumber", "12345678903").Return(nil)
				m.On("NewOrder", 1, "12345678903").Return(status.StOrderDuplicated)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:        "Order belongs to another user",
			userID:      1,
			username:    "testuser",
			orderNumber: "12345678903",
			contentType: "text/plain",
			mockSetup: func(m *mocks.GofermartService) {
				m.On("CheckOrderNumber", "12345678903").Return(nil)
				m.On("NewOrder", 1, "12345678903").Return(status.StOrderAnotherUser)
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:        "Invalid order number format",
			userID:      1,
			username:    "testuser",
			orderNumber: "123",
			contentType: "text/plain",
			mockSetup: func(m *mocks.GofermartService) {
				m.On("CheckOrderNumber", "123").Return(errors.New("invalid order number"))
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "No user in context",
			userID:         1,
			username:       "testuser",
			orderNumber:    "12345678903",
			contentType:    "text/plain",
			mockSetup:      func(m *mocks.GofermartService) {},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container, mockService := setupTest(t)
			tt.mockSetup(mockService)

			req := httptest.NewRequest(http.MethodPost, "/api/user/orders", bytes.NewReader([]byte(tt.orderNumber)))
			req.Header.Set("Content-Type", tt.contentType)

			// Добавляем пользователя в контекст только если тест ожидает его наличие
			if tt.name != "No user in context" {
				req = addUserToContext(req, tt.userID, tt.username)
			}

			rr := httptest.NewRecorder()
			container.PostUserOrders(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// TestGetUserOrders - тестирование получения заказов
func TestGetUserOrders(t *testing.T) {

	sum := float32(11.50)

	testOrders := []model.Order{
		{Order: "12345678903", Status: model.OrderStProcessed, Accrual: &sum},
		{Order: "12345678904", Status: model.OrderStNew},
	}

	tests := []struct {
		name           string
		userID         int
		username       string
		mockSetup      func(*mocks.GofermartService)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:     "Successful get orders",
			userID:   1,
			username: "testuser",
			mockSetup: func(m *mocks.GofermartService) {
				m.On("GetOrders", 1).Return(testOrders, status.StOk)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "No orders found",
			userID:   1,
			username: "testuser",
			mockSetup: func(m *mocks.GofermartService) {
				m.On("GetOrders", 1).Return([]model.Order{}, status.StOk)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "No user in context",
			userID:         1,
			username:       "testuser",
			mockSetup:      func(m *mocks.GofermartService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:     "Service error",
			userID:   1,
			username: "testuser",
			mockSetup: func(m *mocks.GofermartService) {
				m.On("GetOrders", 1).Return([]model.Order{}, status.StGeneralError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container, mockService := setupTest(t)
			tt.mockSetup(mockService)

			req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)

			if tt.name != "No user in context" {
				req = addUserToContext(req, tt.userID, tt.username)
			}

			rr := httptest.NewRecorder()
			container.GetUserOrders(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var orders []model.Order
				err := json.Unmarshal(rr.Body.Bytes(), &orders)
				assert.NoError(t, err)
				assert.Equal(t, len(testOrders), len(orders))
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestGetUserBalance - тестирование получения баланса
func TestGetUserBalance(t *testing.T) {
	testBalance := model.Balance{
		Amount:    500.75,
		Withdrawn: 100.25,
	}

	tests := []struct {
		name            string
		userID          int
		username        string
		mockSetup       func(*mocks.GofermartService)
		expectedStatus  int
		expectedBalance *model.Balance
	}{
		{
			name:     "Successful get balance",
			userID:   1,
			username: "testuser",
			mockSetup: func(m *mocks.GofermartService) {
				m.On("GetBalance", 1).Return(testBalance, status.StOk)
			},
			expectedStatus:  http.StatusOK,
			expectedBalance: &testBalance,
		},
		{
			name:            "No user in context",
			userID:          1,
			username:        "testuser",
			mockSetup:       func(m *mocks.GofermartService) {},
			expectedStatus:  http.StatusUnauthorized,
			expectedBalance: nil,
		},
		{
			name:     "Service error",
			userID:   1,
			username: "testuser",
			mockSetup: func(m *mocks.GofermartService) {
				m.On("GetBalance", 1).Return(model.Balance{}, status.StGeneralError)
			},
			expectedStatus:  http.StatusInternalServerError,
			expectedBalance: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container, mockService := setupTest(t)
			tt.mockSetup(mockService)

			req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)

			if tt.name != "No user in context" {
				req = addUserToContext(req, tt.userID, tt.username)
			}

			rr := httptest.NewRecorder()
			container.GetUserBalance(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedBalance != nil {
				var balance model.Balance
				err := json.Unmarshal(rr.Body.Bytes(), &balance)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBalance.Amount, balance.Amount)
				assert.Equal(t, tt.expectedBalance.Withdrawn, balance.Withdrawn)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestPostUserBalanceWithdraw - тестирование списания бонусов
func TestPostUserBalanceWithdraw(t *testing.T) {
	tests := []struct {
		name           string
		userID         int
		username       string
		withdraw       model.NewWithdraw
		mockSetup      func(*mocks.GofermartService)
		expectedStatus int
	}{
		{
			name:     "Successful withdrawal",
			userID:   1,
			username: "testuser",
			withdraw: model.NewWithdraw{
				Order: "12345678903",
				Sum:   150.75,
			},
			mockSetup: func(m *mocks.GofermartService) {
				m.On("CheckOrderNumber", "12345678903").Return(nil)
				m.On("NewWithdraw", 1, mock.AnythingOfType("*model.NewWithdraw")).Return(status.StOk)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "Insufficient funds",
			userID:   1,
			username: "testuser",
			withdraw: model.NewWithdraw{
				Order: "12345678903",
				Sum:   1000.00,
			},
			mockSetup: func(m *mocks.GofermartService) {
				m.On("CheckOrderNumber", "12345678903").Return(nil)
				m.On("NewWithdraw", 1, mock.AnythingOfType("*model.NewWithdraw")).Return(status.StWithdrawInsufficientFunds)
			},
			expectedStatus: http.StatusPaymentRequired,
		},
		{
			name:     "Invalid order number",
			userID:   1,
			username: "testuser",
			withdraw: model.NewWithdraw{
				Order: "123",
				Sum:   100,
			},
			mockSetup: func(m *mocks.GofermartService) {
				m.On("CheckOrderNumber", "123").Return(errors.New("invalid order number"))
			},
			expectedStatus: http.StatusUnprocessableEntity,
		},
		{
			name:     "No user in context",
			userID:   1,
			username: "testuser",
			withdraw: model.NewWithdraw{
				Order: "12345678903",
				Sum:   100,
			},
			mockSetup: func(m *mocks.GofermartService) {
				// No expectations
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:     "Invalid JSON body",
			userID:   1,
			username: "testuser",
			withdraw: model.NewWithdraw{}, // Will be overridden by invalid JSON
			mockSetup: func(m *mocks.GofermartService) {
				// No expectations
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container, mockService := setupTest(t)
			tt.mockSetup(mockService)

			var body []byte
			var err error

			if tt.name == "Invalid JSON body" {
				body = []byte("invalid json")
			} else {
				body, err = json.Marshal(tt.withdraw)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			if tt.name != "No user in context" {
				req = addUserToContext(req, tt.userID, tt.username)
			}

			rr := httptest.NewRecorder()
			container.PostUserBalanceWithdraw(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			mockService.AssertExpectations(t)
		})
	}
}

// TestGetUserWithdrawals - тестирование получения списка списаний
func TestGetUserWithdrawals(t *testing.T) {

	tm := time.Now()
	testWithdrawals := []model.Withdraw{
		{Order: "12345678903", Sum: 100.50, Uploaded: tm},
		{Order: "12345678904", Sum: 50.25, Uploaded: tm},
	}

	tests := []struct {
		name           string
		userID         int
		username       string
		mockSetup      func(*mocks.GofermartService)
		expectedStatus int
		expectedCount  int
	}{
		{
			name:     "Successful get withdrawals",
			userID:   1,
			username: "testuser",
			mockSetup: func(m *mocks.GofermartService) {
				m.On("GetWithdrawals", 1).Return(testWithdrawals, status.StOk)
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:     "No withdrawals found",
			userID:   1,
			username: "testuser",
			mockSetup: func(m *mocks.GofermartService) {
				m.On("GetWithdrawals", 1).Return([]model.Withdraw{}, status.StOk)
			},
			expectedStatus: http.StatusNoContent,
			expectedCount:  0,
		},
		{
			name:           "No user in context",
			userID:         1,
			username:       "testuser",
			mockSetup:      func(m *mocks.GofermartService) {},
			expectedStatus: http.StatusUnauthorized,
			expectedCount:  0,
		},
		{
			name:     "Service error",
			userID:   1,
			username: "testuser",
			mockSetup: func(m *mocks.GofermartService) {
				m.On("GetWithdrawals", 1).Return([]model.Withdraw{}, status.StGeneralError)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container, mockService := setupTest(t)
			tt.mockSetup(mockService)

			req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)

			if tt.name != "No user in context" {
				req = addUserToContext(req, tt.userID, tt.username)
			}

			rr := httptest.NewRecorder()
			container.GetUserWithdrawals(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusOK {
				var withdrawals []model.Withdraw
				err := json.Unmarshal(rr.Body.Bytes(), &withdrawals)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(withdrawals))
			}

			mockService.AssertExpectations(t)
		})
	}
}
