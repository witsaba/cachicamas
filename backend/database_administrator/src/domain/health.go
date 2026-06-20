// Package domain contains the core business types of the database administrator
// service. It has no dependencies on frameworks, transport, or infrastructure.
package domain

// Status represents the health status of the service.
type Status string

// Health status values reported by a HealthReport.
const (
	// StatusOK means every checked dependency is healthy.
	StatusOK Status = "ok"
	// StatusDown means at least one checked dependency is unhealthy.
	StatusDown Status = "down"
)

// HealthReport is the result of a health check.
type HealthReport struct {
	Status Status `json:"status"`
}
