package database

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type TxnType string

const (
	TxnTypeIncome  TxnType = "income"
	TxnTypeExpense TxnType = "expense"
)

type Record struct {
	ID          int64           `db:"id" json:"id"`
	UserID      string          `db:"user_id" json:"user_id"`
	Amount      decimal.Decimal `db:"amount" json:"amount"`
	TxnType     TxnType         `db:"txn_type" json:"txn_type"`
	Category    string          `db:"category" json:"category"`
	Description *string         `db:"description" json:"description"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   *time.Time      `db:"updated_at" json:"updated_at,omitempty"`
	DeletedAt   *time.Time      `db:"deleted_at" json:"deleted_at,omitempty"`
}

type RecordFilters struct {
	TxnType     string `json:"txn_type"`
	Category    string `json:"category"`
	From        time.Time `json:"from"`
	To          time.Time `json:"to"`
	ShowDeleted bool	`json:"show_deleted"`
	Ascending   bool	`json:"ascending"`
}

func (s *service) GetRecords(ctx context.Context, filters *RecordFilters) ([]Record, error) {
	query := `SELECT id, user_id, amount, 
						 txn_type, category, description,
						 created_at, updated_at, deleted_at
						FROM records
						WHERE 1=1 
						`
	args := []any{}

	i := 1

	if filters.TxnType != "" {
		query += fmt.Sprintf(" AND txn_type = $%d", i)
		args = append(args, filters.TxnType)
		i++
	}

	if filters.Category != "" {
		query += fmt.Sprintf(" AND category = $%d", i)
		args = append(args, filters.Category)
		i++
	}

	if !filters.From.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d", i)
		args = append(args, filters.From)
		i++
	}

	if !filters.To.IsZero() {
		query += fmt.Sprintf(" AND created_at <= $%d", i)
		args = append(args, filters.To)
		i++
	}

	if !filters.ShowDeleted {
		query += " AND deleted_at IS NULL"
	}

	if !filters.Ascending {
		query += " ORDER BY created_at DESC"
	} else {
		query += " ORDER BY created_at ASC"
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		err := rows.Scan(
			&r.ID, &r.UserID, &r.Amount,
			&r.TxnType, &r.Category, &r.Description,
			&r.CreatedAt, &r.UpdatedAt, &r.DeletedAt,
		)

		if err != nil {
			return nil, err
		}

		records = append(records, r)
	}

	return records, nil
}

func (s *service) CreateRecord(ctx context.Context, userID string, record Record) (int64, error) {
	query := `INSERT INTO records (user_id, amount, txn_type, category, description)
						VALUES ($1, $2, $3, $4, $5)`

	var id int64

	err := s.db.QueryRowContext(ctx, query,
		record.UserID,
		record.Amount,
		record.TxnType,
		record.Category,
		record.Description,
	).Scan(&id)

	if err != nil {
		return -1, err
	}

	return id, nil
}

func (s *service) UpdateRecord(ctx context.Context, id int64, updates Record) error {
	query := `UPDATE records SET updated_at = now()`
	args := []any{}
	i := 1

	if updates.Amount.IsPositive() {
		query += fmt.Sprintf(", amount = $%d", i)
		args = append(args, updates.Amount)
		i++
	}

	if updates.TxnType != "" {
		query += fmt.Sprintf(", txn_type = $%d", i)
		args = append(args, updates.TxnType)
		i++
	}

	if updates.Category != "" {
		query += fmt.Sprintf(", category = $%d", i)
		args = append(args, updates.Category)
		i++
	}

	if updates.Description != nil {
		query += fmt.Sprintf(", description = $%d", i)
		args = append(args, updates.Description)
		i++
	}

	query += fmt.Sprintf(" WHERE id = $%d AND deleted_at IS NULL", i)
	args = append(args, id)

	_, err := s.db.ExecContext(ctx, query, args)
	return err
}

func (s *service) DeleteRecord(ctx context.Context, id int64) error {
	query := `UPDATE records SET deleted_at = now() 
						WHERE id = $1`

	// simply set deleted_at timestamp
	_, err := s.db.ExecContext(ctx, query, id)

	return err
}
