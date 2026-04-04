package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/scythe504/zorvyn-rbac-finance/internal/database"
	"github.com/scythe504/zorvyn-rbac-finance/internal/utils"
)

// getRecord retrieves a specific finance record
// @Summary Get a record
// @Description Get details of a specific finance record by ID (Analyst/Admin only)
// @Tags Records
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Record ID"
// @Success 200 {object} map[string]database.Record "Record details"
// @Failure 404 {object} map[string]string "Record not found"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /records/{id} [get]
func (s *Server) getRecord(w http.ResponseWriter, r *http.Request) {
	recordID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		utils.LogError("getRecord", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	record, err := s.db.GetRecord(r.Context(), recordID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Record not found")
			return
		}
		utils.LogError("getRecord", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]any{
		"data": record,
	})
}

// getRecords retrieves a list of finance records with filtering
// @Summary List records
// @Description Get a list of finance records with various filters (Analyst/Admin only)
// @Tags Records
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param category query string false "Filter by category"
// @Param txn_type query string false "Filter by transaction type (income/expense)"
// @Param from query string false "Filter by start date (YYYY-MM-DD)"
// @Param to query string false "Filter by end date (YYYY-MM-DD)"
// @Param show_deleted query bool false "Include deleted records"
// @Param ascending query bool false "Sort ascending"
// @Param limit query int false "Limit number of results"
// @Param page query int false "Page number for pagination"
// @Success 200 {object} map[string]any "List of records and filters"
// @Failure 400 {object} map[string]string "Invalid query parameters"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /records [get]
func (s *Server) getRecords(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	txnType := r.URL.Query().Get("txn_type")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	showDeleted := r.URL.Query().Get("show_deleted")
	ascending := r.URL.Query().Get("ascending")
	limit := r.URL.Query().Get("limit")
	page := r.URL.Query().Get("page")

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
		TxnType:     strings.ToLower(txnType),
		Category:    strings.ToLower(category),
		From:        safeFrom,
		To:          safeTo,
		ShowDeleted: showDeleted == "true",
		Ascending:   strings.ToLower(ascending) == "true",
	}

	if limit != "" {
		filterParams.Limit, _ = strconv.Atoi(limit)
	}

	if page != "" {
		p, _ := strconv.Atoi(page)
		filterParams.Offset = (p - 1) * filterParams.Limit
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

// createRecord creates a new finance record
// @Summary Create a record
// @Description Create a new income or expense record (Admin only)
// @Tags Records
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body database.Record true "Record details"
// @Success 201 {object} map[string]any "Record created"
// @Failure 400 {object} map[string]string "Invalid Request Body"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /records [post]
func (s *Server) createRecord(w http.ResponseWriter, r *http.Request) {
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
		"record_id": id,
		"message":   "success",
	})
}

// updateRecord updates an existing finance record
// @Summary Update a record
// @Description Update details of an existing record (Admin only)
// @Tags Records
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Record ID"
// @Param body body database.Record true "Record updates"
// @Success 200 {object} map[string]string "Success"
// @Failure 400 {object} map[string]string "Invalid Request Body"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /records/{id} [patch]
func (s *Server) updateRecord(w http.ResponseWriter, r *http.Request) {
	recordID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		utils.LogError("updateRecords", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		utils.LogError("updateRecords", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	var body database.Record
	if err = json.Unmarshal(b, &body); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Request Body")
		return
	}
	if err = s.db.UpdateRecord(r.Context(), recordID, body); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "success",
	})
}

// deleteRecord soft-deletes a finance record
// @Summary Delete a record
// @Description Delete a record by ID (Admin only)
// @Tags Records
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Record ID"
// @Success 200 {object} map[string]string "Success"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /records/{id} [delete]
func (s *Server) deleteRecord(w http.ResponseWriter, r *http.Request) {
	recordID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		utils.LogError("deleteRecords", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	if err = s.db.DeleteRecord(r.Context(), recordID); err != nil {
		utils.LogError("deleteRecords", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "success",
	})
}
