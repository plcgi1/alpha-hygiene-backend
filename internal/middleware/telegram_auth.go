package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"alpha-hygiene-backend/config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// TelegramUser represents the Telegram user data structure
type TelegramUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Username     string `json:"username"`
	LanguageCode string `json:"language_code"`
	IsPremium    bool   `json:"is_premium"`
}

// TelegramAuthMiddleware authenticates Telegram Mini App requests
func TelegramAuthMiddleware(cfg *config.Config, log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication if configured
		if cfg.App.AuthGuard.Skip {
			log.Debug("Telegram auth guard is skipped (configured)")
			c.Next()
			return
		}

		// Get authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Warn("Authorization header not found")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header not found",
			})
			return
		}

		// Expected format: "twa <initData>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "twa" {
			log.Warnf("Invalid authorization type: %s", authHeader)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization type. Expected 'twa <initData>'",
			})
			return
		}

		initData := parts[1]

		// Validate init data
		if !validateTelegramInitData(initData, cfg.Telegram.BotToken, cfg.App.AuthGuard.TTL) {
			log.Warn("Invalid Telegram init data")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid Telegram init data",
			})
			return
		}

		// Parse user data
		user, err := parseTelegramUser(initData)
		if err != nil {
			log.Errorf("Failed to parse Telegram user data: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Failed to parse user data",
			})
			return
		}

		// Store user in context for handlers to access
		c.Set("telegram_user", user)
		log.Debugf("Authenticated Telegram user: %s (@%s)", user.FirstName, user.Username)

		// Continue to the next handler
		c.Next()
	}
}

// validateTelegramInitData validates the Telegram Web App init data
func validateTelegramInitData(initData string, botToken string, ttl int) bool {
	// Parse URL parameters
	params, err := url.ParseQuery(initData)
	if err != nil {
		return false
	}

	// Check required fields
	hash := params.Get("hash")
	authDateStr := params.Get("auth_date")
	if hash == "" || authDateStr == "" {
		return false
	}

	// Check expiration
	authDate, err := strconv.ParseInt(authDateStr, 10, 64)
	if err != nil {
		return false
	}

	now := time.Now().Unix()
	if now-authDate > int64(ttl) {
		return false
	}

	// Prepare data check string
	delete(params, "hash")
	var pairs []string
	for key, values := range params {
		if len(values) > 0 {
			pairs = append(pairs, key+"="+values[0])
		}
	}

	sort.Strings(pairs)
	dataCheckString := strings.Join(pairs, "\n")

	// Create secret key
	h := hmac.New(sha256.New, []byte("WebAppData"))
	h.Write([]byte(botToken))
	secretKey := h.Sum(nil)

	// Calculate HMAC-SHA256 of data check string
	hmacObj := hmac.New(sha256.New, secretKey)
	hmacObj.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(hmacObj.Sum(nil))

	// Verify hash
	return hmac.Equal([]byte(calculatedHash), []byte(hash))
}

// parseTelegramUser parses the Telegram user data from initData
func parseTelegramUser(initData string) (*TelegramUser, error) {
	params, err := url.ParseQuery(initData)
	if err != nil {
		return nil, err
	}

	userStr := params.Get("user")
	if userStr == "" {
		return nil, errUserNotFound
	}

	var user TelegramUser
	err = json.Unmarshal([]byte(userStr), &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// errUserNotFound is a custom error for when user data is not found
var errUserNotFound = &struct{ error }{error: nil}
