package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"alpha-hygiene-backend/config"
	_ "alpha-hygiene-backend/docs"
	"alpha-hygiene-backend/internal/aggregator"
	"alpha-hygiene-backend/internal/cache"
	"alpha-hygiene-backend/internal/checker"
	"alpha-hygiene-backend/internal/handler/wallet"
	"alpha-hygiene-backend/internal/middleware"
	"alpha-hygiene-backend/internal/provider"
	"alpha-hygiene-backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Wallet Nutrition Score API
// @version 1.0
// @description API for checking wallet security and calculating nutrition score
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
func main() {
	// Инициализация конфигурации
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// Инициализация логирования
	log, err := logger.New(cfg.App.LogLevel)
	if err != nil {
		panic(err)
	}

	log.Info("Application starting up")

	// Инициализация провайдеров
	goplusClient := provider.NewGoPlusClient(cfg, log.WithContext(&gin.Context{}))
	etherscanClient := provider.NewEtherscanClient(cfg, log.WithContext(&gin.Context{}))
	alchemyClient := provider.NewAlchemyClient(cfg, log.WithContext(&gin.Context{}))

	// Инициализация Redis кэша
	var redisCache cache.Cache
	redisCache, err = cache.NewRedisCache(cfg, log.WithContext(&gin.Context{}))
	if err != nil {
		log.Warnf("Failed to initialize Redis cache: %v. Cache will not be available.", err)
	}

	// Инициализация фабрики проверок
	checkerFactory := checker.NewFactory(cfg, goplusClient, etherscanClient, alchemyClient, log.WithContext(&gin.Context{}))

	// Инициализация агрегатора
	aggregatorService := aggregator.NewService(cfg, checkerFactory, redisCache, log.WithContext(&gin.Context{}))

	// Настройка Gin
	if cfg.App.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Настраиваем Gin для использования JSON логов вместо текстовых
	gin.DefaultWriter = os.Stdout
	r := gin.New()
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logEntry := map[string]interface{}{
			"time":    param.TimeStamp.Format(time.RFC3339),
			"method":  param.Method,
			"path":    param.Path,
			"status":  param.StatusCode,
			"latency": param.Latency.String(),
			"error":   param.ErrorMessage,
		}
		jsonStr, err := json.Marshal(logEntry)
		if err != nil {
			return fmt.Sprintf("Error formatting log: %v\n", err)
		}
		return string(jsonStr) + "\n"
	}))
	r.Use(gin.Recovery())

	// Rate Limiter middleware
	if cfg.App.RateLimit.Enabled {
		log.Infof("Rate limiting enabled: %d requests per %d seconds",
			cfg.App.RateLimit.Requests, cfg.App.RateLimit.Window)
		rl := middleware.NewRateLimiter(cfg, log)
		r.Use(rl.RateLimitMiddleware())

		// Periodically clear expired rate limit entries (every window duration)
		go func() {
			ticker := time.NewTicker(rl.Window)
			defer ticker.Stop()

			for range ticker.C {
				rl.ClearExpiredEntries()
			}
		}()
	}

	// Swagger endpoint
	// url := ginSwagger.URL("http://localhost:8080/swagger/doc.json")
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Обработчики
	r.GET("/health", healthCheckHandler(log))
	r.POST("/api/check", middleware.PaymentCheckMiddleware(redisCache, log), wallet.CheckWalletHandler(aggregatorService, log))

	// Запуск сервера
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.App.Port),
		Handler: r,
	}

	// Graceful shutdown
	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, syscall.SIGINT, syscall.SIGTERM)
		<-sigint

		log.Info("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Errorf("Server shutdown failed: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Infof("Server starting on port %d", cfg.App.Port)
	log.Infof("Swagger documentation available at http://localhost:%d/swagger/index.html", cfg.App.Port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server startup failed: %v", err)
	}

	<-idleConnsClosed
	log.Info("Server stopped")
}

// healthCheckHandler - Проверка статуса сервиса
// @Summary Health check
// @Description Check if the service is running
// @Tags health
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]string
// @Router /health [get]
func healthCheckHandler(log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, map[string]string{
			"status": "healthy",
		})
	}
}
