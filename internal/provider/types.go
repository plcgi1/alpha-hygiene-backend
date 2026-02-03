package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// TokenBalance - Общая структура для баланса токена
type TokenBalance struct {
	Account         string `json:"account"`
	TokenID         string `json:"tokenID"`
	ContractAddress string `json:"contractAddress"`
	TokenName       string `json:"tokenName"`
	TokenSymbol     string `json:"tokenSymbol"`
	TokenDecimal    string `json:"tokenDecimal"`
	Balance         string `json:"balance"`
}

// BaseClient - Базовая структура для всех клиентов провайдеров
type BaseClient struct {
	client *http.Client
	log    *logrus.Entry
}

// NewBaseClient - Создает новый базовый клиент
func NewBaseClient(timeout time.Duration, log *logrus.Entry, component string) *BaseClient {
	return &BaseClient{
		client: &http.Client{
			Timeout: timeout,
		},
		log: log.WithFields(logrus.Fields{"component": component}),
	}
}

// DoRequest - Общий метод для выполнения HTTP запросов
func (c *BaseClient) DoRequest(ctx context.Context, req *http.Request, target interface{}) error {
	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Errorf("API request failed: %v", err)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Errorf("Failed to read response body: %v", err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		c.log.Errorf("API error: %s", string(body))
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, target); err != nil {
		c.log.Errorf("Failed to unmarshal response: %v", err)
		return err
	}

	return nil
}

// Close - Закрывает клиент (по умолчанию не требует действий)
func (c *BaseClient) Close() error {
	return nil
}
