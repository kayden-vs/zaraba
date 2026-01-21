package engine

import (
	"fmt"
	"reflect"
	"testing"
)

func assert(t *testing.T, a, b any) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("%+v != %+v", a, b)
	}
}

func TestLimt(t *testing.T) {
	l := NewLimit(6000)
	buyOrder := NewOrder(true, 10)
	buyOrder1 := NewOrder(true, 30)
	buyOrder2 := NewOrder(true, 20)

	l.AddOrder(buyOrder)
	l.AddOrder(buyOrder1)
	l.AddOrder(buyOrder2)

	l.DeleteOrder(buyOrder2)

	fmt.Printf("Limit price: %d, TotalVolume: %f, Orders: %d\n", l.Price, l.TotalVolume, len(l.Orders))
}

func TestLimitOrder(t *testing.T) {
	ob := NewOrderbook()

	sellOrderA := NewOrder(false, 10)
	sellOrderB := NewOrder(false, 20)

	ob.PlaceLimitOrder(10000, sellOrderA)
	ob.PlaceLimitOrder(1002, sellOrderB)

	assert(t, len(ob.Asks), 2)
}

func TestPlaceMarketOrder(t *testing.T) {
	ob := NewOrderbook()

	sellOrder := NewOrder(false, 30)
	ob.PlaceLimitOrder(100, sellOrder)

	buyOrder := NewOrder(true, 20)
	matches := ob.PlaceMarketOrder(buyOrder)

	assert(t, len(matches), 1)
	assert(t, len(ob.Asks), 1)
	assert(t, ob.AskTotalVolume(), 10.0)
	assert(t, matches[0].Ask.Order, sellOrder.Order)
	assert(t, matches[0].Bid.Order, buyOrder.Order)
	assert(t, matches[0].SizeFilled, 20.0)
	assert(t, matches[0].Price, int64(100))
	assert(t, buyOrder.IsFilled(), true)

	fmt.Printf("%+v", matches)
}
