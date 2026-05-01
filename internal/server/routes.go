package server

import (
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/scythe504/rbac-finance-processing/docs"
	"github.com/scythe504/rbac-finance-processing/internal/database"
	"github.com/scythe504/rbac-finance-processing/internal/utils"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := mux.NewRouter()

	// Apply CORS middleware
	r.Use(s.corsMiddleware)

	r.PathPrefix("/swagger").Handler(httpSwagger.Handler(
		httpSwagger.URL("swagger/doc.json"),
	))
	apiRoutesV1 := r.PathPrefix("/api/v1").Subrouter()
	apiRoutesV1.HandleFunc("/health", s.healthHandler)
	apiRoutesV1.HandleFunc("/", s.HelloWorldHandler)

	authRoutes := apiRoutesV1.PathPrefix("/auth").Subrouter()
	authRoutes.HandleFunc("/register", s.registerUser).Methods("POST")
	authRoutes.HandleFunc("/login", s.loginUser).Methods("POST")

	// Admin, Analyst, Viewer
	protectedRoutes := apiRoutesV1.NewRoute().Subrouter()
	protectedRoutes.Use(s.authMiddleWare)
	protectedRoutes.HandleFunc("/me", s.getUserDetails).Methods("GET")
	protectedRoutes.HandleFunc("/dashboard", s.getDashboardSummary).Methods("GET")

	// Admin, Analyst
	analystRoutes := protectedRoutes.NewRoute().Subrouter()
	analystRoutes.Use(s.requireRole(database.RoleAnalyst, database.RoleAdmin))
	analystRoutes.HandleFunc("/records", s.getRecords).Methods("GET")
	analystRoutes.HandleFunc("/records/{id}", s.getRecord).Methods("GET")

	// Admin Only
	adminRoutes := protectedRoutes.NewRoute().Subrouter()
	adminRoutes.Use(s.requireRole(database.RoleAdmin))
	adminRoutes.HandleFunc("/users/{id}/role", s.setUserRole).Methods("PATCH")
	adminRoutes.HandleFunc("/users/{id}/status", s.toggleUserStatus).Methods("PATCH")
	adminRoutes.HandleFunc("/records/{id}", s.updateRecord).Methods("PATCH")
	adminRoutes.HandleFunc("/records/{id}", s.deleteRecord).Methods("DELETE")
	adminRoutes.HandleFunc("/records", s.createRecord).Methods("POST")

	return r
}

// CORS middleware
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS Headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Wildcard allows all origins
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "false") // Credentials not allowed with wildcard origins

		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// HelloWorldHandler returns a greeting
// @Summary Hello World
// @Description Basic greeting to check if API is reachable
// @Produce json
// @Success 200 {object} utils.MessageResponse
// @ID helloWorld
// @Router / [get]
func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	utils.WriteJSON(w, http.StatusOK, utils.MessageResponse{
		Message: "Hello World",
	})
}

// healthHandler returns database health status
// @Summary API Health
// @Description Check the health of the API and its database connection
// @Produce json
// @Success 200 {object} database.HealthStats "Health status statistics"
// @ID healthCheck
// @Router /health [get]
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	utils.WriteJSON(w, http.StatusOK, s.db.Health())
}
