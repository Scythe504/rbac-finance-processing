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

func (s *Server) getRecords(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	txnType := r.URL.Query().Get("txn_type")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	showDeleted := r.URL.Query().Get("show_deleted")
	ascending := r.URL.Query().Get("ascending")

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

// Admin Only
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

// Admin Only
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
