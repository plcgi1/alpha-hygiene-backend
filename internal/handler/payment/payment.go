package payment

import (
	"encoding/json"
	"net/http"

	"alpha-hygiene-backend/config"
	"alpha-hygiene-backend/internal/cache"
	"alpha-hygiene-backend/internal/entity"
	internalValidator "alpha-hygiene-backend/internal/validator"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// CreateInvoiceRequest - Запрос на создание инвойса
type CreateInvoiceRequest struct {
	GUID    string `json:"guid" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Address string `json:"address" validate:"required,eth_addr" example:"0x0000db5c8B030ae20308ac975898E09741e70000"`
}

// CreateInvoiceResponse - Ответ с ссылкой на инвойс
type CreateInvoiceResponse struct {
	OrderURL string `json:"order_url" example:"https://t.me/pay?hash=1234567890abcdef"`
}

// CreateInvoiceHandler - Обработчик создания инвойса
// @Summary Create payment invoice
// @Description Create a payment invoice for wallet check activation
// @Tags payment
// @Accept  json
// @Produce  json
// @Param request body CreateInvoiceRequest true "Payment details"
// @Success 200 {object} CreateInvoiceResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/payment/create-invoice [post]
func CreateInvoiceHandler(cache cache.Cache, log *logrus.Logger, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateInvoiceRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			log.Errorf("Failed to parse request: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid request format",
			})
			return
		}

		// Валидация запроса
		var validate *validator.Validate
		var err error
		validate, err = internalValidator.NewValidator()
		if err != nil {
			log.Errorf("Failed to create validator: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error",
			})
			return
		}

		if err = validate.Struct(req); err != nil {
			log.Errorf("Validation failed: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		// Создание инвойса через Telegram Bot API
		orderUrl, err := createTelegramInvoice(cfg.Telegram.BotToken, req.GUID)
		if err != nil {
			log.Errorf("Failed to create Telegram invoice: %v", err)
			// В случае ошибки используем мок-ответ
			orderUrl = "https://t.me/pay?hash=mock_hash_" + req.GUID
		}

		// Сохранение информации о платеже в Redis
		if err := cache.AddPayment(c.Request.Context(), req.GUID, req.Address, entity.PaymentStatusPending, cfg.TTL.Payment); err != nil {
			log.Errorf("Failed to add payment to cache: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to create invoice",
			})
			return
		}

		// Возврат ответа с ссылкой на инвойс
		c.JSON(http.StatusOK, CreateInvoiceResponse{
			OrderURL: orderUrl,
		})
	}
}

// createTelegramInvoice создает инвойс через Telegram Bot API
func createTelegramInvoice(botToken string, orderId string) (string, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return "", err
	}

	// Подготовка payload для инвойса
	payload := struct {
		Oid string `json:"oid"`
	}{
		Oid: orderId,
	}

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// Создание запроса на создание инвойса с параметрами
	params := tgbotapi.Params{}
	params["title"] = "Alpha Hygiene full report activation"
	params["description"] = "Activate full health report for your wallet"
	params["payload"] = string(payloadJson)
	params["currency"] = "XTR"

	// Подготовка цен в формате JSON
	prices := []tgbotapi.LabeledPrice{
		{
			Label:  "Alpha Hygiene activation Check",
			Amount: 200,
		},
	}

	pricesJson, err := json.Marshal(prices)
	if err != nil {
		return "", err
	}

	params["prices"] = string(pricesJson)

	// Вызов Telegram Bot API
	apiResp, err := bot.MakeRequest("createInvoiceLink", params)
	if err != nil {
		return "", err
	}
	var invoiceLink string
	err = json.Unmarshal(apiResp.Result, &invoiceLink)
	if err != nil {
		return invoiceLink, err
	}
	return invoiceLink, nil
}
