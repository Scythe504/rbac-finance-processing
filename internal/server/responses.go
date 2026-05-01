package server

import (
	"github.com/scythe504/rbac-finance-processing/internal/database"
)

type DashboardResponse struct {
	Data database.DashboardSummary `json:"data"`
}

type RecordResponse struct {
	Data database.Record `json:"data"`
}

type RecordsListResponse struct {
	Data    []database.Record      `json:"data"`
	Filters database.RecordFilters `json:"filters"`
}

type AuthResponse struct {
	UserID string `json:"user_id" example:"1234"`
	Token  string `json:"token" example:"jwt.token.here"`
}

type UserDetailsResponse struct {
	UserID string `json:"user_id" example:"1234"`
	Name   string `json:"name" example:"John Doe"`
	Email  string `json:"email" example:"john@example.com"`
	Role   string `json:"role" example:"admin"`
}

type RecordCreatedResponse struct {
	RecordID int64  `json:"record_id" example:"1"`
	Message  string `json:"message" example:"success"`
}
