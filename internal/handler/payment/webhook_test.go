package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"alpha-hygiene-backend/config"
	"alpha-hygiene-backend/internal/entity"
	"alpha-hygiene-backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCache - Мок для кэша
type MockCache struct {
	mock.Mock
}

func (m *MockCache) GetWalletReport(ctx context.Context, address string) (*entity.WalletReport, error) {
	args := m.Called(ctx, address)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.WalletReport), args.Error(1)
}

func (m *MockCache) SetWalletReport(ctx context.Context, address string, report *entity.WalletReport, ttl time.Duration) error {
	args := m.Called(ctx, address, report)
	return args.Error(0)
}

func (m *MockCache) GetPayment(ctx context.Context, guid string) (*entity.Payment, error) {
	args := m.Called(ctx, guid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Payment), args.Error(1)
}

func (m *MockCache) AddPayment(ctx context.Context, guid string, address string, status entity.PaymentStatus, ttl time.Duration, username string, tgId int64) error {
	args := m.Called(ctx, guid, address, status, ttl, username, tgId)
	return args.Error(0)
}

func (m *MockCache) ProcessPaymentSuccess(ctx context.Context, userID string, guid string, reportData *entity.WalletReport) error {
	args := m.Called(ctx, userID, guid, reportData)
	return args.Error(0)
}

func (m *MockCache) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestWebhookHandler_PreCheckoutQuery(t *testing.T) {
	// Инициализация
	log, err := logger.New("debug")
	assert.NoError(t, err)

	cfg := &config.Config{
		Payment: struct {
			WebhookSecret string `yaml:"webhook_secret"`
		}{
			WebhookSecret: "test-payment-secret",
		},
		Telegram: struct {
			BotToken         string `yaml:"bot_token"`
			WebhookApiSecret string `yaml:"webhook_secret"`
			OneTimePrice     int64  `yaml:"one_time_price"`
		}{
			WebhookApiSecret: "test-secret",
		},
	}

	mockCache := new(MockCache)
	handler := WebhookHandler(mockCache, log, cfg)

	// Создаем тестовый запрос с pre-checkout query
	reqJson := `{
		"update_id": 12345,
		"message": {
			"message_id": 1,
			"from": {
				"id": 123
			},
			"chat": {
				"id": 456,
				"type": "private"
			},
			"date": 1234567890,
			"pre_checkout_query": {
				"id": "precheck_123"
			}
		}
	}`

	// Создаем тестовый запрос и ответ
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, err := http.NewRequest("POST", "/api/payment/webhook", bytes.NewBufferString(reqJson))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Payment-Webhook-Secret", "test-payment-secret")
	c.Request = req

	// Вызываем обработчик
	handler(c)

	// Проверяем результаты
	assert.Equal(t, http.StatusOK, w.Code)

	var response WebhookResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.OK)
}

func TestWebhookHandler_NoSuccessfulPayment(t *testing.T) {
	// Инициализация
	log, err := logger.New("debug")
	assert.NoError(t, err)

	cfg := &config.Config{
		Payment: struct {
			WebhookSecret string `yaml:"webhook_secret"`
		}{
			WebhookSecret: "test-payment-secret",
		},
		Telegram: struct {
			BotToken         string `yaml:"bot_token"`
			WebhookApiSecret string `yaml:"webhook_secret"`
			OneTimePrice     int64  `yaml:"one_time_price"`
		}{
			WebhookApiSecret: "test-secret",
		},
	}

	mockCache := new(MockCache)
	handler := WebhookHandler(mockCache, log, cfg)

	// Создаем тестовый запрос без успешного платежа
	reqJson := `{
		"update_id": 12345,
		"message": {
			"message_id": 1,
			"from": {
				"id": 123
			},
			"chat": {
				"id": 456,
				"type": "private"
			},
			"date": 1234567890
		}
	}`

	// Создаем тестовый запрос и ответ
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, err := http.NewRequest("POST", "/api/payment/webhook", bytes.NewBufferString(reqJson))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Payment-Webhook-Secret", "test-payment-secret")
	c.Request = req

	// Вызываем обработчик
	handler(c)

	// Проверяем результаты
	assert.Equal(t, http.StatusOK, w.Code)

	var response WebhookResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.OK)
}

