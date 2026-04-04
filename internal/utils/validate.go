package utils

import (
	"time"

	"github.com/scythe504/zorvyn-rbac-finance/internal/database"
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
	switch period {
	case database.PeriodWeekly:
		return true
	case database.PeriodMonthly:
		return true
	case database.PeriodYearly:
		return true
	case database.PeriodAllTime:
		return true
	default:
		return false
	}
}

func ParseTimeParam(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	return time.Parse("2006-01-02", s)
}