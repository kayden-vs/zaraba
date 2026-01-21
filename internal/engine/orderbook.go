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

func (o *Order) String() string {
	return fmt.Sprintf("[size: %.2f]", o.Size)
}

func (o *Order) IsFilled() bool {
	return o.Size == 0.0
}

func (l *Limit) AddOrder(o *Order) {
	o.Limit = l.Limit
	l.Orders = append(l.Orders, o.Order)
	l.TotalVolume += o.Size
}

func (l *Limit) DeleteOrder(o *Order) {
	for i := 0; i < len(l.Orders); i++ {
		if l.Orders[i] == o.Order {
			// Faster delete : O(1) *but we have sort the orders after,so we not using this*
			// l.Orders[i] = l.Orders[len(l.Orders)-1]
			// l.Orders = l.Orders[:len(l.Orders)-1]

			l.Orders = append(l.Orders[:i], l.Orders[i+1:]...)
		}
	}

	o.Limit = nil
	l.TotalVolume -= o.Size
}

func (l *Limit) Fill(o *Order) []Match {
	matches := []Match{}

	for _, order := range l.Orders {
		localOrder := &Order{Order: order}
		match := l.fillOrder(localOrder, o)
		matches = append(matches, match)

		l.TotalVolume -= match.SizeFilled

		if o.IsFilled() {
			break
		}
	}

	return matches
}

func (l *Limit) fillOrder(a, b *Order) Match {
	var (
		bid        *Order
		ask        *Order
		sizeFilled float64
	)

	if a.Bid {
		bid = a
		ask = b
	} else {
		bid = b
		ask = a
	}

	if a.Size >= b.Size {
		a.Size -= b.Size
		sizeFilled = b.Size
		b.Size = 0.0
	} else {
		b.Size -= a.Size
		sizeFilled = a.Size
		a.Size = 0.0
	}

	return Match{
		Bid:        bid,
		Ask:        ask,
		SizeFilled: sizeFilled,
		Price:      l.Price,
	}
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

func (ob *Orderbook) PlaceMarketOrder(o *Order) []Match {
	matches := []Match{}

	if o.Bid {
		if o.Size > ob.AskTotalVolume() {
			panic(fmt.Errorf("Not enough volume [size: %.2f] for market order [size: %.2f]", ob.AskTotalVolume(), o.Size))
		}

		for _, limit := range ob.Asks {
			limitMatches := limit.Fill(o)
			matches = append(matches, limitMatches...)
		}
	} else {
		if o.Size > ob.BidTotalVolume() {
			panic(fmt.Errorf("Not enough volume [size: %.2f] for market order [size: %.2f]", ob.BidTotalVolume(), o.Size))
		}

		for _, limit := range ob.Bids {
			limitMatches := limit.Fill(o)
			matches = append(matches, limitMatches...)
		}
	}

	return matches
}

func (ob *Orderbook) PlaceLimitOrder(price int64, o *Order) {
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
				return ob.Bids[i].Price > ob.Bids[j].Price
			})

			ob.BidLimits[price] = limit
		} else {
			ob.Asks = append(ob.Asks, limit)

			// sort asks here
			sort.Slice(ob.Asks, func(i, j int) bool {
				return ob.Asks[i].Price < ob.Asks[j].Price
			})

			ob.AskLimits[price] = limit
		}
	} else {
		limit.AddOrder(o)
	}
}

func (ob *Orderbook) BidTotalVolume() float64 {
	var totalVolume float64

	for i := 0; i < len(ob.Bids); i++ {
		totalVolume += ob.Bids[i].TotalVolume
	}

	return totalVolume
}

func (ob *Orderbook) AskTotalVolume() float64 {
	var totalVolume float64

	for i := 0; i < len(ob.Asks); i++ {
		totalVolume += ob.Asks[i].TotalVolume
	}

	return totalVolume
}
