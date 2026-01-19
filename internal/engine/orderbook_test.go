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

}