func TestWebhookHandler_SuccessfulPayment(t *testing.T) {
	// Инициализация
	log, err := logger.New("debug")
	assert.NoError(t, err)

	cfg := &config.Config{}
	cfg.Telegram.BotToken = "test_token"
	cfg.Telegram.OneTimePrice = 300
	cfg.Telegram.WebhookApiSecret = "WebhookApiSecret"
	cfg.Payment.WebhookSecret = "test-payment-secret"

	mockCache := new(MockCache)
	// Устанавливаем ожидания
	mockCache.On("AddPayment", mock.Anything, "test_guid", "", entity.PaymentStatusPaid, mock.Anything, "", int64(0)).Return(nil)

	handler := WebhookHandler(mockCache, log, cfg)

	// Создаем тестовый запрос с успешным платежом
	reqJson := `{
		"update_id": 12345,
		"message": {
			"message_id": 1,
			"from": {
				"id": 123
			},
			"chat": {
				"id": 456,
				"type": "private"
			},
			"date": 1234567890,
			"successful_payment": {
				"currency": "XTR",
				"total_amount": 300,
				"invoice_payload": "{\"oid\":\"test_guid\"}",
				"telegram_payment_charge_id": "charge_123",
				"provider_payment_charge_id": "provider_123"
			}
		}
	}`

	// Создаем тестовый запрос и ответ
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, err := http.NewRequest("POST", "/api/payment/webhook", bytes.NewBufferString(reqJson))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Payment-Webhook-Secret", "test-payment-secret")
	c.Request = req

	// Вызываем обработчик
	handler(c)

	// Проверяем результаты
	assert.Equal(t, http.StatusOK, w.Code)

	// var response WebhookResponse
	// err = json.Unmarshal(w.Body.Bytes(), &response)
	// assert.NoError(t, err)
	// assert.True(t, response.OK)

	// // Проверяем, что ожидания выполнены
	// mockCache.AssertExpectations(t)
}

func TestWebhookHandler_InvalidAmount(t *testing.T) {
	// Инициализация
	log, err := logger.New("debug")
	assert.NoError(t, err)

	cfg := &config.Config{
		Payment: struct {
			WebhookSecret string `yaml:"webhook_secret"`
		}{
			WebhookSecret: "test-payment-secret",
		},
		Telegram: struct {
			BotToken         string `yaml:"bot_token"`
			WebhookApiSecret string `yaml:"webhook_secret"`
			OneTimePrice     int64  `yaml:"one_time_price"`
		}{
			WebhookApiSecret: "test-secret",
			OneTimePrice:     200,
		},
	}

	mockCache := new(MockCache)
	// Устанавливаем ожидания
	mockCache.On("GetPayment", mock.Anything, "test_guid").Return(nil, nil)

	handler := WebhookHandler(mockCache, log, cfg)

	// Создаем тестовый запрос с некорректной суммой
	reqJson := `{
		"update_id": 12345,
		"message": {
			"message_id": 1,
			"from": {
				"id": 123
			},
			"chat": {
				"id": 456,
				"type": "private"
			},
			"date": 1234567890,
			"successful_payment": {
				"currency": "XTR",
				"total_amount": 100,
				"invoice_payload": "{\"oid\":\"test_guid\"}",
				"telegram_payment_charge_id": "charge_123",
				"provider_payment_charge_id": "provider_123"
			}
		}
	}`

	// Создаем тестовый запрос и ответ
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, err := http.NewRequest("POST", "/api/payment/webhook", bytes.NewBufferString(reqJson))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Payment-Webhook-Secret", "test-payment-secret")
	c.Request = req

	// Вызываем обработчик
	handler(c)

	// Проверяем результаты
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "incorrect payment amount")
}

func TestWebhookHandler_InvalidPayload(t *testing.T) {
	// Инициализация
	log, err := logger.New("debug")
	assert.NoError(t, err)

	cfg := &config.Config{
		Payment: struct {
			WebhookSecret string `yaml:"webhook_secret"`
		}{
			WebhookSecret: "test-payment-secret",
		},
		Telegram: struct {
			BotToken         string `yaml:"bot_token"`
			WebhookApiSecret string `yaml:"webhook_secret"`
			OneTimePrice     int64  `yaml:"one_time_price"`
		}{
			WebhookApiSecret: "test-secret",
		},
	}

	mockCache := new(MockCache)
	handler := WebhookHandler(mockCache, log, cfg)

	// Создаем тестовый запрос с некорректным payload
	reqJson := `{
		"update_id": 12345,
		"message": {
			"message_id": 1,
			"from": {
				"id": 123
			},
			"chat": {
				"id": 456,
				"type": "private"
			},
			"date": 1234567890,
			"successful_payment": {
				"currency": "XTR",
				"total_amount": 200,
				"invoice_payload": "invalid_json",
				"telegram_payment_charge_id": "charge_123",
				"provider_payment_charge_id": "provider_123"
			}
		}
	}`

	// Создаем тестовый запрос и ответ
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, err := http.NewRequest("POST", "/api/payment/webhook", bytes.NewBufferString(reqJson))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Payment-Webhook-Secret", "test-payment-secret")
	c.Request = req

	// Вызываем обработчик
	handler(c)

	// Проверяем результаты
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "invalid invoice payload")
}
