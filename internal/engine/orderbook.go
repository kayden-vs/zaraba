package engine

import (
	"fmt"
	"time"

	"github.com/kayden-vs/zaraba/pb"
)

type Limit struct {
	*pb.Limit
}

type Order struct {
	*pb.Order
}

func NewLimit(price int64) *Limit {
	return &Limit{
		Limit: &pb.Limit{
			Price:  price,
			Orders: []*pb.Order{},
		},
	}
}

func NewOrder(bid bool, size float64) *Order {
	return &Order{
		Order: &pb.Order{
			Bid:       bid,
			Size:      float64(size),
			Timestamp: time.Now().UnixNano(),
		},
	}
}

func (l *Limit) AddOrder(o *Order) {
	o.Limit = l.Limit
	l.Orders = append(l.Orders, o.Order)
	l.TotalVolume += o.Size
}

func (o *Order) String() string {
	return fmt.Sprintf("[size: %.2f]", o.Size)
}

func (l *Limit) DeleteOrder(o *Order) {
	for i := 0; i < len(l.Orders); i++ {
		if l.Orders[i] == o.Order {
			l.Orders[i] = l.Orders[len(l.Orders)-1]
			l.Orders = l.Orders[:len(l.Orders)-1]
		}
	}

	o.Limit = nil
	l.TotalVolume -= o.Size

	// TODO: resort the whole resting orders
}

type Orderbook struct {
	Asks []*Limit
	Bids []*Limit
}
