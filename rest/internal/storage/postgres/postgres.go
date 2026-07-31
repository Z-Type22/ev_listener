package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"rest/internal/config"
	"rest/internal/domain/models"
	"rest/internal/storage"
	"time"

	_ "github.com/lib/pq"
)

type Storage struct {
	db  *sql.DB
	log *slog.Logger
}

func New(database config.Database, log *slog.Logger) *Storage {
	const op = "storage.postgres.New"

	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		database.Host,
		database.Port,
		database.User,
		database.Password,
		database.Name,
		database.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	return &Storage{
		db:  db,
		log: log,
	}
}

func (s *Storage) Check(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Storage) Close() {
	if err := s.db.Close(); err != nil {
		s.log.Error("failed to close database connection", slog.Any("error", err))

		return
	}

	s.log.Info("database connection closed")
}

func (s *Storage) GetTransaction(ctx context.Context, id int, wallet string) (models.Transaction, error) {
	const op = "storage.postgres.GetTransaction"

	stmt, err := s.db.Prepare("SELECT * FROM transactions WHERE id = $1 AND merchant_wallet = $2")
	if err != nil {
		return models.Transaction{}, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var transaction models.Transaction
	var payload []byte

	err = stmt.QueryRowContext(ctx, id, wallet).Scan(
		&transaction.ID,
		&transaction.TxHash,
		&transaction.UserWallet,
		&transaction.MerchantWallet,
		&transaction.Price,
		&transaction.Type,
		&payload,
		&transaction.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Transaction{}, storage.ErrTransactionNotFound
		}
		return models.Transaction{}, fmt.Errorf("%s: %w", op, err)
	}

	if err := json.Unmarshal(payload, &transaction.Payload); err != nil {
		return models.Transaction{}, fmt.Errorf("%s: %w", op, err)
	}

	return transaction, nil
}

func (s *Storage) GetListTransactions(ctx context.Context, wallet string) ([]models.Transaction, error) {
	const op = "storage.postgres.GetListTransactions"

	stmt, err := s.db.Prepare("SELECT * FROM transactions WHERE merchant_wallet = $1")
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, wallet)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var transactions []models.Transaction

	for rows.Next() {
		var transaction models.Transaction
		var payload []byte

		err = rows.Scan(
			&transaction.ID,
			&transaction.TxHash,
			&transaction.UserWallet,
			&transaction.MerchantWallet,
			&transaction.Price,
			&transaction.Type,
			&payload,
			&transaction.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		if err := json.Unmarshal(payload, &transaction.Payload); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return transactions, nil
}

func (s *Storage) SetTransactionDonate(ctx context.Context, title string, event models.DonateEvent) (int, error) {
	const op = "storage.postgres.SetTransactionDonate"

	query := `
		INSERT INTO transactions (
			tx_hash, user_wallet, merchant_wallet, price, type, payload
		)
		VALUES ($1, $2, $3, $4, $5, '{}'::jsonb)
		RETURNING id
	`

	var lastInsertId int

	err := s.db.QueryRowContext(
		ctx, query, event.TxHash, event.RecipientAddress, event.MerchantAddress, event.Amount, title,
	).Scan(&lastInsertId)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return lastInsertId, nil
}

func (s *Storage) SetTransactionSubscriptionActivated(
	ctx context.Context, title string, event models.ActivatedEvent,
) (int, error) {
	const op = "storage.postgres.SetTransactionSubscriptionActivated"

	query, lastInsertId := s.HelperTransaction()

	payload := struct {
		PlanHash     string       `json:"plan_hash"`
		Plan         *models.Plan `json:"plan"`
		NextChargeAt time.Time    `json:"next_charge_at"`
	}{
		PlanHash:     event.PlanHash,
		Plan:         event.Plan,
		NextChargeAt: event.NextChargeAt,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("%s: marshal payload: %w", op, err)
	}

	err = s.db.QueryRowContext(
		ctx, query, event.TxHash, event.UserAddress, event.MerchantAddress, event.Price, title, payloadJSON,
	).Scan(&lastInsertId)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return lastInsertId, nil
}

func (s *Storage) SetTransactionSubscriptionCharged(
	ctx context.Context, title string, event models.ChargedEvent,
) (int, error) {
	const op = "storage.postgres.SetTransactionSubscriptionCharged"

	query, lastInsertId := s.HelperTransaction()

	payload := struct {
		NextChargeAt time.Time `json:"next_charge_at"`
	}{NextChargeAt: event.NextChargeAt}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("%s: marshal payload: %w", op, err)
	}

	err = s.db.QueryRowContext(
		ctx, query, event.TxHash, event.UserAddress, event.MerchantAddress, event.Price, title, payloadJSON,
	).Scan(&lastInsertId)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return lastInsertId, nil
}

func (s *Storage) SetTransactionSubscriptionCancelled(
	ctx context.Context, title string, event models.DeactivatedEvent,
) (int, error) {
	const op = "storage.postgres.SetTransactionSubscriptionCancelled"

	query, lastInsertId := s.HelperTransaction()

	payload := struct {
		Reason string `json:"reason"`
	}{Reason: event.Reason}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("%s: marshal payload: %w", op, err)
	}

	err = s.db.QueryRowContext(
		ctx, query, event.TxHash, event.UserAddress, event.MerchantAddress, event.Price, title, payloadJSON,
	).Scan(&lastInsertId)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return lastInsertId, nil
}

func (s *Storage) SetTransactionSubscriptionFailedFinal(
	ctx context.Context, title string, event models.DeactivatedEvent,
) (int, error) {
	const op = "storage.postgres.SetTransactionSubscriptionFailedFinal"

	query, lastInsertId := s.HelperTransaction()

	payload := struct {
		Reason string `json:"reason"`
	}{Reason: event.Reason}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("%s: marshal payload: %w", op, err)
	}

	err = s.db.QueryRowContext(
		ctx, query, event.TxHash, event.UserAddress, event.MerchantAddress, event.Price, title, payloadJSON,
	).Scan(&lastInsertId)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return lastInsertId, nil
}

func (s *Storage) SetTransactionSubscriptionRetryScheduled(
	ctx context.Context, title string, event models.RetryScheduledEvent,
) (int, error) {
	const op = "storage.postgres.SetTransactionSubscriptionRetryScheduled"

	query, lastInsertId := s.HelperTransaction()

	payload := struct {
		RetryAt time.Time `json:"retry_at"`
	}{RetryAt: event.RetryAt}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("%s: marshal payload: %w", op, err)
	}

	err = s.db.QueryRowContext(
		ctx, query, event.TxHash, event.UserAddress, event.MerchantAddress, event.Price, title, payloadJSON,
	).Scan(&lastInsertId)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return lastInsertId, nil
}

func (s *Storage) HelperTransaction() (string, int) {
	query := `
		INSERT INTO transactions (
			tx_hash, user_wallet, merchant_wallet, price, type, payload
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var lastInsertId int

	return query, lastInsertId
}
