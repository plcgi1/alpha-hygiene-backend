package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"alpha-hygiene-backend/config"

	"github.com/sirupsen/logrus"
)

// EtherscanClient - Клиент для Etherscan API
type EtherscanClient struct {
	*BaseClient
	apiKey  string
	baseURL string
}

// NewEtherscanClient - Создает новый клиент Etherscan
func NewEtherscanClient(cfg *config.Config, log *logrus.Entry) *EtherscanClient {
	baseURL := cfg.Etherscan.URL
	if baseURL == "" {
		baseURL = "https://api.etherscan.io"
	}
	timeout := 10 * time.Second
	if cfg.App.TimeoutSec > 0 {
		timeout = time.Duration(cfg.App.TimeoutSec) * time.Second
	}
	base := NewBaseClient(timeout, log, "etherscan")
	return &EtherscanClient{
		BaseClient: base,
		apiKey:     cfg.Etherscan.ApiKey,
		baseURL:    baseURL,
	}
}

// TokenTransaction - Структура для транзакции с токеном
type TokenTransaction struct {
	BlockNumber       string `json:"blockNumber"`
	TimeStamp         string `json:"timeStamp"`
	Hash              string `json:"hash"`
	Nonce             string `json:"nonce"`
	BlockHash         string `json:"blockHash"`
	From              string `json:"from"`
	ContractAddress   string `json:"contractAddress"`
	To                string `json:"to"`
	Value             string `json:"value"`
	TokenName         string `json:"tokenName"`
	TokenSymbol       string `json:"tokenSymbol"`
	TokenDecimal      string `json:"tokenDecimal"`
	TransactionIndex  string `json:"transactionIndex"`
	Gas               string `json:"gas"`
	GasPrice          string `json:"gasPrice"`
	GasUsed           string `json:"gasUsed"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	Input             string `json:"input"`
	Confirmations     string `json:"confirmations"`
}

// GetERC20Tokens - Получает список ERC20 токенов для адреса
func (c *EtherscanClient) GetERC20Tokens(ctx context.Context, address string) ([]*TokenBalance, error) {
	// NOTE: Этот метод использует неправильный endpoint!
	// Для получения токенов на кошельке нужно использовать Moralis API
	c.log.Warnf("Etherscan API does not support getting all ERC20 tokens for address. Use Moralis API instead.")
	return nil, fmt.Errorf("Etherscan API does not support getting all ERC20 tokens for address")
}

// GetETHBalance - Получает баланс ETH для адреса
func (c *EtherscanClient) GetETHBalance(ctx context.Context, address string) (float64, error) {
	params := url.Values{}
	params.Set("module", "account")
	params.Set("action", "balance")
	params.Set("address", address)
	params.Set("tag", "latest")
	params.Set("apikey", c.apiKey)

	urlStr := fmt.Sprintf("%s/api?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return 0, err
	}

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  string `json:"result"`
	}

	if err := c.DoRequest(ctx, req, &result); err != nil {
		return 0, err
	}

	if result.Status != "1" {
		c.log.Errorf("Etherscan API error: %s", result.Message)
		return 0, fmt.Errorf("Etherscan API error: %s", result.Message)
	}

	wei, err := strconv.ParseUint(result.Result, 10, 64)
	if err != nil {
		c.log.Errorf("Failed to parse balance: %v", err)
		return 0, err
	}

	eth := float64(wei) / 1e18
	return eth, nil
}

// GetInternalTransactions - Получает внутренние транзакции для адреса
func (c *EtherscanClient) GetInternalTransactions(ctx context.Context, address string, startBlock, endBlock int) ([]map[string]interface{}, error) {
	params := url.Values{}
	params.Set("module", "account")
	params.Set("action", "txlistinternal")
	params.Set("address", address)
	params.Set("startblock", strconv.Itoa(startBlock))
	params.Set("endblock", strconv.Itoa(endBlock))
	params.Set("page", "1")
	params.Set("offset", "100")
	params.Set("sort", "asc")
	params.Set("apikey", c.apiKey)

	urlStr := fmt.Sprintf("%s/api?%s", c.baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Status  string                   `json:"status"`
		Message string                   `json:"message"`
		Result  []map[string]interface{} `json:"result"`
	}

	if err := c.DoRequest(ctx, req, &result); err != nil {
		return nil, err
	}

	if result.Status != "1" {
		c.log.Errorf("Etherscan API error: %s", result.Message)
		return nil, fmt.Errorf("Etherscan API error: %s", result.Message)
	}

	return result.Result, nil
}
