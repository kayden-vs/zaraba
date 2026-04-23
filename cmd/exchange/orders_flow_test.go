package main

import (
	"testing"
	"time"

	"github.com/kayden-vs/zaraba/internal/models"
)

func TestCanCancelOrderAllowsLegacyLimitOrder(t *testing.T) {
	order := &models.Order{
		EngineOrderID: 0,
		OrderType:     models.OrderTypeLimit,
		Status:        models.OrderStatusOpen,
		Quantity:      10,
		Filled:        3,
	}

	if !canCancelOrder(order) {
		t.Fatal("expected open legacy limit order to be cancellable")
	}
}

func TestCanCancelOrderRejectsClosedOrNonLimit(t *testing.T) {
	closed := &models.Order{
		OrderType: models.OrderTypeLimit,
		Status:    models.OrderStatusFilled,
		Quantity:  5,
		Filled:    5,
	}
	if canCancelOrder(closed) {
		t.Fatal("expected filled order to be non-cancellable")
	}

	market := &models.Order{
		OrderType: models.OrderTypeMarket,
		Status:    models.OrderStatusOpen,
		Quantity:  5,
		Filled:    0,
	}
	if canCancelOrder(market) {
		t.Fatal("expected market order to be non-cancellable")
	}
}

func TestMatchesStatusFilterTreatsUnknownAsActive(t *testing.T) {
	_, activeFilter := normalizeOrdersTab("active")

	if !matchesStatusFilter("PENDING", activeFilter) {
		t.Fatal("expected unknown non-closed status to appear in active tab")
	}

	if matchesStatusFilter(models.OrderStatusFilled, activeFilter) {
		t.Fatal("expected filled orders to be excluded from active tab")
	}
}

func TestShouldInferMissingProgress(t *testing.T) {
	start := time.Now()
	app := &application{startedAt: start}

	oldOrder := &models.Order{CreatedAt: start.Add(-time.Minute)}
	if app.shouldInferMissingProgress(oldOrder) {
		t.Fatal("expected pre-boot order to skip missing-order inference")
	}

	newOrder := &models.Order{CreatedAt: start.Add(time.Second)}
	if !app.shouldInferMissingProgress(newOrder) {
		t.Fatal("expected post-boot order to infer missing-order progress")
	}
}
