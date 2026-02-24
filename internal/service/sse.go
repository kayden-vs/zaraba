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

	"github.com/kayden-vs/zaraba/pb"
	"github.com/kayden-vs/zaraba/ui/html/pages"
)

// For Server Side Events
// To update markets page and Orderbook

type Broker struct {
	clients map[chan []byte]bool
	mu      sync.Mutex
}

var PriceBroker = &Broker{
	clients: make(map[chan []byte]bool),
}

var OrderbookBroker = &Broker{
	clients: make(map[chan []byte]bool),
}

func (b *Broker) AddClient() chan []byte {
	ch := make(chan []byte, 5) // buffered so slow clients dont block
	b.mu.Lock()
	b.clients[ch] = true
	b.mu.Unlock()
	return ch
}

func (b *Broker) RemoveClient(ch chan []byte) {
	b.mu.Lock()
	delete(b.clients, ch)
	close(ch)
	b.mu.Unlock()
}

func (b *Broker) Broadcast(data []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for ch := range b.clients {
		select {
		case ch <- data: // send to client
		default: // client is slow/blocked, skip them
			log.Println("skipping slow client")
		}
	}
}

func (b *Broker) ClientCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.clients)
}

func StartMarketFetcher() {
	url := "https://api.coingecko.com/api/v3/coins/markets" +
		"?vs_currency=usd" +
		"&order=market_cap_desc" +
		"&per_page=10" +
		"&page=1" +
		"&price_change_percentage=24h" +
		fmt.Sprintf("&x_cg_demo_api_key=%s", os.Getenv("API_KEY"))

	for {
		if PriceBroker.ClientCount() == 0 {
			time.Sleep(5 * time.Second)
			// log.Println("No client, skipping fetch")
			continue
		}

		resp, err := http.Get(url)
		if err != nil {
			log.Println("Error fetching price:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var symbolData []pages.CoinMarketProps
		if err := json.NewDecoder(resp.Body).Decode(&symbolData); err != nil {
			log.Println("Error decoding response:", err)
			resp.Body.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		resp.Body.Close()

		jsonData, err := json.Marshal(symbolData)
		if err != nil {
			log.Println("Error marshaling data:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		PriceBroker.Broadcast(jsonData) // sends to all clients

		time.Sleep(5 * time.Second)
	}
}

func StartOrderBookFetcher(server *ExchangeServer) {
	for {
		if OrderbookBroker.ClientCount() == 0 {
			time.Sleep(5 * time.Second)
			continue
		}

		snapshot, err := server.StreamOrderBook(context.Background(), &pb.
			OrderBookRequest{Market: "btc"})
		if err != nil {
			log.Println("Error fetching orderbook:", err)
			time.Sleep(2 * time.Second)
			continue
		}

		jsonData, err := json.Marshal(snapshot)
		if err != nil {
			log.Println("Error marshaling snapshot data:", err)
			time.Sleep(2 * time.Second)
			continue
		}

		OrderbookBroker.Broadcast(jsonData)
		time.Sleep(2 * time.Second)
	}
}
