package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"alpha-hygiene-backend/config"

	"github.com/sirupsen/logrus"
)

// GoPlusClient - Клиент для GoPlus Security API
type GoPlusClient struct {
	*BaseClient
	apiKey    string
	apiSecret string
}

// NewGoPlusClient - Создает новый клиент GoPlus
func NewGoPlusClient(cfg *config.Config, log *logrus.Entry) *GoPlusClient {
	timeout := time.Duration(cfg.App.TimeoutSec) * time.Second
	if cfg.App.TimeoutSec <= 0 {
		timeout = 10 * time.Second // Default timeout
	}
	base := NewBaseClient(timeout, log, "goplus")
	return &GoPlusClient{
		BaseClient: base,
		apiKey:     cfg.GoPlus.ApiKey,
		apiSecret:  cfg.GoPlus.ApiSecret,
	}
}

// GoPlusResponse — корневая структура ответа
type TokenApprovalResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Result  []TokenApproval `json:"result"`
}

// TokenApproval — информация о токене и его аппрувах
type TokenApproval struct {
	TokenAddress      string            `json:"token_address"`
	ChainID           string            `json:"chain_id"`
	TokenName         string            `json:"token_name"`
	TokenSymbol       string            `json:"token_symbol"`
	Decimals          int               `json:"decimals"`
	Balance           string            `json:"balance"` // Строка, т.к. может быть BigInt
	IsOpenSource      int               `json:"is_open_source"`
	MaliciousAddress  int               `json:"malicious_address"`
	MaliciousBehavior []interface{}     `json:"malicious_behavior"` // interface{}, так как массив пустой, тип неизвестен (обычно строки)
	ApprovedList      []ApprovedSpender `json:"approved_list"`
}

// ApprovedSpender — кому выдано разрешение
type ApprovedSpender struct {
	ApprovedContract    string      `json:"approved_contract"`
	ApprovedAmount      string      `json:"approved_amount"` // "Unlimited" или число в строке
	ApprovedTime        int64       `json:"approved_time"`
	InitialApprovalTime int64       `json:"initial_approval_time"`
	InitialApprovalHash string      `json:"initial_approval_hash"`
	Hash                string      `json:"hash"`
	AddressInfo         AddressInfo `json:"address_info"`
}

// AddressInfo — информация о контракте-спендере
type AddressInfo struct {
	ContractName      string        `json:"contract_name"`
	Tag               *string       `json:"tag"` // *string, так как в JSON пришел null
	CreatorAddress    string        `json:"creator_address"`
	IsContract        int           `json:"is_contract"`
	DoubtList         int           `json:"doubt_list"` // 1 если есть сомнения
	MaliciousBehavior []interface{} `json:"malicious_behavior"`
	DeployedTime      int64         `json:"deployed_time"`
	TrustList         int           `json:"trust_list"` // 1 если доверенный
	IsOpenSource      int           `json:"is_open_source"`
}

// TokenSecurityResponse - Ответ API GoPlus для токен security
type TokenSecurityResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Result  map[string]struct {
		BuyTax         string `json:"buy_tax"`
		CannotBuy      string `json:"cannot_buy"`
		CreatorAddress string `json:"creator_address"`
		CreatorBalance string `json:"creator_balance"`
		CreatorPercent string `json:"creator_percent"`
		HolderCount    string `json:"holder_count"`
		Holders        []struct {
			Address    string `json:"address"`
			Tag        string `json:"tag"`
			IsContract int    `json:"is_contract"`
			Balance    string `json:"balance"`
			Percent    string `json:"percent"`
			IsLocked   int    `json:"is_locked"`
		} `json:"holders"`
		HoneypotWithSameCreator string `json:"honeypot_with_same_creator"`
		IsInDex                 string `json:"is_in_dex"`
		IsOpenSource            string `json:"is_open_source"`
		IsProxy                 string `json:"is_proxy"`
		OwnerAddress            string `json:"owner_address"`
		SellTax                 string `json:"sell_tax"`
		TokenName               string `json:"token_name"`
		TokenSymbol             string `json:"token_symbol"`
		TotalSupply             string `json:"total_supply"`
	} `json:"result"`
}

// GetTokenApprovals - Получает информацию о токен approvals
func (c *GoPlusClient) GetTokenApprovals(ctx context.Context, address string) (*TokenApprovalResponse, error) {
	url := fmt.Sprintf("https://api.gopluslabs.io/api/v2/token_approval_security/1?addresses=%s", address)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("API-Key", c.apiKey)

	var result TokenApprovalResponse
	if err := c.DoRequest(ctx, req, &result); err != nil {
		return nil, err
	}

	if result.Code != 1 {
		return nil, fmt.Errorf("GoPlus API error: %s", result.Message)
	}

	return &result, nil
}

// GetTokenSecurity - Получает информацию о безопасности токенов
func (c *GoPlusClient) GetTokenSecurity(ctx context.Context, tokenAddresses []string) (*TokenSecurityResponse, error) {
	url := fmt.Sprintf("https://api.gopluslabs.io/api/v1/token_security/1?contract_addresses=%s", strings.Join(tokenAddresses, ","))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	var result TokenSecurityResponse
	if err := c.DoRequest(ctx, req, &result); err != nil {
		return nil, err
	}

	if result.Code != 1 {
		return nil, fmt.Errorf("GoPlus API error: %s", result.Message)
	}

	return &result, nil
}

// getEnv - Получает значение переменной окружения
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// (Note: В реальной реализации config.getEnv должен быть реальным методом или использовать os.Getenv)
