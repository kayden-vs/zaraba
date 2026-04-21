package pages

import "time"

type OrderSummaryProps struct {
	TotalOrders         int
	ActiveOrders        int
	FilledOrders        int
	CancelledOrders     int
	TotalFilledNotional int64
	TotalUnrealizedPnL  int64
}

type OrderRowProps struct {
	ID               int64
	EngineOrderID    int64
	Pair             string
	Symbol           string
	Side             string
	OrderType        string
	Status           string
	Price            int64
	Quantity         int64
	Filled           int64
	Remaining        int64
	FillPercent      float64
	FilledNotional   int64
	AvgFillPrice     int64
	LastPrice        int64
	UnrealizedPnL    int64
	UnrealizedPnLPct float64
	HasPnL           bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CanCancel        bool
}

type OrdersPageProps struct {
	ActiveTab string
	Orders    []OrderRowProps
	Summary   OrderSummaryProps
}
