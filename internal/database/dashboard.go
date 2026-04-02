package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

type PeriodType string

const (
	PeriodWeekly  PeriodType = "day"
	PeriodMonthly PeriodType = "month"
	PeriodYearly  PeriodType = "year"
	PeriodAllTime PeriodType = "quarter"
)

type BalanceTotals struct {
	TotalIncome  decimal.Decimal
	TotalExpense decimal.Decimal
	NetBalance   decimal.Decimal
}

type CategoryTotals struct {
	Category string          `json:"category"`
	Income   decimal.Decimal `json:"income"`
	Expense  decimal.Decimal `json:"expense"`
}

type TrendsEntry struct {
	Date    time.Time       `json:"date"`
	Income  decimal.Decimal `json:"income"`
	Expense decimal.Decimal `json:"expense"`
}

type DashboardSummary struct {
	Period           PeriodType       `json:"period"`
	TotalIncome      decimal.Decimal  `json:"total_income"`
	TotalExpense     decimal.Decimal  `json:"total_expense"`
	NetBalance       decimal.Decimal  `json:"net_balance"`
	CategoryTotals   []CategoryTotals `json:"category_totals"`
	Trends           []TrendsEntry    `json:"trends_entry"`
	RecentActivities []Record         `json:"recent_activities"`
}

func whereClauseBuilder(from, to time.Time) ([]string, []any) {
	var whereClauses []string
	var args []any

	whereClauses = append(whereClauses, "deleted_at IS NULL")

	if !from.IsZero() {
		args = append(args, from)
		whereClauses = append(whereClauses, fmt.Sprintf("created_at >= $%d", len(args)))
	}

	if !to.IsZero() {
		args = append(args, to)
		whereClauses = append(whereClauses, fmt.Sprintf("created_at <= $%d", len(args)))
	}

	return whereClauses, args
}

func (s *service) fetchBalanceTotals(ctx context.Context, from, to time.Time) (BalanceTotals, error) {
	whereClauses, args := whereClauseBuilder(from, to)
	query := fmt.Sprintf(`SELECT
							COALESCE(SUM(amount) FILTER (WHERE txn_type = 'income'), 0) AS total_income,
							COALESCE(SUM(amount) FILTER (WHERE txn_type = 'expense'), 0) AS total_expense,
							COALESCE(SUM(amount) FILTER (WHERE txn_type = 'income'), 0) -
							COALESCE(SUM(amount) FILTER (WHERE txn_type = 'expense'), 0) AS net_balance
						FROM records
						WHERE %s`, strings.Join(whereClauses, " AND "))

	var balanceTotal BalanceTotals
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&balanceTotal.TotalIncome,
		&balanceTotal.TotalExpense,
		&balanceTotal.NetBalance,
	)

	return balanceTotal, err
}

func (s *service) fetchCategoryTotals(ctx context.Context, from time.Time, to time.Time) ([]CategoryTotals, error) {
	whereClauses, args := whereClauseBuilder(from, to)
	query := fmt.Sprintf(`SELECT 
		category,
		COALESCE(SUM(amount) FILTER (WHERE txn_type = 'income'), 0) AS income,
		COALESCE(SUM(amount) FILTER (WHERE txn_type = 'expense'), 0) AS expense
		FROM records
		WHERE %s
		GROUP BY category
	`, strings.Join(whereClauses, " AND "))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categoryTotal []CategoryTotals
	for rows.Next() {
		var category CategoryTotals
		err := rows.Scan(&category.Category, &category.Income, &category.Expense)
		if err != nil {
			return nil, err
		}

		categoryTotal = append(categoryTotal, category)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return categoryTotal, nil
}

func (s *service) fetchTrends(
	ctx context.Context,
	from time.Time,
	to time.Time,
	period PeriodType,
) ([]TrendsEntry, error) {
	whereClauses, args := whereClauseBuilder(from, to)
	query := fmt.Sprintf(`SELECT 
			DATE_TRUNC('%s', created_at) AS date,
			COALESCE(SUM(amount) FILTER (WHERE txn_type = 'income'), 0) AS income,
			COALESCE(SUM(amount) FILTER(WHERE txn_type = 'expense'), 0) AS expense
		FROM records
		WHERE %s
		GROUP BY date
		ORDER BY date ASC`, period, strings.Join(whereClauses, " AND "))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var trendEntries []TrendsEntry

	for rows.Next() {
		var trendEntry TrendsEntry
		err = rows.Scan(&trendEntry.Date, &trendEntry.Income, &trendEntry.Expense)
		if err != nil {
			return nil, err
		}
		trendEntries = append(trendEntries, trendEntry)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return trendEntries, nil
}

func (s *service) fetchRecentActivities(ctx context.Context) ([]Record, error) {
	query := `SELECT id, amount, txn_type, category, created_at, description
		FROM records
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 10
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recentActivities []Record
	for rows.Next() {
		var recentActivity Record
		err = rows.Scan(
			&recentActivity.ID, &recentActivity.Amount, &recentActivity.TxnType,
			&recentActivity.Category, &recentActivity.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		recentActivities = append(recentActivities, recentActivity)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return recentActivities, nil
}

func (s *service) GetDashboardSummary(ctx context.Context, period PeriodType) (DashboardSummary, error) {
	to := time.Now()
	var from time.Time

	switch period {
	case PeriodWeekly:
		from = to.AddDate(0, 0, -7)
	case PeriodMonthly:
		from = to.AddDate(0, -1, 0)
	case PeriodYearly:
		from = to.AddDate(-1, 0, 0)
	case PeriodAllTime:
		from = time.Time{}
		to = time.Time{}
	}
	g, ctx := errgroup.WithContext(ctx)
	var balanceTotals BalanceTotals
	var categoryTotals []CategoryTotals
	var trendsEntries []TrendsEntry
	var recentActivities []Record

	g.Go(func() error {
		result, err := s.fetchBalanceTotals(ctx, from, to)
		balanceTotals = result
		return err
	})

	g.Go(func() error {
		result, err := s.fetchCategoryTotals(ctx, from, to)
		categoryTotals = result
		return err
	})

	g.Go(func() error {
		result, err := s.fetchTrends(ctx, from, to, period)
		trendsEntries = result
		return err
	})

	g.Go(func() error {
		result, err := s.fetchRecentActivities(ctx)
		recentActivities = result
		return err
	})

	if err := g.Wait(); err != nil {
		return DashboardSummary{}, err
	}

	dashboardSummary := DashboardSummary{
		Period:           period,
		TotalIncome:      balanceTotals.TotalIncome,
		TotalExpense:     balanceTotals.TotalExpense,
		NetBalance:       balanceTotals.NetBalance,
		CategoryTotals:   categoryTotals,
		Trends:           trendsEntries,
		RecentActivities: recentActivities,
	}
	return dashboardSummary, nil
}
