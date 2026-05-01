package server

import (
	"net/http"
	"strings"

	"github.com/scythe504/rbac-finance-processing/internal/database"
	"github.com/scythe504/rbac-finance-processing/internal/utils"
)

// getDashboardSummary retrieves the financial dashboard summary
// @Summary Get dashboard summary
// @Description Get a summary of income, expenses, trends and recent activities
// @Tags Dashboard
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param period query string false "Summary period (weekly, monthly, yearly, all)"
// @Success 200 {object} DashboardResponse "Dashboard summary data"
// @Failure 400 {object} utils.ErrorResponse "Invalid Query Parameter"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @ID getDashboardSummary
// @Router /dashboard [get]
func (s *Server) getDashboardSummary(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	safePeriod := database.PeriodType(strings.ToLower(period))
	if safePeriod == "" {
		safePeriod = database.PeriodWeekly
	}
	if !utils.ValidDashboardQueryPeriod(database.PeriodType(safePeriod)) {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Query Parameter - period")
		return
	}

	dashbSummary, err := s.db.GetDashboardSummary(r.Context(), safePeriod)
	if err != nil {
		utils.LogError("getDashboardSummary", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, DashboardResponse{
		Data: dashbSummary,
	})
}
