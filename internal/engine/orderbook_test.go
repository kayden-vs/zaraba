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

	fmt.Printf("Limit price: %d, TotalVolume: %d, Orders: %d\n", l.Price, l.TotalVolume, len(l.Orders))
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
	assert(t, ob.AskTotalVolume(), int64(10))
	assert(t, matches[0].Ask.Order, sellOrder.Order)
	assert(t, matches[0].Bid.Order, buyOrder.Order)
	assert(t, matches[0].SizeFilled, int64(20))
	assert(t, matches[0].Price, int64(100))
	assert(t, buyOrder.IsFilled(), true)

	fmt.Printf("%+v\n", matches)
}

func TestPlaceMarketOrderStopsWhenFilled(t *testing.T) {
	ob := NewOrderbook()

	// Two ask levels; buy order only needs part of first level.
	ob.PlaceLimitOrder(100, NewOrder(false, 30))
	ob.PlaceLimitOrder(101, NewOrder(false, 40))

	buyOrder := NewOrder(true, 20)
	matches := ob.PlaceMarketOrder(buyOrder)

	assert(t, len(matches), 1)
	assert(t, matches[0].Price, int64(100))
	assert(t, matches[0].SizeFilled, int64(20))
	assert(t, ob.Asks[0].TotalVolume, int64(10))
	assert(t, ob.AskTotalVolume(), int64(50))
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

	assert(t, ob.BidTotalVolume(), int64(31))

	sellOrder := NewOrder(false, 25)
	matches := ob.PlaceMarketOrder(sellOrder)

	assert(t, ob.BidTotalVolume(), int64(6))
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

	assert(t, ob.BidTotalVolume(), int64(15))

	ob.CancelOrder(buyOrderA)
	assert(t, ob.BidTotalVolume(), int64(5))
}

func TestPlaceLimitOrderCrossesAndRestsRemainder(t *testing.T) {
	ob := NewOrderbook()

	// Existing ask liquidity: 10 @ 100, 15 @ 101
	ob.PlaceLimitOrder(100, NewOrder(false, 10))
	ob.PlaceLimitOrder(101, NewOrder(false, 15))

	// Buy limit crosses both levels and should rest remaining 5 at 101.
	buy := NewOrder(true, 30)
	matches := ob.PlaceLimitOrder(101, buy)

	assert(t, len(matches), 2)
	assert(t, matches[0].Price, int64(100))
	assert(t, matches[0].SizeFilled, int64(10))
	assert(t, matches[1].Price, int64(101))
	assert(t, matches[1].SizeFilled, int64(15))
	assert(t, buy.Size, int64(5))

	assert(t, len(ob.Asks), 0)
	assert(t, len(ob.Bids), 1)
	assert(t, ob.Bids[0].Price, int64(101))
	assert(t, ob.Bids[0].TotalVolume, int64(5))
}
