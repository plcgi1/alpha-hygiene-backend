package wallet

import (
	"net/http"

	"alpha-hygiene-backend/internal/aggregator"
	"alpha-hygiene-backend/internal/entity"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

// CheckWalletRequest - Запрос на проверку кошелька
type CheckWalletRequest struct {
	Address string `json:"address" validate:"required,eth_addr" example:"0x0000db5c8B030ae20308ac975898E09741e70000"`
	GUID    string `json:"guid" validate:"omitempty,uuid4" example:"123e4567-e89b-12d3-a456-426614174000"`
}

// CheckWalletResponse - Ответ с результатом проверки кошелька
type CheckWalletResponse struct {
	*entity.WalletReport `json:",inline"`
}

// CheckWalletHandler - Обработчик проверки кошелька
// @Summary Check wallet security
// @Description Check wallet security and get nutrition score
// @Tags wallet
// @Accept  json
// @Produce  json
// @Param request body CheckWalletRequest true "Wallet address to check"
// @Success 200 {object} CheckWalletResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/check [post]
func CheckWalletHandler(service *aggregator.Service, log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CheckWalletRequest

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

		ctx := c.Request.Context()
		hasValidPayment, _ := c.Get("has_valid_payment")
		boolHasValidPayment := false
		if hvp, ok := hasValidPayment.(bool); ok {
			boolHasValidPayment = hvp
		}

		report, err := service.CheckWallet(ctx, req.Address, boolHasValidPayment)
		if err != nil {
			log.Errorf("Check wallet failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to check wallet",
			})
			return
		}

		c.JSON(http.StatusOK, CheckWalletResponse{
			WalletReport: report,
		})
	}
}
