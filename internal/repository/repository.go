package repository

import (
	"context"
	"time"

	"alpha-hygiene-backend/internal/entity"
)

// Repository - Интерфейс для доступа к данным.
// Этот интерфейс абстрагирует конкретную реализацию БД (SQLite/PostgreSQL/etc.).
type Repository interface {
	// GetWalletReport retrieves a cached wallet report by address.
	GetWalletReport(ctx context.Context, address string) (*entity.WalletReport, error)
	// SetWalletReport saves a wallet report to the cache.
	SetWalletReport(ctx context.Context, address string, report *entity.WalletReport, ttl time.Duration) error
	// GetPayment retrieves payment information by GUID.
	GetPayment(ctx context.Context, guid string) (*entity.Payment, error)
	// AddPayment adds a new payment record.
	AddPayment(ctx context.Context, guid string, address string, status entity.PaymentStatus, ttl time.Duration, username string, tgId int64) error
	// ProcessPaymentSuccess processes a successful payment in a single transaction.
	ProcessPaymentSuccess(ctx context.Context, userID string, guid string, reportData *entity.WalletReport) error
	// Close closes the database connection.
	Close() error
}
