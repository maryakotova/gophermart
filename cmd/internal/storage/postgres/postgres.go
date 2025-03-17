package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgerrcode"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/maryakotova/gophermart/internal/config"
	"github.com/maryakotova/gophermart/internal/models"
	"go.uber.org/zap"
)

type PostgresStorage struct {
	db     *sql.DB
	config *config.Config
	logger *zap.Logger
	mtx    sync.RWMutex
}

// ------------------------------------------------------------------------------------
// нужно ли при таком создании конекшен вызывать defer db.Close(), если да, то где?
// ------------------------------------------------------------------------------------
func NewPostgresStorage(cfg *config.Config, logger *zap.Logger) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", cfg.DatabaseURI)
	if err != nil {
		// можно ли тут вызывать панику? ----------------------------------------
		err = fmt.Errorf("не удалось подключиться к бд: %w", err)
		logger.Error(err.Error())
		return nil, err
	}
	return &PostgresStorage{
		db:     db,
		config: cfg,
		logger: logger,
	}, nil
}

func (ps *PostgresStorage) Bootstrap(ctx context.Context) error {

	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
	CREATE TABLE IF NOT EXISTS users (
		user_id SERIAL PRIMARY KEY,
		user_name VARCHAR(50) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL
	);
	`

	_, err = tx.ExecContext(ctx, query)
	if err != nil {
		ps.logger.Error(err.Error())
		return err
	}

	query = `
	CREATE TABLE IF NOT EXISTS orders (
		order_num BIGINT PRIMARY KEY,
		user_id INT NOT NULL,
		status VARCHAR(10) NOT NULL,
		uploaded_at TIMESTAMP NOT NULL,
		points DOUBLE PRECISION,
		FOREIGN KEY (user_id) REFERENCES users(user_id)
	);
	`

	_, err = tx.ExecContext(ctx, query)
	if err != nil {
		ps.logger.Error(err.Error())
		return err
	}

	query = `
	CREATE TABLE IF NOT EXISTS withdrawals (
		order_num BIGINT PRIMARY KEY,
		user_id INT NOT NULL,
		processed_at TIMESTAMP NOT NULL,
		points DOUBLE PRECISION,
		FOREIGN KEY (user_id) REFERENCES users(user_id)
	);
	`

	_, err = tx.ExecContext(ctx, query)
	if err != nil {
		ps.logger.Error(err.Error())
		return err
	}

	query = `
	CREATE TABLE IF NOT EXISTS balance (
		user_id INT PRIMARY KEY,
		sum DOUBLE PRECISION NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(user_id)
	);
	`

	_, err = tx.ExecContext(ctx, query)
	if err != nil {
		ps.logger.Error(err.Error())
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("error creating tables: %v", err)
	}

	return nil
}

func (ps *PostgresStorage) GetUserID(ctx context.Context, userName string) (userID int, err error) {

	query := `
	SELECT user_id 
		FROM users 
		WHERE user_name = $1;
	`
	ps.mtx.Lock()
	err = ps.db.QueryRowContext(ctx, query, userName).Scan(userID)
	ps.mtx.Unlock()
	if err != nil {
		return -1, err
	}

	if userID == 0 {
		return -1, nil
	}

	return
}

func (ps *PostgresStorage) CreateUser(ctx context.Context, login string, hashedPassword string) (userID int, err error) {

	query := `
	INSERT INTO users (user_name, password)
		VALUES ($1, $2)
		RETURNING user_id;
	`

	ps.mtx.Lock()
	err = ps.db.QueryRowContext(ctx, query, login, hashedPassword).Scan(userID)
	ps.mtx.Unlock()
	if err != nil {
		return -1, err
	}

	return
}

func (ps *PostgresStorage) GetUserAuthData(ctx context.Context, login string) (userID int, hashedPassword string, err error) {

	query := `
	SELECT user_id, password 
		FROM users 
		WHERE user_name = $1;
	`
	ps.mtx.Lock()
	err = ps.db.QueryRowContext(ctx, query, login).Scan(userID, hashedPassword)
	ps.mtx.Unlock()
	if err != nil {
		return -1, "", err
	}

	return
}

func (ps *PostgresStorage) GetUserByOrderNum(ctx context.Context, orderNumber int64) (userID int, err error) {

	query := `
	SELECT user_id 
		FROM orders 
		WHERE order_num = $1;
	`

	ps.mtx.Lock()
	err = ps.db.QueryRowContext(ctx, query, orderNumber).Scan(userID)
	ps.mtx.Unlock()
	if err != nil {
		return -1, err
	}

	return
}

func (ps *PostgresStorage) InsertOrder(ctx context.Context, userID int, accrualResponce models.AccrualSystemResponce) error {

	query := `
	INSERT INTO orders (order_num, user_id, status, uploaded_at, points)
		VALUES ($1, $2, $3, $4, $5);
	`

	ps.mtx.Lock()
	_, err := ps.db.ExecContext(ctx, query, accrualResponce.Order, userID, accrualResponce.Status, time.Now(), accrualResponce.Accrual)
	ps.mtx.Unlock()
	if err != nil {
		return err
	}

	return nil
}

func (ps *PostgresStorage) UpdateOrder(ctx context.Context, accrualResponce models.AccrualSystemResponce) error {

	queryWoPoints := `
	UPDATE orders 
		SET status = $1
		WHERE order_num = $2;
	`

	queryWPoints := `
	UPDATE orders 
		SET status = $1, points = $2
		WHERE order_num = $3;
	`
	var err error

	ps.mtx.Lock()
	if accrualResponce.Accrual > 0 {
		_, err = ps.db.ExecContext(ctx, queryWPoints, accrualResponce.Status, accrualResponce.Accrual, accrualResponce.Order)
	} else {
		_, err = ps.db.ExecContext(ctx, queryWoPoints, accrualResponce.Status, accrualResponce.Order)
	}
	ps.mtx.Unlock()

	if err != nil {
		return err
	}

	return nil
}

func (ps *PostgresStorage) GetOrdersForUser(ctx context.Context, userID int) (orders []models.OrderList, err error) {

	query := `
	SELECT order_num, status, uploaded_at, points
		FROM orders
		WHERE iser_id = $1
		ORDER BY uploaded_at DESC;
	`
	ps.mtx.Lock()
	rows, err := ps.db.QueryContext(ctx, query, userID)
	ps.mtx.Unlock()
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var order models.OrderList
		err := rows.Scan(&order.OrderNumber, &order.Status, &order.UploadedAt, &order.Accrual)
		if err != nil {
			err = fmt.Errorf("ошибка при считывании строки: %w", err)
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (ps *PostgresStorage) UpdateBalance(ctx context.Context, userID int, points float64) error {

	query := `
	UPDATE balance
		SET sum = $1
		WHERE user_id = S2;
	`
	ps.mtx.Lock()
	_, err := ps.db.ExecContext(ctx, query, points, userID)
	ps.mtx.Unlock()
	if err != nil {
		return err
	}

	return nil
}

func (ps *PostgresStorage) GetCurrentBalance(ctx context.Context, userID int) (balance float64, err error) {

	query := `
	SELECT sum 
		FROM balance 
		WHERE user_name = $1;
	`
	ps.mtx.Lock()
	err = ps.db.QueryRowContext(ctx, query, userID).Scan(balance)
	ps.mtx.Unlock()
	if err != nil {
		return 0, err
	}

	return balance, nil
}

func (ps *PostgresStorage) GetWithdrawalSum(ctx context.Context, userID int) (withdrawalSum float64, err error) {

	query := ` 
	SELECT SUM(points) AS total_points
		FROM withdrawals
		WHERE user_id = $1;
	`

	ps.mtx.Lock()
	err = ps.db.QueryRowContext(ctx, query, userID).Scan(withdrawalSum)
	ps.mtx.Unlock()
	if err != nil {
		return 0, err
	}

	return withdrawalSum, nil
}

func (ps *PostgresStorage) IncreaseBalance(ctx context.Context, userID int, points float64) error {

	query := `
	INSERT INTO balance (user_id, sum)
		VALUES ($1, $2)
		ON CONFLICT (user_id) 
		DO UPDATE SET sum = balance.sum + EXCLUDED.sum;
		`
	_, err := ps.db.ExecContext(ctx, query, userID, points)
	if err != nil {
		return err
	}

	return nil
}

func (ps *PostgresStorage) InsertWithdrawal(ctx context.Context, userID int, orderNumber int64, points float64) error {

	query := `
	INSERT INTO orders (order_num, user_id, processed_at, points)
		VALUES ($1, $2, $3, $4);
	`

	ps.mtx.Lock()
	_, err := ps.db.ExecContext(ctx, query, orderNumber, userID, time.Now(), points)
	ps.mtx.Unlock()
	if err != nil {
		return err
	}

	return nil

}

func (ps *PostgresStorage) GetWithdrawalsForUser(ctx context.Context, userID int) (withdrawals []models.Withdrawals, err error) {

	query := `
	SELECT order_num, points, processed_at
		FROM withdrawals
		WHERE iser_id = $1
		ORDER BY processed_at DESC;
	`
	ps.mtx.Lock()
	rows, err := ps.db.QueryContext(ctx, query, userID)
	ps.mtx.Unlock()
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var withdrawal models.Withdrawals
		err := rows.Scan(&withdrawal.OrderNumber, &withdrawal.ProcessedAt, &withdrawal.Sum)
		if err != nil {
			err = fmt.Errorf("ошибка при считывании строки: %w", err)
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}

	return withdrawals, nil
}
