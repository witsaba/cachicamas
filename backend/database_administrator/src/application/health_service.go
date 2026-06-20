// Package application contains the use cases of the database administrator
// service. It orchestrates domain logic and is the only layer that depends on
// both the domain types and the driving adapters (e.g. HTTP).
package application

import (
	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// HealthService answers health check queries.
//
// In a fuller implementation this would delegate to domain.Checker
// implementations (DB, downstream services, etc.). For now we just
// report "ok" — KISS.
type HealthService struct{}

// NewHealthService constructs an empty HealthService.
func NewHealthService() *HealthService {
	return &HealthService{}
}

// Check returns the current health report.
func (s *HealthService) Check() domain.HealthReport {
	return domain.HealthReport{Status: domain.StatusOK}
}
