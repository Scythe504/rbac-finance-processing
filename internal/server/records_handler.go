package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/scythe504/zorvyn-rbac-finance/internal/database"
	"github.com/scythe504/zorvyn-rbac-finance/internal/utils"
	"github.com/shopspring/decimal"
)

// getRecord retrieves a specific finance record
// @Summary Get a record
// @Description Get details of a specific finance record by ID (Analyst/Admin only)
// @Tags Records
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Record ID"
// @Success 200 {object} RecordResponse "Record details"
// @Failure 400 {object} utils.ErrorResponse "Invalid Record ID"
// @Failure 404 {object} utils.ErrorResponse "Record not found"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @ID getRecord
// @Router /records/{id} [get]
func (s *Server) getRecord(w http.ResponseWriter, r *http.Request) {
	recordID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		utils.LogError("getRecord", err)
		utils.WriteError(w, http.StatusBadRequest, "Invalid Record ID")
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

	utils.WriteJSON(w, http.StatusOK, RecordResponse{
		Data: record,
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
// @Success 200 {object} RecordsListResponse "List of records and filters"
// @Failure 400 {object} utils.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @ID getRecords
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

	utils.WriteJSON(w, http.StatusOK, RecordsListResponse{
		Data:    records,
		Filters: filterParams,
	})
}

type CreateRecordRequest struct {
	Amount      decimal.Decimal  `json:"amount" swaggertype:"string" example:"100.50"`
	TxnType     database.TxnType `json:"txn_type" swaggertype:"string" enums:"income,expense" example:"expense"`
	Category    string           `json:"category" example:"Food"`
	Description *string          `json:"description" example:"Lunch at the mall"`
	Date        time.Time        `json:"date" example:"2023-10-27T10:00:00Z"`
}

type UpdateRecordRequest struct {
	Amount      decimal.Decimal  `json:"amount,omitempty" swaggertype:"string" example:"150.00"`
	TxnType     database.TxnType `json:"txn_type,omitempty" swaggertype:"string" enums:"income,expense" example:"income"`
	Category    string           `json:"category,omitempty" example:"Salary"`
	Description *string          `json:"description,omitempty" example:"Monthly paycheck"`
	Date        time.Time        `json:"date,omitempty" example:"2023-10-28T09:00:00Z"`
}

// createRecord creates a new finance record
// @Summary Create a record
// @Description Create a new income or expense record (Admin only)
// @Tags Records
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateRecordRequest true "Record details"
// @Success 201 {object} RecordCreatedResponse "Record created"
// @Failure 400 {object} utils.ErrorResponse "Invalid Request Body"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @ID createRecord
// @Router /records [post]
func (s *Server) createRecord(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		utils.LogError("createRecords", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	var body CreateRecordRequest
	if err = json.Unmarshal(b, &body); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Request Body")
		return
	}

	userID := r.Context().Value(contextKeyUserID).(string)

	// Map request to database model
	record := database.Record{
		Amount:      body.Amount,
		TxnType:     database.TxnType(strings.ToLower(string(body.TxnType))),
		Category:    strings.ToLower(body.Category),
		Description: body.Description,
		Date:        body.Date,
	}

	id, err := s.db.CreateRecord(r.Context(), userID, record)

	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, RecordCreatedResponse{
		RecordID: id,
		Message:  "success",
	})
}

// updateRecord updates an existing finance record
// @Summary Update a record
// @Description Update details of an existing record (Admin only)
// @Tags Records
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body UpdateRecordRequest true "Record updates"
// @Success 200 {object} utils.MessageResponse "Success"
// @Failure 400 {object} utils.ErrorResponse "Invalid Request Body / ID"
// @Failure 404 {object} utils.ErrorResponse "Record not found"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @ID updateRecord
// @Router /records/{id} [patch]
func (s *Server) updateRecord(w http.ResponseWriter, r *http.Request) {
	recordID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		utils.LogError("updateRecords", err)
		utils.WriteError(w, http.StatusBadRequest, "Invalid Record ID")
		return
	}
	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		utils.LogError("updateRecords", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	var body UpdateRecordRequest
	if err = json.Unmarshal(b, &body); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Request Body")
		return
	}

	// Map request to database model for updates
	updates := database.Record{
		Amount:      body.Amount,
		TxnType:     body.TxnType,
		Category:    strings.ToLower(body.Category),
		Description: body.Description,
		Date:        body.Date,
	}

	if err = s.db.UpdateRecord(r.Context(), recordID, updates); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "Not found")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.MessageResponse{
		Message: "success",
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
// @Success 200 {object} utils.MessageResponse "Success"
// @Failure 500 {object} utils.ErrorResponse "Internal Server Error"
// @Failure 400 {object} utils.ErrorResponse "Invalid Record ID"
// @ID deleteRecord
// @Router /records/{id} [delete]
func (s *Server) deleteRecord(w http.ResponseWriter, r *http.Request) {
	recordID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		utils.LogError("deleteRecords", err)
		utils.WriteError(w, http.StatusBadRequest, "Invalid Record ID")
		return
	}

	if err = s.db.DeleteRecord(r.Context(), recordID); err != nil {
		utils.LogError("deleteRecords", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.MessageResponse{
		Message: "success",
	})
}
