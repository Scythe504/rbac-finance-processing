package utils

import (
	"time"

	"github.com/scythe504/rbac-finance-processing/internal/database"
)

func IsValidRole(role string) bool {
	switch role {
	case string(database.RoleAdmin):
		return true
	case string(database.RoleAnalyst):
		return true
	case string(database.RoleViewer):
		return true
	default:
		return false
	}
}

func ValidDashboardQueryPeriod(period database.PeriodType) bool {
	_, ok := database.PeriodConfigs[period]
	return ok
}

func ParseTimeParam(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	return time.Parse("2006-01-02", s)
}