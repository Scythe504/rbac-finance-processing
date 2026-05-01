package main

import (
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
	"github.com/scythe504/rbac-finance-processing/internal/database"
	"github.com/scythe504/rbac-finance-processing/internal/utils"
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

	appEnv := strings.ToLower(os.Getenv("APP_ENV"))
	domain := "local"
	mdFile := "USERS-LOCAL.md"
	if appEnv == "prod" {
		domain = "prod"
		mdFile = "USERS.md"
	}

	dbHost := os.Getenv("BLUEPRINT_DB_HOST")
	dbPort := os.Getenv("BLUEPRINT_DB_PORT")
	dbUser := os.Getenv("BLUEPRINT_DB_USERNAME")
	dbPass := os.Getenv("BLUEPRINT_DB_PASSWORD")
	dbName := os.Getenv("BLUEPRINT_DB_DATABASE")
	dbSchema := os.Getenv("BLUEPRINT_DB_SCHEMA")
	dbUrl := os.Getenv("DATABASE_URL")
	var connStr string
	if dbUrl != "" {
		connStr = dbUrl
	} else {
		connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s", dbUser, dbPass, dbHost, dbPort, dbName, dbSchema)
	}
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("%sError opening database: %v%s\n", ColorRed, err, ColorReset)
	}
	defer db.Close()

	ctx := context.Background()

	// 1. Create Users within a transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatalf("%sError starting user transaction: %v%s\n", ColorRed, err, ColorReset)
	}

	usersToSeed := []struct {
		Name     string
		Email    string
		Password string
		Role     database.RoleType
	}{
		{"Admin User", "admin@rbac-finance." + domain, "password123", database.RoleAdmin},
		{"Analyst User", "analyst@rbac-finance." + domain, "password123", database.RoleAnalyst},
		{"Viewer User", "viewer@rbac-finance." + domain, "password123", database.RoleViewer},
	}

	var seededUsers []database.User
	var responses []string

	for _, u := range usersToSeed {
		fmt.Printf("%sProcessing user: %s (%s)%s\n", ColorYellow, u.Name, u.Role, ColorReset)

		hash, _ := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		user := database.User{
			Name:     u.Name,
			Email:    u.Email,
			Password: string(hash),
			Role:     u.Role,
		}

		var existingID string
		err := tx.QueryRowContext(ctx, "SELECT id FROM users WHERE email = $1", u.Email).Scan(&existingID)
		if err == nil {
			fmt.Printf("%sUser %s already exists, updating ID reference.%s\n", ColorCyan, u.Email, ColorReset)
			user.ID = existingID
		} else {
			err = tx.QueryRowContext(ctx, `INSERT INTO users (name, email, password, role) VALUES ($1, $2, $3, $4) RETURNING id`,
				user.Name, user.Email, user.Password, user.Role).Scan(&user.ID)
			if err != nil {
				tx.Rollback()
				log.Fatalf("%sError creating user %s: %v%s\n", ColorRed, u.Email, err, ColorReset)
			}
			fmt.Printf("%sUser %s created with ID: %s%s\n", ColorGreen, u.Email, user.ID, ColorReset)
		}

		seededUsers = append(seededUsers, user)

		token, err := utils.GenerateJWTToken(user.ID, user.Role)
		if err != nil {
			tx.Rollback()
			log.Fatalf("%sError generating token for %s: %v%s\n", ColorRed, u.Email, err, ColorReset)
		}

		resp := fmt.Sprintf("## %s (%s)\n- **Email:** %s\n- **Password:** %s\n- **Response:**\n```json\n{\n  \"user_id\": \"%s\",\n  \"token\": \"%s\"\n}\n```\n",
			u.Name, u.Role, u.Email, u.Password, user.ID, token)
		responses = append(responses, resp)
	}

	if err = tx.Commit(); err != nil {
		log.Fatalf("%sError committing user transaction: %v%s\n", ColorRed, err, ColorReset)
	}

	// 2. Write Markdown file
	var mdContent strings.Builder
	mdContent.WriteString(fmt.Sprintf("# Sample Users (%s)\n\n", strings.ToUpper(appEnv)))
	for _, r := range responses {
		mdContent.WriteString(r + "\n")
	}
	if err = os.WriteFile(mdFile, []byte(mdContent.String()), 0644); err != nil {
		log.Fatalf("%sError writing %s: %v%s\n", ColorRed, mdFile, err, ColorReset)
	}
	fmt.Printf("%sCreated %s with sample responses.%s\n", ColorGreen, mdFile, ColorReset)

	// 3. Seed Records with Batched Inserts and Transaction
	categories := []string{"food", "housing", "transport", "entertainment", "utilities", "salary", "investment", "shopping", "health"}
	incomeCategories := []string{"salary", "investment"}

	now := time.Now()
	startDate := now.AddDate(-2, 0, 0)
	adminID := seededUsers[0].ID

	fmt.Printf("%sSeeding records from %s to %s...%s\n", ColorCyan, startDate.Format("2006-01-02"), now.Format("2006-01-02"), ColorReset)

	recordTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Fatalf("%sError starting record transaction: %v%s\n", ColorRed, err, ColorReset)
	}

	totalRecords := 0
	batchSize := 100
	var batch []any
	
	// Helper to execute batch insert
	flushBatch := func(data []any) {
		if len(data) == 0 {
			return
		}
		numFields := 6
		numRows := len(data) / numFields
		placeholderParts := make([]string, numRows)
		for i := 0; i < numRows; i++ {
			base := i * numFields
			placeholderParts[i] = fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)", 
				base+1, base+2, base+3, base+4, base+5, base+6)
		}
		
		query := fmt.Sprintf("INSERT INTO records (user_id, amount, txn_type, category, description, date) VALUES %s", 
			strings.Join(placeholderParts, ","))
		
		_, err := recordTx.ExecContext(ctx, query, data...)
		if err != nil {
			recordTx.Rollback()
			log.Fatalf("%sError executing batch insert: %v%s\n", ColorRed, err, ColorReset)
		}
	}

	for d := startDate; d.Before(now); d = d.AddDate(0, 1, 0) {
		recordCount := 30 + rand.Intn(11) // 30-40 records per month
		fmt.Printf("%sMonth: %s - Generating %d records%s\n", ColorYellow, d.Format("2006-01"), recordCount, ColorReset)

		for range recordCount {
			day := rand.Intn(28) + 1
			txnDate := time.Date(d.Year(), d.Month(), day, 0, 0, 0, 0, time.UTC)
			if txnDate.After(now) {
				txnDate = now
			}

			category := categories[rand.Intn(len(categories))]
			isIncome := false
			for _, ic := range incomeCategories {
				if ic == category {
					isIncome = true
					break
				}
			}

			txnType := "expense"
			amount := decimal.NewFromFloat(10.0 + rand.Float64()*100.0)
			if isIncome {
				txnType = "income"
				amount = decimal.NewFromFloat(500.0 + rand.Float64()*2000.0)
			}

			description := fmt.Sprintf("Sample %s for %s", txnType, category)
			
			batch = append(batch, adminID, amount, txnType, category, description, txnDate)
			totalRecords++

			if len(batch)/6 >= batchSize {
				flushBatch(batch)
				batch = nil
			}
		}
	}

	// Final flush
	flushBatch(batch)

	if err = recordTx.Commit(); err != nil {
		log.Fatalf("%sError committing record transaction: %v%s\n", ColorRed, err, ColorReset)
	}

	fmt.Printf("%sSeeding complete! Total records inserted: %d%s\n", ColorGreen, totalRecords, ColorReset)
}
