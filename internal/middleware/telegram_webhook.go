package middleware

import (
	"net/http"

	"alpha-hygiene-backend/config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// TelegramWebhookMiddleware проверяет, что запрос пришел от Telegram
func TelegramWebhookMiddleware(cfg *config.Config, log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем секретный токен из заголовка запроса
		signature := c.GetHeader("X-Telegram-Bot-Api-Secret-Token")
		log.Debugf("Received webhook signature: %s", signature)

		// Проверяем наличие заголовка
		if signature == "" {
			log.Warn("Missing X-Telegram-Bot-Api-Secret-Token header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header not found",
			})
			return
		}

		// Проверяем правильность секретного токена
		if signature != cfg.Telegram.WebhookApiSecret {
			log.Warnf("Invalid webhook secret: %s", signature)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid webhook secret",
			})
			return
		}

		// Если проверка пройдена, продолжаем обработку запроса
		log.Debug("Webhook secret is valid")
		c.Next()
	}
}
