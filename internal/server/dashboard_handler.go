package server

import (
	"net/http"
	"strings"

	"github.com/scythe504/zorvyn-rbac-finance/internal/database"
	"github.com/scythe504/zorvyn-rbac-finance/internal/utils"
)

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

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"data": dashbSummary,
	})
}
