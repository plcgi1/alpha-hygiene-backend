package aggregator

import (
	"context"
	"testing"
	"time"

	"alpha-hygiene-backend/config"
	"alpha-hygiene-backend/internal/checker"
	"alpha-hygiene-backend/internal/entity"
	"alpha-hygiene-backend/pkg/logger"

	"github.com/stretchr/testify/assert"
)

func TestCheckWallet(t *testing.T) {
	// Инициализация конфигурации и логирования для тестов
	log, err := logger.New("debug")
	assert.NoError(t, err)

	// Создаем тестовую конфигурацию
	cfg := &config.Config{
		App: struct {
			Port       int    `yaml:"port"`
			LogLevel   string `yaml:"log_level"`
			TimeoutSec int    `yaml:"timeout_sec"`
			RateLimit  struct {
				Enabled  bool `yaml:"enabled"`
				Requests int  `yaml:"requests"`
				Window   int  `yaml:"window_seconds"`
			} `yaml:"rate_limit"`
		}{
			Port:       8080,
			LogLevel:   "debug",
			TimeoutSec: 30,
			RateLimit: struct {
				Enabled  bool `yaml:"enabled"`
				Requests int  `yaml:"requests"`
				Window   int  `yaml:"window_seconds"`
			}{
				Enabled:  false,
				Requests: 100,
				Window:   60,
			},
		},
		Scoring: struct {
			BaseScore float64            `yaml:"base_score"`
			Weights   map[string]float64 `yaml:"weights"`
		}{
			BaseScore: 100,
			Weights: map[string]float64{
				"approvals":   0.4,
				"scam_tokens": 0.2,
				"rug_pulls":   0.2,
				"dead_nft":    0.1,
				"asset_ratio": 0.1,
			},
		},
	}

	// Создаем моки для тестов
	mockFactory := &mockCheckerFactory{}

	// Создаем мок для кэша
	mockCache := &mockCache{}

	service := NewService(cfg, mockFactory, mockCache, log.WithContext(t.Context()))

	// Тестируем проверку кошелька
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	report, err := service.CheckWallet(ctx, "0x742d35Cc6634C0532925a3b88650D7241EfF5cbc", true)

	// Проверяем результаты
	assert.NoError(t, err)
	assert.Equal(t, "0x742d35Cc6634C0532925a3b88650D7241EfF5cbc", report.Address)
	assert.GreaterOrEqual(t, report.Score, 0.0)
	assert.LessOrEqual(t, report.Score, 100.0)
	assert.NotEmpty(t, report.Checks)

	// Проверяем, что все ошибки обработаны
	assert.Empty(t, report.Errors)
}

// mockCheckerFactory - Мок для фабрики проверок
type mockCache struct{}

func (m *mockCache) GetWalletReport(ctx context.Context, address string) (*entity.WalletReport, error) {
	return nil, nil // Возвращаем nil, чтобы не использовать кэш в тестах
}

func (m *mockCache) SetWalletReport(ctx context.Context, address string, report *entity.WalletReport) error {
	return nil
}

func (m *mockCache) GetPayment(ctx context.Context, guid string) (*entity.Payment, error) {
	return nil, nil
}

func (m *mockCache) Close() error {
	return nil
}

type mockCheckerFactory struct{}

func (f *mockCheckerFactory) CreateCheck(t checker.CheckType) checker.IHealthCheck {
	return &mockHealthCheck{t}
}

// mockHealthCheck - Мок для проверки
type mockHealthCheck struct {
	checkType checker.CheckType
}

func (c *mockHealthCheck) Name() string {
	return string(c.checkType)
}

func (c *mockHealthCheck) Execute(ctx context.Context, address string) (*entity.CheckResult, error) {
	return &entity.CheckResult{
		CheckName:    c.Name(),
		RiskFound:    false,
		RiskLevel:    entity.RiskLevelLow,
		ScorePenalty: 0,
		Details:      "Mock check passed",
		RawData:      nil,
	}, nil
}
