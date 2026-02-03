package aggregator

import (
	"context"
	"time"

	"alpha-hygiene-backend/config"
	"alpha-hygiene-backend/internal/cache"
	"alpha-hygiene-backend/internal/checker"
	"alpha-hygiene-backend/internal/entity"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// CheckFactory - Интерфейс для фабрики проверок
type CheckFactory interface {
	CreateCheck(t checker.CheckType) checker.IHealthCheck
}

// Service - Агрегатор проверок
type Service struct {
	cfg     *config.Config
	factory CheckFactory
	cache   cache.Cache
	log     *logrus.Entry
}

// NewService - Создает новый агрегатор
func NewService(cfg *config.Config, factory CheckFactory, cache cache.Cache, log *logrus.Entry) *Service {
	logger := log.WithFields(logrus.Fields{"component": "service"})
	return &Service{
		cfg:     cfg,
		factory: factory,
		cache:   cache,
		log:     logger,
	}
}

// CheckWallet - Проверяет безопасность кошелька
func (s *Service) CheckWallet(ctx context.Context, address string, hasValidPayment bool) (*entity.WalletReport, error) {
	// Создаем основной контекст с таймаутом. Увеличиваем для AI сервиса.
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute) // 5 минут для работы с AI
	defer cancel()
	// Проверяем кэш
	if s.cache != nil {
		cachedReport, err := s.cache.GetWalletReport(ctxWithTimeout, address)

		if err != nil {
			s.log.Errorf("Failed to get cached report: %v", err)
		}
		if !hasValidPayment {
			cachedReport = lightReport(cachedReport)
		}
		if cachedReport != nil {
			s.log.Debugf("Returning cached report for address: %s", address)
			return cachedReport, nil
		}
	}

	// Создаем группу для параллельного запуска проверок с отдельным контекстом
	g, errGrpCtx := errgroup.WithContext(ctxWithTimeout)

	// Получаем все доступные проверки
	checkTypes := checker.GetAllCheckTypes()
	resultsChan := make(chan *entity.CheckResult, len(checkTypes))
	errorsChan := make(chan error, len(checkTypes))

	// Запускаем каждую проверку в отдельной горутине
	for _, t := range checkTypes {
		checkType := t
		g.Go(func() error {
			check := s.factory.CreateCheck(checkType)
			if check == nil {
				return nil
			}

			s.log.Debugf("Executing check: %s for address: %s", check.Name(), address)
			result, err := check.Execute(errGrpCtx, address)
			if err != nil {
				errorsChan <- err
				return nil
			}

			resultsChan <- result
			return nil
		})
	}

	// Ожидаем завершения всех проверок
	if err := g.Wait(); err != nil {
		s.log.Errorf("Error during checks: %v", err)
	}
	close(resultsChan)
	close(errorsChan)

	// Собираем результаты
	var results []*entity.CheckResult
	for res := range resultsChan {
		if res != nil {
			results = append(results, res)
		}
	}

	// Собираем ошибки
	var errors []string
	for err := range errorsChan {
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Рассчитываем итоговый балл
	score := s.calculateScore(results)

	// Формируем отчет
	report := &entity.WalletReport{
		Address: address,
		Score:   score,
		Checks:  make([]entity.CheckResult, len(results)),
		Errors:  errors,
	}

	for i, res := range results {
		report.Checks[i] = *res
	}

	s.log.Infof("Check completed for address: %s, score: %.2f", address, score)

	// предотвращаем кеширование - если были ошибки провайдеров
	if len(errors) > 0 {
		return report, nil
	}
	// Сохраняем в кэш используя основной контекст с таймаутом
	if s.cache != nil {
		if err := s.cache.SetWalletReport(ctxWithTimeout, address, report); err != nil {
			s.log.Errorf("Failed to cache report: %v", err)
		}
	}
	if !hasValidPayment {
		report = lightReport(report)
	}

	return report, nil
}

// calculateScore - Рассчитывает итоговый балл безопасности
func (s *Service) calculateScore(results []*entity.CheckResult) float64 {
	score := s.cfg.Scoring.BaseScore

	for _, res := range results {
		if res.RiskFound {
			score -= res.ScorePenalty
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

func lightReport(report *entity.WalletReport) *entity.WalletReport {
	if report == nil {
		return nil
	}
	for i := range report.Checks {
		report.Checks[i].RawData = nil
	}
	return report
}
