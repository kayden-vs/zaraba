package models

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kayden-vs/zaraba/internal/engine"
	"github.com/lib/pq"
)

const (
	OrderStatusOpen            = "OPEN"
	OrderStatusPartiallyFilled = "PARTIALLY_FILLED"
	OrderStatusFilled          = "FILLED"
	OrderStatusCancelled       = "CANCELLED"

	OrderTypeLimit  = "LIMIT"
	OrderTypeMarket = "MARKET"

	OrderSideBuy  = "BUY"
	OrderSideSell = "SELL"
)

type OrderModelInterface interface {
	EnsureSchema() error
	Create(order *Order) (int64, error)
	DeleteByEngineOrderID(userID, engineOrderID int64) error
	UpdateExecution(orderID, filled, filledNotional int64, status string) error
	MarkCancelled(orderID int64) error
	GetByIDForUser(orderID, userID int64) (*Order, error)
	ListByUser(userID int64, statuses []string, limit int) ([]*Order, error)
	ListOpenByUser(userID int64) ([]*Order, error)
}

type Order struct {
	ID             int64
	EngineOrderID  int64
	UserID         int64
	Symbol         string
	Side           string
	OrderType      string
	Price          int64
	Quantity       int64
	Filled         int64
	FilledNotional int64
	AvgFillPrice   int64
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ClosedAt       sql.NullTime
}

func (o *Order) Remaining() int64 {
	remaining := o.Quantity - o.Filled
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (o *Order) IsBuy() bool {
	return strings.EqualFold(o.Side, OrderSideBuy)
}

type OrderModel struct {
	DB *sql.DB
}

func (m *OrderModel) EnsureSchema() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS orders (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			engine_order_id BIGINT,
			symbol VARCHAR(20) NOT NULL,
			side VARCHAR(4) NOT NULL,
			order_type VARCHAR(16) NOT NULL DEFAULT 'LIMIT',
			price BIGINT NOT NULL,
			quantity BIGINT NOT NULL,
			filled BIGINT NOT NULL DEFAULT 0,
			filled_notional BIGINT NOT NULL DEFAULT 0,
			avg_fill_price BIGINT NOT NULL DEFAULT 0,
			status VARCHAR(20) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT now(),
			updated_at TIMESTAMP NOT NULL DEFAULT now(),
			closed_at TIMESTAMP,
			CONSTRAINT fk_orders_user
				FOREIGN KEY (user_id)
				REFERENCES users(id)
				ON DELETE CASCADE
		);`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS engine_order_id BIGINT;`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS order_type VARCHAR(16) NOT NULL DEFAULT 'LIMIT';`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS filled_notional BIGINT NOT NULL DEFAULT 0;`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS avg_fill_price BIGINT NOT NULL DEFAULT 0;`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT now();`,
		`ALTER TABLE orders ADD COLUMN IF NOT EXISTS closed_at TIMESTAMP;`,
		`UPDATE orders SET order_type = 'LIMIT' WHERE order_type IS NULL;`,
		`UPDATE orders SET updated_at = created_at WHERE updated_at IS NULL;`,
		`CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_orders_symbol_status ON orders(symbol, status);`,
		`CREATE INDEX IF NOT EXISTS idx_orders_user_status_created ON orders(user_id, status, created_at DESC);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_engine_order_id ON orders(engine_order_id) WHERE engine_order_id IS NOT NULL;`,
	}

	for _, stmt := range stmts {
		if _, err := m.DB.Exec(stmt); err != nil {
			return fmt.Errorf("ensure orders schema: %w", err)
		}
	}

	return nil
}

