package payment

import (
	"encoding/json"
	"net/http"

	"alpha-hygiene-backend/config"
	"alpha-hygiene-backend/internal/cache"
	"alpha-hygiene-backend/internal/entity"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// WebhookRequest - Запрос от Telegram с информацией о платеже
type WebhookRequest struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID int `json:"id"`
		} `json:"from"`
		Chat struct {
			ID   int    `json:"id"`
			Type string `json:"type"`
		} `json:"chat"`
		Date             int `json:"date"`
		PreCheckoutQuery *struct {
			ID string `json:"id"`
		} `json:"pre_checkout_query"`
		SuccessfulPayment *struct {
			Currency                string `json:"currency"`
			TotalAmount             int64  `json:"total_amount"`
			InvoicePayload          string `json:"invoice_payload"`
			TelegramPaymentChargeID string `json:"telegram_payment_charge_id"`
			ProviderPaymentChargeID string `json:"provider_payment_charge_id"`
		} `json:"successful_payment"`
	} `json:"message"`
}

// WebhookResponse - Ответ на запрос от Telegram
type WebhookResponse struct {
	OK bool `json:"ok"`
}

// WebhookHandler - Обработчик вебхука от Telegram
// @Summary Handle Telegram webhook
// @Description Handle payment notifications from Telegram
// @Tags payment
// @Accept  json
// @Produce  json
// @Param request body WebhookRequest true "Webhook data from Telegram"
// @Success 200 {object} WebhookResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/payment/webhook [post]
func WebhookHandler(cache cache.Cache, log *logrus.Logger, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req WebhookRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			log.Errorf("Failed to parse webhook request: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid request format",
			})
			return
		}

		// Обработка подтверждения готовности к платежу
		if req.Message.PreCheckoutQuery != nil {
			log.Infof("Processing pre-checkout query: %s", req.Message.PreCheckoutQuery.ID)
			c.JSON(http.StatusOK, WebhookResponse{OK: true})
			return
		}

		// Проверка наличия информации о успешном платеже
		if req.Message.SuccessfulPayment == nil {
			log.Debugf("No successful payment information in update")
			c.JSON(http.StatusOK, WebhookResponse{OK: true})
			return
		}

		success := req.Message.SuccessfulPayment
		log.Infof("Processing successful payment: %s", success.TelegramPaymentChargeID)

		// Разбор payload инвойса
		var payload struct {
			Oid string `json:"oid"`
		}
		if err := json.Unmarshal([]byte(success.InvoicePayload), &payload); err != nil {
			log.Errorf("Failed to parse invoice payload: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid invoice payload",
			})
			return
		}

		// Проверка суммы платежа (ожидается 200 XTR = 200 units)
		if success.TotalAmount != cfg.Telegram.OneTimePrice {
			log.Errorf("Incorrect payment amount. Expected: %d, got: %d", cfg.Telegram.OneTimePrice, success.TotalAmount)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "incorrect payment amount",
			})
			return
		}

		// Обновление статуса платежа в Redis
		if err := cache.AddPayment(c.Request.Context(), payload.Oid, "", entity.PaymentStatusPaid, cfg.TTL.Payment); err != nil {
			log.Errorf("Failed to update payment status: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to process payment",
			})
			return
		}

		log.Infof("Payment processed successfully for GUID: %s", payload.Oid)
		c.JSON(http.StatusOK, WebhookResponse{OK: true})
	}
}
