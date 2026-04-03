package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/scythe504/zorvyn-rbac-finance/internal/database"
	"github.com/scythe504/zorvyn-rbac-finance/internal/utils"
)

func (s *Server) getRecords(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	txn_type := r.URL.Query().Get("txn-type")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	includeDeleted := r.URL.Query().Get("deleted")
	orderBy := r.URL.Query().Get("orderby")

	safeFrom, err := utils.ParseTimeParam(from)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid from timestring, required YYYY-MM-DD format")
		return
	}

	safeTo, err := utils.ParseTimeParam(to)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid to timestring, required YYYY-MM-DD format")
		return
	}

	filterParams := database.RecordFilters{
		TxnType:     strings.ToLower(txn_type),
		Category:    strings.ToLower(category),
		From:        safeFrom,
		To:          safeTo,
		ShowDeleted: includeDeleted == "true",
		Ascending:   strings.ToLower(orderBy) == "asc",
	}

	records, err := s.db.GetRecords(r.Context(), &filterParams)
	if err != nil {
		utils.LogError("getRecords", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"data":    records,
		"filters": filterParams,
	})
}

// Admin Only
func (s *Server) createRecords(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		utils.LogError("createRecords", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	var body database.Record
	if err = json.Unmarshal(b, &body); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Request Body")
		return
	}

	userID := r.Context().Value(contextKeyUserID).(string)
	id, err := s.db.CreateRecord(r.Context(), userID, body)

	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]any{
		"record_id":      id,
		"message": "success",
	})
}

// Admin Only
func (s *Server) updateRecords(w http.ResponseWriter, r *http.Request) {

}

// Admin Only
func (s *Server) deleteRecords(w http.ResponseWriter, r *http.Request) {

}