func (m *OrderModel) Create(order *Order) (int64, error) {
	if order == nil {
		return 0, fmt.Errorf("nil order")
	}

	stmt := `INSERT INTO orders (
		user_id,
		engine_order_id,
		symbol,
		side,
		order_type,
		price,
		quantity,
		filled,
		filled_notional,
		avg_fill_price,
		status,
		created_at,
		updated_at,
		closed_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW(), $12)
	RETURNING id`

	avgFillPrice := order.AvgFillPrice
	if avgFillPrice == 0 && order.Filled > 0 && order.FilledNotional > 0 {
		avgFillPrice = averageFillPrice(order.FilledNotional, order.Filled)
	}

	status := normalizeStatus(order.Status)
	side := normalizeSide(order.Side)
	orderType := normalizeOrderType(order.OrderType)

	var closedAt any
	if status == OrderStatusFilled || status == OrderStatusCancelled {
		closedAt = time.Now()
	}

	var id int64
	err := m.DB.QueryRow(
		stmt,
		order.UserID,
		nullIfZero(order.EngineOrderID),
		strings.TrimSpace(strings.ToLower(order.Symbol)),
		side,
		orderType,
		order.Price,
		order.Quantity,
		order.Filled,
		order.FilledNotional,
		avgFillPrice,
		status,
		closedAt,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (m *OrderModel) DeleteByEngineOrderID(userID, engineOrderID int64) error {
	if engineOrderID == 0 {
		return nil
	}

	stmt := `DELETE FROM orders WHERE user_id = $1 AND engine_order_id = $2`
	_, err := m.DB.Exec(stmt, userID, engineOrderID)
	return err
}

func (m *OrderModel) UpdateExecution(orderID, filled, filledNotional int64, status string) error {
	status = normalizeStatus(status)
	avgFillPrice := int64(0)
	if filled > 0 && filledNotional > 0 {
		avgFillPrice = averageFillPrice(filledNotional, filled)
	}

	closeNow := status == OrderStatusFilled || status == OrderStatusCancelled

	stmt := `UPDATE orders
		SET
			filled = $1,
			filled_notional = $2,
			avg_fill_price = $3,
			status = $4,
			updated_at = NOW(),
			closed_at = CASE WHEN $6 THEN NOW() ELSE NULL END
		WHERE id = $5`

	_, err := m.DB.Exec(stmt, filled, filledNotional, avgFillPrice, status, orderID, closeNow)
	return err
}

func (m *OrderModel) MarkCancelled(orderID int64) error {
	stmt := `UPDATE orders
		SET status = 'CANCELLED', updated_at = NOW(), closed_at = NOW()
		WHERE id = $1`
	_, err := m.DB.Exec(stmt, orderID)
	return err
}

func (m *OrderModel) GetByIDForUser(orderID, userID int64) (*Order, error) {
	stmt := `SELECT
		id,
		COALESCE(engine_order_id, 0),
		user_id,
		symbol,
		side,
		order_type,
		price,
		quantity,
		filled,
		filled_notional,
		avg_fill_price,
		status,
		created_at,
		updated_at,
		closed_at
	FROM orders
	WHERE id = $1 AND user_id = $2`

	order := &Order{}
	err := m.DB.QueryRow(stmt, orderID, userID).Scan(
		&order.ID,
		&order.EngineOrderID,
		&order.UserID,
		&order.Symbol,
		&order.Side,
		&order.OrderType,
		&order.Price,
		&order.Quantity,
		&order.Filled,
		&order.FilledNotional,
		&order.AvgFillPrice,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.ClosedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		}
		return nil, err
	}

	return order, nil
}

func (m *OrderModel) ListByUser(userID int64, statuses []string, limit int) ([]*Order, error) {
	if limit <= 0 {
		limit = 200
	}

	normalizedStatuses := make([]string, 0, len(statuses))
	for _, status := range statuses {
		normalizedStatuses = append(normalizedStatuses, normalizeStatus(status))
	}

	base := `SELECT
		id,
		COALESCE(engine_order_id, 0),
		user_id,
		symbol,
		side,
		order_type,
		price,
		quantity,
		filled,
		filled_notional,
		avg_fill_price,
		status,
		created_at,
		updated_at,
		closed_at
	FROM orders
	WHERE user_id = $1`

	args := []any{userID}
	if len(normalizedStatuses) > 0 {
		base += ` AND status = ANY($2)`
		base += ` ORDER BY created_at DESC LIMIT $3`
		args = append(args, pq.Array(normalizedStatuses), limit)
	} else {
		base += ` ORDER BY created_at DESC LIMIT $2`
		args = append(args, limit)
	}

	rows, err := m.DB.Query(base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []*Order{}
	for rows.Next() {
		order := &Order{}
		err = rows.Scan(
			&order.ID,
			&order.EngineOrderID,
			&order.UserID,
			&order.Symbol,
			&order.Side,
			&order.OrderType,
			&order.Price,
			&order.Quantity,
			&order.Filled,
			&order.FilledNotional,
			&order.AvgFillPrice,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
			&order.ClosedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (m *OrderModel) ListOpenByUser(userID int64) ([]*Order, error) {
	return m.ListByUser(userID, []string{OrderStatusOpen, OrderStatusPartiallyFilled}, 500)
}

func averageFillPrice(filledNotional, filledQty int64) int64 {
	if filledNotional <= 0 || filledQty <= 0 {
		return 0
	}

	price := (float64(filledNotional) / float64(filledQty)) * float64(engine.QuantityScale)
	return int64(math.Round(price))
}

func nullIfZero(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func normalizeStatus(status string) string {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	switch normalized {
	case OrderStatusOpen, OrderStatusPartiallyFilled, OrderStatusFilled, OrderStatusCancelled:
		return normalized
	default:
		return OrderStatusOpen
	}
}

func normalizeSide(side string) string {
	normalized := strings.ToUpper(strings.TrimSpace(side))
	if normalized == OrderSideSell {
		return OrderSideSell
	}
	return OrderSideBuy
}

func normalizeOrderType(orderType string) string {
	normalized := strings.ToUpper(strings.TrimSpace(orderType))
	if normalized == OrderTypeMarket {
		return OrderTypeMarket
	}
	return OrderTypeLimit
}
