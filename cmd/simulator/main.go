package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type config struct {
	BaseURL        string
	Name           string
	Email          string
	Password       string
	Symbol         string
	CoinGeckoID    string
	APIKey         string
	Interval       time.Duration
	PriceRefresh   time.Duration
	Layers         int
	SpreadBps      float64
	BaseSize       float64
	MarketEvery    int
	MarketSize     float64
	AutoSignup     bool
	DepositUSDT    int
	RequestTimeout time.Duration
}

type simulator struct {
	cfg          config
	client       *http.Client
	rng          *rand.Rand
	cycle        int
	lastCGPrice  float64
	lastCGFetch  time.Time
	cgFetchCount int
}

type coinGeckoMarket struct {
	ID           string  `json:"id"`
	Symbol       string  `json:"symbol"`
	CurrentPrice float64 `json:"current_price"`
}

var csrfTokenRegex = regexp.MustCompile(`name=["']csrf_token["']\s+value=["']([^"']+)["']`)

func main() {
	loadDotEnvFile(".env")

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	sim := &simulator{
		cfg: cfg,
		client: &http.Client{
			Jar:     jar,
			Timeout: cfg.RequestTimeout,
		},
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := sim.bootstrapAuth(ctx); err != nil {
		log.Fatalf("login failed: %v", err)
	}
	log.Printf("simulator authenticated as %s", cfg.Email)

	if cfg.DepositUSDT > 0 {
		if err := sim.fundWallet(ctx, cfg.DepositUSDT); err != nil {
			log.Fatalf("wallet funding failed: %v", err)
		}
		log.Printf("wallet funded with %d USDT", cfg.DepositUSDT)
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		if err := sim.runCycle(ctx); err != nil {
			log.Printf("cycle failed: %v", err)
		}

		select {
		case <-ctx.Done():
			log.Println("simulator stopped")
			return
		case <-ticker.C:
		}
	}
}

func loadConfig() (config, error) {
	cfg := config{
		BaseURL:        getenvDefault("SIM_BASE_URL", "http://localhost:8080"),
		Name:           getenvDefault("SIM_NAME", "Market Maker Bot"),
		Email:          strings.TrimSpace(os.Getenv("SIM_EMAIL")),
		Password:       strings.TrimSpace(os.Getenv("SIM_PASSWORD")),
		Symbol:         getenvDefault("SIM_SYMBOL", "bitcoin"),
		CoinGeckoID:    getenvDefault("SIM_COINGECKO_ID", "bitcoin"),
		APIKey:         strings.TrimSpace(os.Getenv("API_KEY")),
		Interval:       getenvDurationMs("SIM_INTERVAL_MS", 1500),
		PriceRefresh:   getenvDurationMs("SIM_PRICE_REFRESH_MS", 15000),
		Layers:         getenvInt("SIM_LAYERS", 3),
		SpreadBps:      getenvFloat("SIM_SPREAD_BPS", 15),
		BaseSize:       getenvFloat("SIM_BASE_SIZE", 0.0008),
		MarketEvery:    getenvInt("SIM_MARKET_EVERY", 15),
		MarketSize:     getenvFloat("SIM_MARKET_SIZE", 0.0003),
		AutoSignup:     getenvBool("SIM_AUTO_SIGNUP", true),
		DepositUSDT:    getenvInt("SIM_DEPOSIT_USDT", 100000),
		RequestTimeout: 10 * time.Second,
	}

	if cfg.Email == "" || cfg.Password == "" {
		return config{}, fmt.Errorf("SIM_EMAIL and SIM_PASSWORD are required")
	}
	if cfg.Layers <= 0 {
		return config{}, fmt.Errorf("SIM_LAYERS must be > 0")
	}
	if cfg.Interval <= 0 {
		return config{}, fmt.Errorf("SIM_INTERVAL_MS must be > 0")
	}
	if cfg.PriceRefresh <= 0 {
		return config{}, fmt.Errorf("SIM_PRICE_REFRESH_MS must be > 0")
	}
	if cfg.BaseSize <= 0 || cfg.MarketSize <= 0 {
		return config{}, fmt.Errorf("SIM_BASE_SIZE and SIM_MARKET_SIZE must be > 0")
	}
	if cfg.MarketEvery <= 0 {
		cfg.MarketEvery = 15
	}

	return cfg, nil
}

func (s *simulator) bootstrapAuth(ctx context.Context) error {
	err := s.login(ctx)
	if err == nil {
		if ok := s.isAuthenticated(ctx); ok {
			return nil
		}
		err = fmt.Errorf("login did not establish authenticated session")
	}

	if !s.cfg.AutoSignup {
		return err
	}

	if signupErr := s.signup(ctx); signupErr != nil {
		return fmt.Errorf("login failed (%v), signup failed (%v)", err, signupErr)
	}

	if loginErr := s.login(ctx); loginErr != nil {
		return fmt.Errorf("post-signup login failed: %w", loginErr)
	}

	if ok := s.isAuthenticated(ctx); !ok {
		return fmt.Errorf("post-signup login did not establish authenticated session")
	}

	return nil
}

func (s *simulator) login(ctx context.Context) error {
	csrf, err := s.fetchCSRFToken(ctx, "/user/login")
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("email", s.cfg.Email)
	form.Set("password", s.cfg.Password)
	if csrf != "" {
		form.Set("csrf_token", csrf)
	}

	resp, err := s.postForm(ctx, "/user/login", form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2000))
		return fmt.Errorf("login rejected status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func (s *simulator) signup(ctx context.Context) error {
	csrf, err := s.fetchCSRFToken(ctx, "/user/signup")
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("name", s.cfg.Name)
	form.Set("email", s.cfg.Email)
	form.Set("password", s.cfg.Password)
	if csrf != "" {
		form.Set("csrf_token", csrf)
	}

	resp, err := s.postForm(ctx, "/user/signup", form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2000))
		return fmt.Errorf("signup rejected status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func (s *simulator) isAuthenticated(ctx context.Context) bool {
	endpoint := strings.TrimRight(s.cfg.BaseURL, "/") + "/user/wallet"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.Request != nil && strings.HasPrefix(resp.Request.URL.Path, "/user/login") {
		return false
	}

	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

func (s *simulator) fundWallet(ctx context.Context, amountUSDT int) error {
	csrf, err := s.fetchCSRFToken(ctx, "/user/wallet")
	if err != nil {
		return err
	}

	form := url.Values{}
	if csrf != "" {
		form.Set("csrf_token", csrf)
	}
	form.Set("amount", strconv.Itoa(amountUSDT))
	form.Set("type", "deposit")

	resp, err := s.postForm(ctx, "/api/wallet/transactions", form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2000))
		return fmt.Errorf("wallet deposit rejected status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func (s *simulator) runCycle(ctx context.Context) error {
	price, err := s.referencePrice(ctx)
	if err != nil {
		return err
	}

	s.cycle++

	csrf, err := s.fetchCSRFToken(ctx, "/trade/"+s.cfg.Symbol)
	if err != nil {
		return err
	}

	for layer := 1; layer <= s.cfg.Layers; layer++ {
		spreadMultiplier := 1 + (s.cfg.SpreadBps*float64(layer))/10_000
		askPrice := price * spreadMultiplier
		bidPrice := price / spreadMultiplier
		size := s.jitterSize(s.cfg.BaseSize)

		if err := s.placeLimitOrder(ctx, csrf, "buy", bidPrice, size); err != nil {
			return err
		}
		if err := s.placeLimitOrder(ctx, csrf, "sell", askPrice, size); err != nil {
			return err
		}
	}

	if s.cycle%s.cfg.MarketEvery == 0 {
		side := "buy"
		if s.cycle%(s.cfg.MarketEvery*2) == 0 {
			side = "sell"
		}
		if err := s.placeMarketOrder(ctx, csrf, side, s.jitterSize(s.cfg.MarketSize)); err != nil {
			log.Printf("market order skipped: %v", err)
		}
	}

	log.Printf("cycle=%d symbol=%s ref=%.2f layers=%d", s.cycle, s.cfg.Symbol, price, s.cfg.Layers)
	return nil
}

func (s *simulator) referencePrice(ctx context.Context) (float64, error) {
	now := time.Now()
	if s.lastCGPrice > 0 && now.Sub(s.lastCGFetch) < s.cfg.PriceRefresh {
		return s.lastCGPrice, nil
	}

	price, err := s.fetchCoinGeckoPrice(ctx)
	if err != nil {
		if s.lastCGPrice > 0 {
			log.Printf("coingecko refresh failed, using cached price %.2f: %v", s.lastCGPrice, err)
			return s.lastCGPrice, nil
		}
		return 0, err
	}

	s.lastCGPrice = price
	s.lastCGFetch = now
	s.cgFetchCount++
	log.Printf("coingecko refresh #%d price=%.2f", s.cgFetchCount, price)

	return price, nil
}

func (s *simulator) fetchCoinGeckoPrice(ctx context.Context) (float64, error) {
	query := url.Values{}
	query.Set("vs_currency", "usd")
	query.Set("ids", s.cfg.CoinGeckoID)
	if s.cfg.APIKey != "" {
		query.Set("x_cg_demo_api_key", s.cfg.APIKey)
	}

	endpoint := "https://api.coingecko.com/api/v3/coins/markets?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return 0, fmt.Errorf("coingecko status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var data []coinGeckoMarket
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, fmt.Errorf("no coingecko market data for %s", s.cfg.CoinGeckoID)
	}
	if data[0].CurrentPrice <= 0 {
		return 0, fmt.Errorf("invalid coingecko price: %.8f", data[0].CurrentPrice)
	}

	return data[0].CurrentPrice, nil
}

func (s *simulator) placeLimitOrder(ctx context.Context, csrf, side string, price, size float64) error {
	form := url.Values{}
	if csrf != "" {
		form.Set("csrf_token", csrf)
	}
	form.Set("side", side)
	form.Set("price", fmt.Sprintf("%.6f", price))
	form.Set("size", fmt.Sprintf("%.8f", size))

	resp, err := s.postForm(ctx, "/trade/"+s.cfg.Symbol+"/placelimitorder", form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1200))
		return fmt.Errorf("limit order failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func (s *simulator) placeMarketOrder(ctx context.Context, csrf, side string, size float64) error {
	form := url.Values{}
	if csrf != "" {
		form.Set("csrf_token", csrf)
	}
	form.Set("side", side)
	form.Set("size", fmt.Sprintf("%.8f", size))

	resp, err := s.postForm(ctx, "/trade/"+s.cfg.Symbol+"/placemarketorder", form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1200))
		return fmt.Errorf("market order failed status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func (s *simulator) fetchCSRFToken(ctx context.Context, path string) (string, error) {
	endpoint := strings.TrimRight(s.cfg.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("failed to fetch csrf page %s status=%d", path, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", err
	}

	matches := csrfTokenRegex.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return "", nil
	}

	return matches[1], nil
}

func (s *simulator) postForm(ctx context.Context, path string, form url.Values) (*http.Response, error) {
	endpoint := strings.TrimRight(s.cfg.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json, text/html;q=0.9")

	return s.client.Do(req)
}

func (s *simulator) jitterSize(base float64) float64 {
	factor := 0.85 + s.rng.Float64()*0.35
	return base * factor
}

func getenvDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func getenvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getenvFloat(key string, fallback float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func getenvDurationMs(key string, fallback int) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return time.Duration(fallback) * time.Millisecond
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return time.Duration(fallback) * time.Millisecond
	}
	return time.Duration(n) * time.Millisecond
}

func getenvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	if v == "1" || v == "true" || v == "yes" || v == "on" {
		return true
	}
	if v == "0" || v == "false" || v == "no" || v == "off" {
		return false
	}
	return fallback
}

func loadDotEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, `"`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, val)
	}
}
