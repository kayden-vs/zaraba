package main

import (
	"fmt"
	"sync"
	"time"
)

type Wallet struct {
	UserID    int64
	Balance   int64
	Locked    int64 // for open orders
	UpdatedAt time.Time
}

func (w *Wallet) CreditWallet(amount int64, mutex *sync.Mutex) error {
	if amount < 0 {
		return fmt.Errorf("Invalid amount: %d", amount)
	}

	mutex.Lock()
	w.Balance += amount
	mutex.Unlock()
	return nil
}

func (w *Wallet) DebitWallet(amount int64, mutex *sync.Mutex) {
	mutex.Lock()
	w.Balance -= amount
	mutex.Unlock()
}

func (w *Wallet) GetTotalBalance() int64 {
	return w.Balance + w.Locked
}

func (w *Wallet) LockAmount(amount int64, mutex *sync.Mutex) {
	mutex.Lock()
	w.Locked += amount
	mutex.Unlock()
}

func (w *Wallet) UnlockAmount(amount int64, mutex *sync.Mutex) {
	mutex.Lock()
	w.Locked -= amount
	mutex.Unlock()
}
