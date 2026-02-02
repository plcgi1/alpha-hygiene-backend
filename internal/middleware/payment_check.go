package middleware

import (
	"bytes"
	"io"
	"net/http"

	"alpha-hygiene-backend/internal/cache"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// PaymentCheckMiddleware - Middleware для проверки платежа в Redis
func PaymentCheckMiddleware(cache cache.Cache, log *logrus.Logger) gin.HandlerFunc {
	logger := log.WithFields(logrus.Fields{"component": "payment-check-middleware"})

	return func(c *gin.Context) {
		// Сохраняем тело запроса, чтобы оно можно было прочитать снова в обработчике
		bodyBytes, err := c.GetRawData()
		if err != nil {
			logger.Warnf("Failed to read request body: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request format",
			})
			c.Abort()
			return
		}

		// Восстанавливаем тело запроса
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Получаем GUID платежа из запроса
		var request struct {
			GUID string `json:"guid" binding:"omitempty"`
		}

		// Восстанавливаем тело запроса перед парсингом
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		if err := c.ShouldBindJSON(&request); err != nil {
			logger.Warnf("Failed to bind request: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request format",
			})
			c.Abort()
			return
		}

		// Восстанавливаем тело запроса еще раз для обработчика
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Проверяем наличие платежа в Redis только если GUID предоставлен
		if request.GUID != "" {
			payment, err := cache.GetPayment(c.Request.Context(), request.GUID)
			if err != nil {
				logger.Errorf("Failed to get payment from cache: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to check payment",
				})
				c.Abort()
				return
			}

			// Проверяем статус платежа и устанавливаем флаг для обработчика
			if payment != nil {
				logger.Debugf("Payment found: %s, status: %s", request.GUID, payment.Status)
				c.Set("has_valid_payment", true)
				c.Set("payment", payment)
			} else {
				logger.Debugf("Payment not found: %s", request.GUID)
				c.Set("has_valid_payment", false)
			}
		} else {
			// Если GUID не предоставлен, устанавливаем флаг отсутствия платежа
			logger.Debugf("No GUID provided, skipping payment check")
			c.Set("has_valid_payment", false)
		}

		c.Next()
	}
}
