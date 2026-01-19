package engine

import (
	"fmt"
	"testing"
)

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

func TestOrderbook(t *testing.T) {
	ob := NewOrderbook()

	buyOrder := NewOrder(true, 20)
	buyOrder1 := NewOrder(true, 4)
	buyOrder2 := NewOrder(true, 13)
	buyOrder3 := NewOrder(true, 11)
	askOrder := NewOrder(false, 50)
	askOrder1 := NewOrder(false, 75)

	ob.PlaceOrder(2000, buyOrder)
	ob.PlaceOrder(1500, buyOrder1)
	ob.PlaceOrder(2000, buyOrder2)
	ob.PlaceOrder(200, buyOrder3)
	ob.PlaceOrder(150, askOrder)
	ob.PlaceOrder(570, askOrder1)

	for _, v := range ob.Bids {
		fmt.Printf("Volume: %.2f, Price: %d\n", v.TotalVolume, v.Price)
	}
	for _, v := range ob.Asks {
		fmt.Printf("Volume: %.2f, Price: %d\n", v.TotalVolume, v.Price)
	}
}
