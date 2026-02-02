package payment

import (
	"net/http"

	"alpha-hygiene-backend/internal/cache"
	"alpha-hygiene-backend/internal/entity"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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
func CreateInvoiceHandler(cache cache.Cache, log *logrus.Logger, botApiUrl string) gin.HandlerFunc {
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
		validate := validator.New()

		// Кастомный валидатор для Ethereum адресов
		validate.RegisterValidation("eth_addr", func(fl validator.FieldLevel) bool {
			addr := fl.Field().String()
			if len(addr) != 42 {
				return false
			}
			if addr[:2] != "0x" {
				return false
			}
			// Проверка на наличие только hex символов
			for _, char := range addr[2:] {
				if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
					return false
				}
			}
			return true
		})

		if err := validate.Struct(req); err != nil {
			log.Errorf("Validation failed: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		// Создание инвойса через Telegram Bot API
		// TODO: Реализовать реальный запрос к Telegram Bot API
		// Временно возвращаем мок-ответ
		orderUrl := "https://t.me/pay?hash=mock_hash_" + req.GUID

		// Сохранение информации о платеже в Redis
		if err := cache.AddPayment(c.Request.Context(), req.GUID, req.Address, entity.PaymentStatusPending); err != nil {
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
