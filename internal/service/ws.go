package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kayden-vs/zaraba/pb"
	"github.com/kayden-vs/zaraba/ui/html/pages"
)

type Hub struct {
	Clients map[*websocket.Conn]bool
	Mu      sync.Mutex
}

var CenterHub = Hub{
	Clients: make(map[*websocket.Conn]bool),
}

type OrderBookHub struct {
	Clients map[*websocket.Conn]bool
	Mu      sync.Mutex
}

var OBhub = OrderBookHub{
	Clients: make(map[*websocket.Conn]bool),
}

var Upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func StartPriceFetcher() {
	for {
		// Only fetch if there are active clients
		CenterHub.Mu.Lock()
		clientCount := len(CenterHub.Clients)
		CenterHub.Mu.Unlock()

		if clientCount == 0 {
			log.Println("No active WebSocket clients, skipping fetch")
			time.Sleep(5 * time.Second)
			continue
		}

		url := "https://api.coingecko.com/api/v3/coins/markets" +
			"?vs_currency=usd" +
			"&order=market_cap_desc" +
			"&per_page=10" +
			"&page=1" +
			"&price_change_percentage=24h" +
			fmt.Sprintf("&x_cg_demo_api_key=%s", os.Getenv("API_KEY"))

		resp, err := http.Get(url)
		if err != nil {
			log.Println("Error fetching price:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var symbolData []pages.CoinMarketProps
		json.NewDecoder(resp.Body).Decode(&symbolData)
		resp.Body.Close()

		broadcast(symbolData)

		time.Sleep(5 * time.Second)
	}
}

func broadcast(message []pages.CoinMarketProps) {
	CenterHub.Mu.Lock()
	defer CenterHub.Mu.Unlock()

	data, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshaling message:", err)
		return
	}

	for client := range CenterHub.Clients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			client.Close()
			delete(CenterHub.Clients, client)
		}
	}
}

func StartOrderBookFetcher(symbolID string) {
	for {
		CenterHub.Mu.Lock()
		clientCount := len(OBhub.Clients)
		CenterHub.Mu.Unlock()

		if clientCount == 0 {
			log.Println("No active Orderbook clients, skipping fetch")
			time.Sleep(5 * time.Second)
			continue
		}

		data, err := NewExchangeServer().StreamOrderBook(context.Background(), &pb.OrderBookRequest{Market: symbolID})
		if err != nil {
			log.Println(err)
			continue
		}

		broadcastOrderBook(data)
		time.Sleep(2 * time.Second)
	}
}

func broadcastOrderBook(snapshot *pb.OrderbookSnapshot) {
	OBhub.Mu.Lock()
	defer OBhub.Mu.Unlock()

	data, err := json.Marshal(snapshot)
	if err != nil {
		log.Println("Error marshaling orderbook:", err)
		return
	}

	for client := range OBhub.Clients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			client.Close()
			delete(OBhub.Clients, client)
		}
	}
}
