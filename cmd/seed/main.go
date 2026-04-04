package main

import (
	"slices"
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
	"github.com/scythe504/zorvyn-rbac-finance/internal/database"
	"github.com/scythe504/zorvyn-rbac-finance/internal/utils"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
)

func main() {
	fmt.Printf("%sStarting database seeding...%s\n", ColorCyan, ColorReset)

	dbHost := os.Getenv("BLUEPRINT_DB_HOST")
	dbPort := os.Getenv("BLUEPRINT_DB_PORT")
	dbUser := os.Getenv("BLUEPRINT_DB_USERNAME")
	dbPass := os.Getenv("BLUEPRINT_DB_PASSWORD")
	dbName := os.Getenv("BLUEPRINT_DB_DATABASE")
	dbSchema := os.Getenv("BLUEPRINT_DB_SCHEMA")

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s", dbUser, dbPass, dbHost, dbPort, dbName, dbSchema)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("%sError opening database: %v%s\n", ColorRed, err, ColorReset)
	}
	defer db.Close()

	ctx := context.Background()

	// 1. Create Users
	users := []struct {
		Name     string
		Email    string
		Password string
		Role     database.RoleType
	}{
		{"Admin User", "admin@zorvyn.local", "password123", database.RoleAdmin},
		{"Analyst User", "analyst@zorvyn.local", "password123", database.RoleAnalyst},
		{"Viewer User", "viewer@zorvyn.local", "password123", database.RoleViewer},
	}

	var seededUsers []database.User
	var responses []string

	for _, u := range users {
		fmt.Printf("%sCreating user: %s (%s)%s\n", ColorYellow, u.Name, u.Role, ColorReset)

		hash, _ := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		user := database.User{
			Name:     u.Name,
			Email:    u.Email,
			Password: string(hash),
			Role:     u.Role,
		}

		// Check if user exists
		var existingID string
		err := db.QueryRowContext(ctx, "SELECT id FROM users WHERE email = $1", u.Email).Scan(&existingID)
		if err == nil {
			fmt.Printf("%sUser %s already exists, skipping creation.%s\n", ColorCyan, u.Email, ColorReset)
			user.ID = existingID
		} else {
			err = db.QueryRowContext(ctx, `INSERT INTO users (name, email, password, role) VALUES ($1, $2, $3, $4) RETURNING id`,
				user.Name, user.Email, user.Password, user.Role).Scan(&user.ID)
			if err != nil {
				log.Fatalf("%sError creating user %s: %v%s\n", ColorRed, u.Email, err, ColorReset)
			}
			fmt.Printf("%sUser %s created with ID: %s%s\n", ColorGreen, u.Email, user.ID, ColorReset)
		}

		seededUsers = append(seededUsers, user)

		token, err := utils.GenerateJWTToken(user.ID, user.Role)
		if err != nil {
			log.Fatalf("%sError generating token for %s: %v%s\n", ColorRed, u.Email, err, ColorReset)
		}

		resp := fmt.Sprintf("## %s (%s)\n- **Email:** %s\n- **Password:** %s\n- **Response:**\n```json\n{\n  \"user_id\": \"%s\",\n  \"token\": \"%s\"\n}\n```\n",
			u.Name, u.Role, u.Email, u.Password, user.ID, token)
		responses = append(responses, resp)
	}

	// 2. Write users.md
	var usersMdContent strings.Builder
	usersMdContent.WriteString("# Sample Users\n\n")
	for _, r := range responses {
		usersMdContent.WriteString(r + "\n")
	}
	err = os.WriteFile("USERS.md", []byte(usersMdContent.String()), 0644)
	if err != nil {
		log.Fatalf("%sError writing users.md: %v%s\n", ColorRed, err, ColorReset)
	}
	fmt.Printf("%sCreated users.md with sample responses.%s\n", ColorGreen, ColorReset)

	// 3. Seed Records
	categories := []string{"Food", "Housing", "Transport", "Entertainment", "Utilities", "Salary", "Investment", "Shopping", "Health"}
	incomeCategories := []string{"Salary", "Investment"}

	now := time.Now()
	startDate := now.AddDate(-2, 0, 0)

	totalRecords := 0
	fmt.Printf("%sSeeding records from %s to %s...%s\n", ColorCyan, startDate.Format("2006-01-02"), now.Format("2006-01-02"), ColorReset)

	// Use Admin User as the owner for the seeded records
	adminID := seededUsers[0].ID

	for d := startDate; d.Before(now); d = d.AddDate(0, 1, 0) {
		recordCount := 30 + rand.Intn(11) // 30-40 records
		fmt.Printf("%sMonth: %s - Seeding %d records%s\n", ColorYellow, d.Format("2006-01"), recordCount, ColorReset)

		for range recordCount {
			// Random day in month
			day := rand.Intn(28) + 1
			txnDate := time.Date(d.Year(), d.Month(), day, 0, 0, 0, 0, time.UTC)
			if txnDate.After(now) {
				txnDate = now
			}

			category := categories[rand.Intn(len(categories))]
			isIncome := slices.Contains(incomeCategories, category)

			txnType := "expense"
			amount := decimal.NewFromFloat(10.0 + rand.Float64()*100.0)
			if isIncome {
				txnType = "income"
				amount = decimal.NewFromFloat(500.0 + rand.Float64()*2000.0)
			}

			description := fmt.Sprintf("Sample %s for %s", txnType, category)

			_, err = db.ExecContext(ctx, `INSERT INTO records (user_id, amount, txn_type, category, description, date) VALUES ($1, $2, $3, $4, $5, $6)`,
				adminID, amount, txnType, category, description, txnDate)
			if err != nil {
				log.Fatalf("%sError inserting record: %v%s\n", ColorRed, err, ColorReset)
			}
			totalRecords++
		}
	}

	fmt.Printf("%sSeeding complete! Total records inserted: %d%s\n", ColorGreen, totalRecords, ColorReset)
}
