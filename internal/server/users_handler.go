package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/mail"

	"github.com/gorilla/mux"
	"github.com/scythe504/zorvyn-rbac-finance/internal/database"
	"github.com/scythe504/zorvyn-rbac-finance/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

type RegisterUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type LoginUser struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// registerUser registers a new user
// @Summary Register a new user
// @Description Create a new account with a specific role
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body RegisterUser true "User registration details"
// @Success 201 {object} map[string]string "User registered successfully"
// @Failure 400 {object} map[string]string "Invalid Request Body / Email / Role"
// @Failure 409 {object} map[string]string "User with email already exists"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @ID registerUser
// @Router /auth/register [post]
func (s *Server) registerUser(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		utils.LogError("registerUser", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	var body RegisterUser
	if err = json.Unmarshal(b, &body); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Request Body")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.LogError("registerUser", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	emailAddr, err := mail.ParseAddress(body.Email)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Email Address")
		return
	}

	if !utils.IsValidRole(body.Role) {
		utils.WriteError(w, http.StatusBadRequest, "Invalid User Role")
		return
	}

	user := database.User{
		Name:     body.Name,
		Email:    emailAddr.Address,
		Password: string(hash),
		Role:     database.RoleType(body.Role),
	}

	userId, err := s.db.CreateUser(r.Context(), user)
	if err != nil {
		if errors.Is(err, database.ErrDuplicateEmail) {
			utils.WriteError(w, http.StatusConflict, "User with email already exists, Please Login")
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	token, err := utils.GenerateJWTToken(userId, database.RoleType(body.Role))
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, map[string]string{
		"user_id": userId,
		"token":   token,
	})
}

// loginUser authenticates a user
// @Summary Login a user
// @Description Authenticate user and return JWT token
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body LoginUser true "Login credentials"
// @Success 200 {object} map[string]string "Login successful"
// @Failure 400 {object} map[string]string "Invalid Request Body"
// @Failure 401 {object} map[string]string "Invalid Credentials"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @ID loginUser
// @Router /auth/login [post]
func (s *Server) loginUser(w http.ResponseWriter, r *http.Request) {
	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		utils.LogError("loginUser", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	var body LoginUser
	if err = json.Unmarshal(b, &body); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Request Body")
		return
	}

	if _, err := mail.ParseAddress(body.Email); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Email Address")
		return
	}

	authUser, err := s.db.GetUserByEmail(r.Context(), body.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.WriteError(w, http.StatusUnauthorized, "Invalid Credentials")
			return
		}
		utils.LogError("loginUser", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(authUser.Password), []byte(body.Password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			utils.WriteError(w, http.StatusUnauthorized, "Invalid Credentials")
			return
		}
		utils.LogError("loginUser", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	token, err := utils.GenerateJWTToken(authUser.ID, authUser.Role)
	if err != nil {
		utils.LogError("loginUser", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"user_id": authUser.ID,
		"token":   token,
	})
}

// getUserDetails returns current user information
// @Summary Get user details
// @Description Get current logged in user information
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "User details retrieved"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @ID getUserDetails
// @Router /me [get]
func (s *Server) getUserDetails(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(contextKeyUserID).(string)
	user, err := s.db.GetUserById(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			utils.WriteError(w, http.StatusNotFound, "User not found")
			return
		}
		utils.LogError("getUserDetails", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"user_id": user.ID,
		"name":    user.Name,
		"email":   user.Email,
		"role":    string(user.Role),
	})
}

// toggleUserStatus toggles user active status
// @Summary Toggle user status
// @Description Enable or disable a user account (Admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Success 200 {object} map[string]string "Success"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @ID toggleUserStatus
// @Router /users/{id}/status [patch]
func (s *Server) toggleUserStatus(w http.ResponseWriter, r *http.Request) {
	targetUserId := mux.Vars(r)["id"]

	if err := s.db.ToggleUserStatus(r.Context(), targetUserId); err != nil {
		utils.LogError("toggleUserStatus", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "success",
	})
}

// setUserRole updates user role
// @Summary Set user role
// @Description Update the role of a specific user (Admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID"
// @Param body body object{target_user_role=string} true "Target role details"
// @Success 200 {object} map[string]string "Success"
// @Failure 400 {object} map[string]string "Invalid Request Body / Role"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @ID setUserRole
// @Router /users/{id}/role [patch]
func (s *Server) setUserRole(w http.ResponseWriter, r *http.Request) {
	targetUserId := mux.Vars(r)["id"]
	b, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		utils.LogError("setUserRole", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	var body struct {
		Role database.RoleType `json:"target_user_role"`
	}

	if err = json.Unmarshal(b, &body); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Invalid Request Body")
		return
	}

	if !utils.IsValidRole(string(body.Role)) {
		utils.WriteError(w, http.StatusBadRequest, "Invalid User Role")
		return
	}

	if err = s.db.AssignUserRole(r.Context(), targetUserId, body.Role); err != nil {
		utils.LogError("setUserRole", err)
		utils.WriteError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "success",
	})
}
