package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"alpha-hygiene-backend/config"
	"alpha-hygiene-backend/internal/entity"

	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

// SQLiteRepository - Реализация Repository для SQLite.
type SQLiteRepository struct {
	db  *sql.DB
	log *logrus.Entry
}

// NewSQLiteRepository - Создает новый экземпляр SQLiteRepository.
func NewSQLiteRepository(cfg *config.Config, log *logrus.Entry) (*SQLiteRepository, error) {
	logger := log.WithFields(logrus.Fields{"component": "sqlite"})

	// Подключение к SQLite
	db, err := sql.Open("sqlite", cfg.SQLite.Path)
	if err != nil {
		logger.Errorf("Failed to open SQLite connection: %v", err)
		return nil, err
	}

	// Настройка соединения
	db.SetMaxOpenConns(1) // SQLite поддерживает только одного пишущего одновременно
	db.SetMaxIdleConns(1)

	// Настройка PRAGMA для SQLite
	if err := setupPRAGMA_WAL(db, logger); err != nil {
		db.Close()
		return nil, err
	}

	// Создание таблиц
	if err := createTables(db, logger); err != nil {
		db.Close()
		return nil, err
	}

	logger.Info("Successfully connected to SQLite")

	repo := &SQLiteRepository{
		db:  db,
		log: logger,
	}

	// Запуск задачи на удаление просроченных записей, если конфиг позволяет
	if cfg.Tasks.SetExpiresTasks {
		logger.Info("Expiration task enabled")
		expirationTask := NewExpirationTask(repo, logger)
		go expirationTask.Run(context.Background())
	} else {
		logger.Info("Expiration task disabled")
	}

	return repo, nil
}

// setupPRAGMA_WAL - Настраивает PRAGMA для SQLite.
func setupPRAGMA_WAL(db *sql.DB, log *logrus.Entry) error {
	// Режим WAL (Write-Ahead Logging) для улучшения параллелизма
	if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		log.Errorf("Failed to set WAL mode: %v", err)
		return err
	}

	// Таймаут ожидания блокировки (5 секунд)
	if _, err := db.Exec("PRAGMA busy_timeout = 5000;"); err != nil {
		log.Errorf("Failed to set busy timeout: %v", err)
		return err
	}

	// Синхронизация с диском (более быстрая запись)
	if _, err := db.Exec("PRAGMA synchronous = NORMAL;"); err != nil {
		log.Errorf("Failed to set synchronous mode: %v", err)
		return err
	}

	return nil
}

// createTables - Создает необходимые таблицы в SQLite.
func createTables(db *sql.DB, log *logrus.Entry) error {
	// Таблица для хранения платежей
	paymentTableSQL := `
		CREATE TABLE IF NOT EXISTS payments (
			guid TEXT PRIMARY KEY,
			user_id TEXT,
			status TEXT NOT NULL,
			address TEXT NOT NULL,
			amount REAL,
			created_at DATETIME NOT NULL,
			expires_at DATETIME,
			username TEXT,
			tg_id INTEGER
		);
	`

	if _, err := db.Exec(paymentTableSQL); err != nil {
		log.Errorf("Failed to create payments table: %v", err)
		return err
	}

	// Таблица для хранения отчетов кошельков (с JSON полем)
	walletReportsSQL := `
		CREATE TABLE IF NOT EXISTS wallet_reports (
			address TEXT PRIMARY KEY,
			report TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			expires_at DATETIME
		);
	`

	if _, err := db.Exec(walletReportsSQL); err != nil {
		log.Errorf("Failed to create wallet_reports table: %v", err)
		return err
	}

	// Индексы для ускорения поиска
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_payments_address ON payments(address);"); err != nil {
		log.Errorf("Failed to create payments address index: %v", err)
		return err
	}

	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_wallet_reports_expires ON wallet_reports(expires_at);"); err != nil {
		log.Errorf("Failed to create wallet_reports expires index: %v", err)
		return err
	}

	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_payments_expires ON payments(expires_at);"); err != nil {
		log.Errorf("Failed to create payments expires index: %v", err)
		return err
	}

	return nil
}

// GetWalletReport - Получает отчет о кошельке из SQLite.
func (r *SQLiteRepository) GetWalletReport(ctx context.Context, address string) (*entity.WalletReport, error) {
	var reportJSON string
	query := `SELECT report FROM wallet_reports WHERE address = ? AND (expires_at IS NULL OR expires_at > ?)`
	err := r.db.QueryRowContext(ctx, query, address, time.Now()).Scan(&reportJSON)
	if err == sql.ErrNoRows {
		r.log.Debugf("Wallet report not found for address: %s", address)
		return nil, nil
	} else if err != nil {
		r.log.Errorf("Failed to get wallet report: %v", err)
		return nil, err
	}

	var report entity.WalletReport
	if err := json.Unmarshal([]byte(reportJSON), &report); err != nil {
		r.log.Errorf("Failed to unmarshal wallet report: %v", err)
		return nil, err
	}

	r.log.Debugf("Wallet report found for address: %s", address)
	return &report, nil
}

// SetWalletReport - Сохраняет отчет о кошельке в SQLite.
func (r *SQLiteRepository) SetWalletReport(ctx context.Context, address string, report *entity.WalletReport, ttl time.Duration) error {
	reportJSON, err := json.Marshal(report)
	if err != nil {
		r.log.Errorf("Failed to marshal wallet report: %v", err)
		return err
	}

	expiresAt := time.Now().Add(ttl)
	query := `
		INSERT INTO wallet_reports (address, report, created_at, expires_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(address)
		DO UPDATE SET report = excluded.report, expires_at = excluded.expires_at
	`

	_, err = r.db.ExecContext(ctx, query, address, reportJSON, time.Now(), expiresAt)
	if err != nil {
		r.log.Errorf("Failed to set wallet report: %v", err)
		return err
	}

	r.log.Debugf("Wallet report saved for address: %s", address)
	return nil
}

