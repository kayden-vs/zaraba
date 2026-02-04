package models

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/kayden-vs/zaraba/internal/engine"
	_ "github.com/lib/pq"
)

type WalletModelInterface interface {
	CreditWallet(userID int64, amount int64) (*Wallet, error)
	DebitWallet(userID int64, amount int64) (*Wallet, error)
	LocketAmount(userID, amount int64) (*Wallet, error)
	UnlockAmount(userID, amount int64) (*Wallet, error)
	GetBalance(userID int64) (int64, error)
	GetTotalBalance(userID int64) (int64, error)
}

type Wallet struct {
	UserID    int64
	Balance   int64
	Locked    int64 // for open orders
	UpdatedAt time.Time
}

type WalletModel struct {
	DB *sql.DB
}

func (w *WalletModel) CreditWallet(userID, amount int64) (*Wallet, error) {
	if amount < 0 {
		return nil, fmt.Errorf("Invalid amount: %d", amount)
	}

	stmt := `UPDATE wallets 
		SET balance = balance + $1, updated_at = $2
		WHERE user_id = $3
		RETURNING user_id, balance, locked, updated_at`

	wallet := &Wallet{}
	err := w.DB.QueryRow(stmt, amount, time.Now(), userID).Scan(
		&wallet.UserID,
		&wallet.Balance,
		&wallet.Locked,
		&wallet.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return wallet, nil
}

func (w *WalletModel) DebitWallet(userID, amount int64) (*Wallet, error) {
	if amount < 0 {
		return nil, fmt.Errorf("Invalid amount: %d", amount)
	}

	// Add balance check
	stmt := `UPDATE wallets
		SET balance = balance - $1, updated_at = $2
		WHERE user_id = $3 AND balance >= $1
		RETURNING user_id, balance, locked, updated_at`

	wallet := &Wallet{}
	err := w.DB.QueryRow(stmt, amount, time.Now(), userID).Scan(
		&wallet.UserID,
		&wallet.Balance,
		&wallet.Locked,
		&wallet.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("insufficient balance")
	}
	if err != nil {
		return nil, err
	}

	return wallet, nil
}

func (w *WalletModel) GetBalance(userID int64) (int64, error) {
	stmt := `SELECT balance FROM wallets WHERE user_id = $1`
	var balance int64
	err := w.DB.QueryRow(stmt, userID).Scan(&balance)
	if err != nil {
		return 0, err
	}
	return balance, nil
}

func (w *WalletModel) GetTotalBalance(userID int64) (int64, error) {
	stmt := `SELECT balance + locked FROM wallets
		WHERE user_id = $1`

	var totalBalance int64
	err := w.DB.QueryRow(stmt, userID).Scan(&totalBalance)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("wallet not found for user %d", userID)
		}
		return 0, err
	}

	return totalBalance, nil
}

func (w *WalletModel) LockAmount(userID, amount int64) (*Wallet, error) {
	stmt := `UPDATE wallets
		SET locked = locked + $1, updated_at = $2
		WHERE user_id = $3 AND locked >= $1
		RETURNING user_id, balance, locked, updated_at`

	wallet := &Wallet{}

	err := w.DB.QueryRow(stmt, amount, time.Now(), userID).Scan(
		&wallet.UserID,
		&wallet.Balance,
		&wallet.Locked,
		&wallet.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return wallet, nil
}

func (w *WalletModel) UnlockAmount(userID, amount int64) (*Wallet, error) {
	stmt := `UPDATE wallets
		SET locked = locked - $1, updated_at = $2
		WHERE user_id = $3
		RETURNING user_id, balance, locked, updated_at`

	wallet := &Wallet{}

	err := w.DB.QueryRow(stmt, amount, time.Now(), userID).Scan(
		&wallet.UserID,
		&wallet.Balance,
		&wallet.Locked,
		&wallet.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return wallet, nil
}

func (w *Wallet) HasEnoughBalance(price, quantity int64) bool {
	// Calculate required USDT in micro-USDT
	requiredMicroUSDT := engine.CalculateNotionalInt(price, quantity)

	// Check if available balance >= required
	availableBalance := w.Balance - w.Locked
	return availableBalance >= requiredMicroUSDT
}

func (w *WalletModel) GetWallet(userID int64) (*Wallet, error) {
	stmt := `SELECT * FROM wallets WHERE user_id = $1`

	wallet := &Wallet{}
	err := w.DB.QueryRow(stmt, userID).Scan(
		&wallet.UserID,
		&wallet.Balance,
		&wallet.Locked,
		&wallet.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return wallet, nil
}
