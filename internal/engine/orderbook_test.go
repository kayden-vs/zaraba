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

	fmt.Printf("%+v\n", matches)
}

func TestPlaceMarketOrderMultiFill(t *testing.T) {
	ob := NewOrderbook()

	buyOrder1 := NewOrder(true, 15)
	buyOrder2 := NewOrder(true, 5)
	buyOrder3 := NewOrder(true, 10)
	buyOrder4 := NewOrder(true, 1)

	ob.PlaceLimitOrder(1000, buyOrder3)
	ob.PlaceLimitOrder(10000, buyOrder1)
	ob.PlaceLimitOrder(5000, buyOrder4)
	ob.PlaceLimitOrder(5000, buyOrder2)

	assert(t, ob.BidTotalVolume(), 31.0)

	sellOrder := NewOrder(false, 25)
	matches := ob.PlaceMarketOrder(sellOrder)

	assert(t, ob.BidTotalVolume(), 6.0)
	assert(t, len(ob.Bids), 1)
	assert(t, len(matches), 4)
	fmt.Printf("%+v", matches)
}

func TestCancelOrder(t *testing.T) {
	ob := NewOrderbook()
	buyOrderA := NewOrder(true, 10)
	buyOrderB := NewOrder(true, 5)

	ob.PlaceLimitOrder(1000, buyOrderA)
	ob.PlaceLimitOrder(5000, buyOrderB)

	assert(t, ob.BidTotalVolume(), 15.0)

	ob.CancelOrder(buyOrderA)
	assert(t, ob.BidTotalVolume(), 5.0)
}
