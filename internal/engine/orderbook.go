package engine

import (
	"fmt"
	"sort"
	"time"

	"github.com/kayden-vs/zaraba/pb"
)

type Limit struct {
	*pb.Limit
}

type Order struct {
	*pb.Order
}

type Match struct {
	Ask        *Order
	Bid        *Order
	SizeFilled float64
	Price      int64
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
			// Faster delete : O(1)
			// l.Orders[i] = l.Orders[len(l.Orders)-1]
			// l.Orders = l.Orders[:len(l.Orders)-1]

			l.Orders = append(l.Orders[:i], l.Orders[i+1:]...)
		}
	}

	o.Limit = nil
	l.TotalVolume -= o.Size

	// TODO: resort the whole resting orders (Do not need if im using slower deletion)
}

type Orderbook struct {
	Asks []*Limit
	Bids []*Limit

	AskLimits map[int64]*Limit
	BidLimits map[int64]*Limit
}

func NewOrderbook() *Orderbook {
	return &Orderbook{
		Asks: []*Limit{},
		Bids: []*Limit{},

		AskLimits: make(map[int64]*Limit),
		BidLimits: make(map[int64]*Limit),
	}
}

func (ob *Orderbook) PlaceOrder(price int64, o *Order) []Match {
	// 1. Try to match the orders
	// matching logic

	// 2. add the rest of the orders to the books
	if o.Size > 0.0 {
		ob.add(price, o)
	}

	return []Match{}
}

func (ob *Orderbook) add(price int64, o *Order) {
	var limit *Limit

	if o.Bid {
		limit = ob.BidLimits[price]
	} else {
		limit = ob.AskLimits[price]
	}

	if limit == nil {
		limit = NewLimit(price)
		limit.AddOrder(o)

		if o.Bid {
			ob.Bids = append(ob.Bids, limit)

			// sort bids here
			sort.Slice(ob.Bids, func(i, j int) bool {
				return ob.Bids[i].Price < ob.Bids[j].Price
			})

			ob.BidLimits[price] = limit
		} else {
			ob.Asks = append(ob.Asks, limit)

			// sort asks here
			sort.Slice(ob.Asks, func(i, j int) bool {
				return ob.Asks[i].Price > ob.Asks[j].Price
			})

			ob.AskLimits[price] = limit
		}
	} else {
		limit.AddOrder(o)
	}
}