// GetPayment - Получает информацию о платеже из SQLite по GUID.
func (r *SQLiteRepository) GetPayment(ctx context.Context, guid string) (*entity.Payment, error) {
	var payment entity.Payment
	query := `SELECT guid, status, address, created_at, username, tg_id FROM payments WHERE guid = ? AND (expires_at IS NULL OR expires_at > ?)`
	err := r.db.QueryRowContext(ctx, query, guid, time.Now()).Scan(&payment.GUID, &payment.Status, &payment.Address, &payment.CreatedAt, &payment.Username, &payment.TgId)
	if err == sql.ErrNoRows {
		r.log.Debugf("Payment not found for GUID: %s", guid)
		return nil, nil
	} else if err != nil {
		r.log.Errorf("Failed to get payment: %v", err)
		return nil, err
	}

	r.log.Debugf("Payment found for GUID: %s", guid)
	return &payment, nil
}

// AddPayment - Добавляет новый платеж в SQLite.
func (r *SQLiteRepository) AddPayment(ctx context.Context, guid string, address string, status entity.PaymentStatus, ttl time.Duration, username string, tgId int64) error {
	expiresAt := time.Now().Add(ttl)
	query := `
		INSERT INTO payments (guid, user_id, status, address, amount, created_at, expires_at, username, tg_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(guid)
		DO UPDATE SET status = excluded.status, expires_at = excluded.expires_at, username = excluded.username, tg_id = excluded.tg_id
	`

	_, err := r.db.ExecContext(ctx, query, guid, nil, status, address, nil, time.Now(), expiresAt, username, tgId)
	if err != nil {
		r.log.Errorf("Failed to add payment: %v", err)
		return err
	}

	r.log.Debugf("Payment added for GUID: %s", guid)
	return nil
}

// deleteExpiredWalletReports - Удаляет просроченные отчеты кошельков.
func (r *SQLiteRepository) deleteExpiredWalletReports(ctx context.Context) error {
	query := `DELETE FROM wallet_reports WHERE expires_at <= ?`
	_, err := r.db.ExecContext(ctx, query, time.Now())
	return err
}

// ProcessPaymentSuccess processes a successful payment in a single transaction.
func (r *SQLiteRepository) ProcessPaymentSuccess(ctx context.Context, userID string, guid string, reportData *entity.WalletReport) error {
	// Создаем транзакцию с контекстом для управления временем ожидания
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.log.Errorf("Failed to begin transaction: %v", err)
		return err
	}

	// Устанавливаем defer для роллбэка в случае ошибки
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				r.log.Errorf("Failed to rollback transaction: %v", rollbackErr)
			}
		}
	}()

	// Шаг 1: Проверка существования платежа с данным guid и статусом pending
	var existingStatus string
	checkQuery := `SELECT status FROM payments WHERE guid = ? AND status = ?`
	err = tx.QueryRowContext(ctx, checkQuery, guid, entity.PaymentStatusPending).Scan(&existingStatus)
	if err == sql.ErrNoRows {
		return fmt.Errorf("payment with GUID %s not found or not in pending status", guid)
	} else if err != nil {
		r.log.Errorf("Failed to check payment existence: %v", err)
		return err
	}

	// Шаг 2: Обновление статуса платежа на paid и установка address = '' (очистка временных данных)
	updateQuery := `UPDATE payments SET status = ?, address = '', user_id = ? WHERE guid = ?`
	_, err = tx.ExecContext(ctx, updateQuery, entity.PaymentStatusPaid, userID, guid)
	if err != nil {
		r.log.Errorf("Failed to update payment status: %v", err)
		return err
	}

	// Шаг 3: Сохранение финального JSON-отчета в таблицу wallet_reports
	reportJSON, err := json.Marshal(reportData)
	if err != nil {
		r.log.Errorf("Failed to marshal report data: %v", err)
		return err
	}

	// Если отчет уже существует, обновляем его; иначе создаем новый
	upsertQuery := `
		INSERT INTO wallet_reports (address, report, created_at, expires_at)
		VALUES (?, ?, ?, NULL)
		ON CONFLICT(address)
		DO UPDATE SET report = excluded.report, expires_at = NULL
	`

	_, err = tx.ExecContext(ctx, upsertQuery, reportData.Address, reportJSON, time.Now())
	if err != nil {
		r.log.Errorf("Failed to save report data: %v", err)
		return err
	}

	// Коммит транзакции
	if err = tx.Commit(); err != nil {
		r.log.Errorf("Failed to commit transaction: %v", err)
		return err
	}

	r.log.Debugf("Payment processed successfully for GUID: %s", guid)
	return nil
}

// deleteExpiredPayments - Удаляет просроченные платежи.
func (r *SQLiteRepository) deleteExpiredPayments(ctx context.Context) error {
	query := `DELETE FROM payments WHERE expires_at <= ?`
	_, err := r.db.ExecContext(ctx, query, time.Now())
	return err
}

// Close - Закрывает соединение с SQLite.
func (r *SQLiteRepository) Close() error {
	if err := r.db.Close(); err != nil {
		r.log.Errorf("Failed to close SQLite connection: %v", err)
		return err
	}

	r.log.Info("SQLite connection closed")
	return nil
}
