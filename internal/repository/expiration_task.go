package repository

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// ExpirationTask - Задача для удаления просроченных записей.
type ExpirationTask struct {
	repo *SQLiteRepository
	log  *logrus.Entry
	tick time.Duration
}

// NewExpirationTask - Создает новую задачу для удаления просроченных записей.
func NewExpirationTask(repo *SQLiteRepository, log *logrus.Entry) *ExpirationTask {
	return &ExpirationTask{
		repo: repo,
		log:  log,
		tick: 1 * time.Minute, // По умолчанию проверка каждую минуту
	}
}

// Run запускает задачу на удаление просроченных записей.
func (t *ExpirationTask) Run(ctx context.Context) {
	ticker := time.NewTicker(t.tick)
	defer ticker.Stop()

	t.log.Info("Expiration task started")

	for {
		select {
		case <-ctx.Done():
			t.log.Info("Expiration task stopped")
			return
		case <-ticker.C:
			if err := t.runCleanup(ctx); err != nil {
				t.log.Errorf("Cleanup failed: %v", err)
			}
		}
	}
}

// runCleanup выполняет очистку просроченных записей.
func (t *ExpirationTask) runCleanup(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Удаляем просроченные отчеты кошельков
	if err := t.repo.deleteExpiredWalletReports(ctx); err != nil {
		return err
	}

	// Удаляем просроченные платежи
	if err := t.repo.deleteExpiredPayments(ctx); err != nil {
		return err
	}

	t.log.Debug("Cleanup completed successfully")
	return nil
}
