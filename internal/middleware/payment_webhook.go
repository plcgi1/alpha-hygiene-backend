package middleware

import (
	"net/http"

	"alpha-hygiene-backend/config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// PaymentWebhookMiddleware проверяет, что запрос пришел от платежного сервиса
func PaymentWebhookMiddleware(cfg *config.Config, log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем секретный токен из заголовка запроса
		signature := c.GetHeader("X-Payment-Webhook-Secret")
		log.Debugf("Received payment webhook signature: %s", signature)

		// Проверяем наличие заголовка
		if signature == "" {
			log.Warn("Missing X-Payment-Webhook-Secret header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "PaymentWebhookMiddleware.Authorization header not found",
			})
			return
		}

		// Проверяем правильность секретного токена
		if signature != cfg.Payment.WebhookSecret {
			log.Warnf("Invalid payment webhook secret: %s", signature)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "PaymentWebhookMiddleware.Invalid webhook secret",
			})
			return
		}

		// Если проверка пройдена, продолжаем обработку запроса
		log.Debug("Payment webhook secret is valid")
		c.Next()
	}
}
