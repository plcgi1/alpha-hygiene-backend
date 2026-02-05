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

func TestTelegramWebhookMiddleware_ValidSignature(t *testing.T) {
	// Инициализация
	log, err := logger.New("debug")
	assert.NoError(t, err)

	cfg := &config.Config{
		Telegram: struct {
			BotToken         string `yaml:"bot_token"`
			WebhookApiSecret string `yaml:"webhook_secret"`
			OneTimePrice     int64  `yaml:"one_time_price"`
		}{
			WebhookApiSecret: "test-secret",
		},
	}

	// Создаем тестовый роутер с middleware
	r := gin.Default()
	r.Use(TelegramWebhookMiddleware(cfg, log))
	r.POST("/webhook", func(c *gin.Context) {
		c.String(http.StatusOK, "Success")
	})

	// Создаем тестовый запрос с правильным сигнатурой
	req, err := http.NewRequest("POST", "/webhook", nil)
	assert.NoError(t, err)
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "test-secret")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверка результата
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Success", w.Body.String())
}

func TestTelegramWebhookMiddleware_MissingSignature(t *testing.T) {
	// Инициализация
	log, err := logger.New("debug")
	assert.NoError(t, err)

	cfg := &config.Config{
		Telegram: struct {
			BotToken         string `yaml:"bot_token"`
			WebhookApiSecret string `yaml:"webhook_secret"`
			OneTimePrice     int64  `yaml:"one_time_price"`
		}{
			WebhookApiSecret: "test-secret",
		},
	}

	// Создаем тестовый роутер с middleware
	r := gin.Default()
	r.Use(TelegramWebhookMiddleware(cfg, log))
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
	assert.Contains(t, w.Body.String(), "Authorization header not found")
}

func TestTelegramWebhookMiddleware_InvalidSignature(t *testing.T) {
	// Инициализация
	log, err := logger.New("debug")
	assert.NoError(t, err)

	cfg := &config.Config{
		Telegram: struct {
			BotToken         string `yaml:"bot_token"`
			WebhookApiSecret string `yaml:"webhook_secret"`
			OneTimePrice     int64  `yaml:"one_time_price"`
		}{
			WebhookApiSecret: "test-secret",
		},
	}

	// Создаем тестовый роутер с middleware
	r := gin.Default()
	r.Use(TelegramWebhookMiddleware(cfg, log))
	r.POST("/webhook", func(c *gin.Context) {
		c.String(http.StatusOK, "Success")
	})

	// Создаем тестовый запрос с неправильной сигнатурой
	req, err := http.NewRequest("POST", "/webhook", nil)
	assert.NoError(t, err)
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong-secret")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Проверка результата
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid webhook secret")
}
