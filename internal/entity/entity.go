package entity

import "time"

// Language - Язык для отчета
type Language string

const (
	LanguageEN Language = "en"
	LanguageRU Language = "ru"
)

// WalletReport - Финальный отчет о безопасности кошелька
type WalletReport struct {
	Address         string        `json:"address"`
	Score           float64       `json:"score"`
	Checks          []CheckResult `json:"checks"`
	Errors          []string      `json:"errors,omitempty"`
	Recommendations string        `json:"recommendations,omitempty"`
}

// CheckResult - Результат одной проверки
type CheckResult struct {
	CheckName    string      `json:"check_name"`
	RiskFound    bool        `json:"risk_found"`
	RiskLevel    RiskLevel   `json:"risk_level"`
	ScorePenalty float64     `json:"score_penalty"`
	Details      string      `json:"details"`
	RawData      interface{} `json:"raw_data"`
}

// RiskLevel - Уровень риска
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "LOW"
	RiskLevelMedium   RiskLevel = "MEDIUM"
	RiskLevelHigh     RiskLevel = "HIGH"
	RiskLevelCritical RiskLevel = "CRITICAL"
)

// TokenInfo - Информация о токене
type TokenInfo struct {
	Address    string  `json:"address"`
	AddressURL string  `json:"addressURL"`
	Name       string  `json:"name"`
	Symbol     string  `json:"symbol"`
	Balance    float64 `json:"balance"`
	USDValue   float64 `json:"usd_value"`
	IsStable   bool    `json:"is_stable"`
}

// ApprovalInfo - Информация о разрешении на токен
type ApprovalInfo struct {
	TokenAddress    string  `json:"token_address"`
	TokenURL        string  `json:"token_url"`
	TokenName       string  `json:"token_name"`
	SpenderAddress  string  `json:"spender_address"`
	SpenderURL      string  `json:"spender_url"`
	ApprovedAmount  string  `json:"approved_amount"`
	ExposureBalance float64 `json:"exposure_balance"`
	IsUnlimited     bool    `json:"is_unlimited"`
	IsMalicious     bool    `json:"is_malicious"`
}

// PaymentStatus - Статус платежа
type PaymentStatus string

const (
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusPaid     PaymentStatus = "paid"
	PaymentStatusCanceled PaymentStatus = "canceled"
)

// Payment - Информация о платеже
type Payment struct {
	GUID      string        `json:"guid"`
	Status    PaymentStatus `json:"status"`
	Address   string        `json:"address"`
	CreatedAt time.Time     `json:"createdAt"`
}
