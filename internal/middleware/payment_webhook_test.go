package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"alpha-hygiene-backend/config"
	"alpha-hygiene-backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestPaymentWebhookMiddleware_ValidSignature(t *testing.T) {
	// Инициализация
	log, err := logger.New("debug")
	assert.NoError(t, err)

	cfg := &config.Config{
		Payment: struct {
			WebhookSecret string `yaml:"webhook_secret"`
		}{
			WebhookSecret: "test-payment-secret",
		},
	}

	// Создаем тестовый роутер с middleware
	r := gin.Default()
	r.Use(PaymentWebhookMiddleware(cfg, log))
	r.POST("/webhook", func(c *gin.Context) {
		c.String(http.StatusOK, "Success")
	})

	// Создаем тестовый запрос с правильным сигнатурой
	req, err := http.NewRequest("POST", "/webhook", nil)
	assert.NoError(t, err)
	req.Header.Set("X-Payment-Webhook-Secret", "test-payment-secret")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверка результата
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Success", w.Body.String())
}

func TestPaymentWebhookMiddleware_MissingSignature(t *testing.T) {
	// Инициализация
	log, err := logger.New("debug")
	assert.NoError(t, err)

	cfg := &config.Config{
		Payment: struct {
			WebhookSecret string `yaml:"webhook_secret"`
		}{
			WebhookSecret: "test-payment-secret",
		},
	}

	// Создаем тестовый роутер с middleware
	r := gin.Default()
	r.Use(PaymentWebhookMiddleware(cfg, log))
	r.POST("/webhook", func(c *gin.Context) {
		c.String(http.StatusOK, "Success")
	})

	// Создаем тестовый запрос без сигнатуры
	req, err := http.NewRequest("POST", "/webhook", nil)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверка результата
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "PaymentWebhookMiddleware.Authorization header not found")
}

func TestPaymentWebhookMiddleware_InvalidSignature(t *testing.T) {
	// Инициализация
	log, err := logger.New("debug")
	assert.NoError(t, err)

	cfg := &config.Config{
		Payment: struct {
			WebhookSecret string `yaml:"webhook_secret"`
		}{
			WebhookSecret: "test-payment-secret",
		},
	}

	// Создаем тестовый роутер с middleware
	r := gin.Default()
	r.Use(PaymentWebhookMiddleware(cfg, log))
	r.POST("/webhook", func(c *gin.Context) {
		c.String(http.StatusOK, "Success")
	})

	// Создаем тестовый запрос с неправильной сигнатурой
	req, err := http.NewRequest("POST", "/webhook", nil)
	assert.NoError(t, err)
	req.Header.Set("X-Payment-Webhook-Secret", "wrong-secret")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверка результата
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "PaymentWebhookMiddleware.Invalid webhook secret")
}
