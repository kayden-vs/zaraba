package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/kayden-vs/zaraba/internal/engine"
	"github.com/kayden-vs/zaraba/internal/models"
	"github.com/kayden-vs/zaraba/internal/service"
	"github.com/kayden-vs/zaraba/internal/validator"
	"github.com/kayden-vs/zaraba/pb"
	"github.com/kayden-vs/zaraba/ui/html/pages"
)

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

type userSignupForm struct {
	Name                string `form:"name"`
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {

	props := pages.SignupFormParams{}
	app.RenderPage(w, r, func(flash string, isAuthenticated bool, csrfToken string) templ.Component {
		props.CSRFToken = csrfToken
		return pages.SignupPage(props, isAuthenticated)
	})
}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	var form userSignupForm
	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Validate the form contents using our helper functions.
	form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")
	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")
	form.CheckField(validator.MinChars(form.Password, 8), "password", "This field must be at least 8 characters long")

	props := pages.SignupFormParams{
		Name:        form.Name,
		Email:       form.Email,
		FieldErrors: form.FieldErrors,
	}

	// If there are any errors, redisplay the signup form along with a 422 status code.
	if !form.Valid() {
		app.RenderPage(w, r, func(flash string, isAuthenticated bool, csrfToken string) templ.Component {
			props.CSRFToken = csrfToken
			return pages.SignupPage(props, isAuthenticated)
		})
		return
	}

	var id int

	id, err = app.users.Insert(form.Name, form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateEmail) {
			form.AddFieldError("email", "Email address is already in use")
			props.FieldErrors = form.FieldErrors
			app.RenderPage(w, r, func(flash string, isAuthenticated bool, csrfToken string) templ.Component {
				props.CSRFToken = csrfToken
				return pages.SignupPage(props, isAuthenticated)
			})
		} else {
			app.serverError(w, err)
		}
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Account created Succesfully.")

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)

	// Create wallet for the new user
	err = app.wallet.CreateWallet(int64(id))
	if err != nil {
		app.serverError(w, err)
		return
	}

	http.Redirect(w, r, "/markets", http.StatusSeeOther)
}

type userLoginForm struct {
	Email               string `form:"email"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	props := pages.LoginFormParams{}
	app.RenderPage(w, r, func(flash string, isAuthenticated bool, csrfToken string) templ.Component {
		props.CSRFToken = csrfToken
		return pages.LoginPage(props, flash, isAuthenticated)
	})
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	var form userLoginForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Email), "email", "This field cannot be blank")
	form.CheckField(validator.Matches(form.Email, validator.EmailRX), "email", "This field must be a valid email address")
	form.CheckField(validator.NotBlank(form.Password), "password", "This field cannot be blank")

	props := pages.LoginFormParams{
		Email:          form.Email,
		FieldErrors:    form.FieldErrors,
		NonFieldErrors: form.NonFieldErrors,
	}
	if !form.Valid() {
		app.RenderPage(w, r, func(flash string, isAuthenticated bool, csrfToken string) templ.Component {
			props.CSRFToken = csrfToken
			return pages.LoginPage(props, flash, isAuthenticated)
		})
		return
	}

	id, err := app.users.Authenticate(form.Email, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Email or password is incorrect")
			props.NonFieldErrors = form.NonFieldErrors
			app.RenderPage(w, r, func(flash string, isAuthenticated bool, csrfToken string) templ.Component {
				props.CSRFToken = csrfToken
				return pages.LoginPage(props, flash, isAuthenticated)
			})
		} else {
			app.serverError(w, err)
		}
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}
	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)
	app.sessionManager.Put(r.Context(), "flash", "Logged in succesfully!")

	http.Redirect(w, r, "/markets", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Remove(r.Context(), "authenticatedUserID")
	app.sessionManager.Put(r.Context(), "flash", "You've been logged out Succesfully!")
	http.Redirect(w, r, "/markets", http.StatusSeeOther)
}

// ----- AUTH END -------

func (app *application) HomeHandler(w http.ResponseWriter, r *http.Request) {
	app.RenderPage(w, r, func(flash string, isAuthenticated bool, csrfToken string) templ.Component {
		return pages.LandingPage(flash, isAuthenticated, csrfToken)
	})
}

func (app *application) MarketsHandler(w http.ResponseWriter, r *http.Request) {
	symbolListProps, err := app.fetchCoinMarket()
	if err != nil {
		fmt.Println(err)
		// Still render page with empty data, WebSocket will populate
		symbolListProps = nil
	}

	app.RenderPage(w, r, func(flash string, isAuthenticated bool, csrfToken string) templ.Component {
		return pages.MarketsPage(symbolListProps, flash, isAuthenticated, csrfToken)
	})
}

func (app *application) TradeHandler(w http.ResponseWriter, r *http.Request) {
	userID := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")

	symbolID := chi.URLParam(r, "symbol")

	// chart
	var symbolData pages.CoinMarketProps
	symbolData, err := app.FetchSymbolData(symbolID)
	if err != nil {
		fmt.Println(err)
		return
	}

	tvSymbol := GetTvSymbol(symbolData.Symbol)

	balance, err := app.wallet.GetBalance(int64(userID))

	// trading panel

	app.RenderPage(w, r, func(flash string, isAuthenticated bool, csrfToken string) templ.Component {
		return pages.TradePage(symbolData, tvSymbol, balance, flash, isAuthenticated, csrfToken)
	})
}

func (app *application) WalletHandler(w http.ResponseWriter, r *http.Request) {
	userID := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")

	wallet, err := app.wallet.GetWallet(int64(userID))
	if err != nil {
		app.serverError(w, err)
		return
	}
	app.RenderPage(w, r, func(flash string, isAuthenticated bool, csrfToken string) templ.Component {
		return pages.WalletPage(wallet.Balance, wallet.Locked, flash, isAuthenticated, csrfToken)
	})
}

func (app *application) WalletHandlerPost(w http.ResponseWriter, r *http.Request) {
	// extract data
	value := r.FormValue("amount")
	transctionType := r.FormValue("type")
	amountUser, err := strconv.Atoi(value)

	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	amount := engine.PriceToInt(float64(amountUser))

	userID := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")

	// validate
	if amount <= 0 {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	currBalance, err := app.wallet.GetTotalBalance(int64(userID))

	if transctionType == "withdraw" && currBalance < amount {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// add to wallet
	if transctionType == "deposit" {
		_, err = app.wallet.CreditWallet(int64(userID), int64(amount))
		app.sessionManager.Put(r.Context(), "flash", fmt.Sprintf("$%d added successfully!", amountUser))
	}

	// debit from wallet
	if transctionType == "withdraw" {
		_, err = app.wallet.DebitWallet(int64(userID), amount)
		app.sessionManager.Put(r.Context(), "flash", fmt.Sprintf("$%d withdrawn successfully!", amountUser))
	}

	// TODO: add to portfolio summary

	http.Redirect(w, r, "/user/wallet", http.StatusSeeOther)
}

func (app *application) SseMarketHandler(w http.ResponseWriter, r *http.Request) {
	// set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		app.serverError(w, fmt.Errorf("streaming not supported"))
		return
	}

	// Establish the connection immediately so the browser doesn't time out
	// waiting for the first event
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// register this client with the broker
	clientChan := service.PriceBroker.AddClient()
	defer service.PriceBroker.RemoveClient(clientChan)

	for {
		select {
		case data := <-clientChan:
			// SSE format requires "data: ...\n\n"
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush() // push to browser immediately
		case <-r.Context().Done():
			// client disconnected
			// defer above handles cleanup automatically
			return
		}
	}
}

func (app *application) SseOrderBookHandler(w http.ResponseWriter, r *http.Request) {
	// set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		app.serverError(w, fmt.Errorf("streaming not supported"))
		return
	}

	// Establish the connection immediately
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Register client first, then broadcast so this client receives it
	clientChan := service.OrderbookBroker.AddClient()
	defer service.OrderbookBroker.RemoveClient(clientChan)

	// Send current orderbook state immediately on connect
	service.BroadcastOrderBook(app.exchangeServer)

	for {
		select {
		case data := <-clientChan:
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (app *application) SseTradesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		app.serverError(w, fmt.Errorf("streaming not supported"))
		return
	}

	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	clientChan := service.TradeBroker.AddClient()
	defer service.TradeBroker.RemoveClient(clientChan)

	initial := service.RecentTradesPayload()
	fmt.Fprintf(w, "data: %s\n\n", initial)
	flusher.Flush()

	for {
		select {
		case data := <-clientChan:
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (app *application) PlaceMarketOrderPost(w http.ResponseWriter, r *http.Request) {
	userID := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	symbol := chi.URLParam(r, "symbol")
	bid, size, _, err := parseOrderInput(r, false)
	if err != nil {
		app.respondOrderError(w, r, symbol, http.StatusBadRequest, err.Error())
		return
	}

	if err := app.ensureMarketDepth(bid, size); err != nil {
		app.respondOrderError(w, r, symbol, http.StatusBadRequest, err.Error())
		return
	}

	if bid {
		wallet, err := app.wallet.GetWallet(int64(userID))
		if err != nil {
			app.serverError(w, err)
			return
		}

		bestAsk, err := app.bestPrice(false)
		if err != nil {
			app.respondOrderError(w, r, symbol, http.StatusBadRequest, err.Error())
			return
		}

		if !wallet.HasEnoughBalance(bestAsk, size) {
			app.respondOrderError(w, r, symbol, http.StatusBadRequest, "insufficient wallet balance for this market buy")
			return
		}
	}

	order := &engine.Order{Order: &pb.Order{
		Id:        time.Now().UnixNano(),
		Bid:       bid,
		Size:      size,
		Timestamp: time.Now().UnixNano(),
	}}

	match, err := app.safePlaceMarketOrder(r.Context(), order.Order)
	if err != nil {
		app.respondOrderError(w, r, symbol, http.StatusBadRequest, err.Error())
		return
	}

	message := "market order accepted"
	if match != nil && match.SizeFilled > 0 {
		message = fmt.Sprintf("market order filled %s at %s", engine.FormatQuantity(match.SizeFilled), engine.FormatPrice(match.Price))
	}

	app.respondOrderSuccess(w, r, symbol, message, match)
}

func (app *application) PlaceLimitOrderPost(w http.ResponseWriter, r *http.Request) {
	userID := app.sessionManager.GetInt(r.Context(), "authenticatedUserID")
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	symbol := chi.URLParam(r, "symbol")
	bid, size, price, err := parseOrderInput(r, true)
	if err != nil {
		app.respondOrderError(w, r, symbol, http.StatusBadRequest, err.Error())
		return
	}

	if bid {
		wallet, err := app.wallet.GetWallet(int64(userID))
		if err != nil {
			app.serverError(w, err)
			return
		}

		if !wallet.HasEnoughBalance(price, size) {
			app.respondOrderError(w, r, symbol, http.StatusBadRequest, "insufficient wallet balance for this limit buy")
			return
		}
	}

	order := &pb.Order{
		Id:        time.Now().UnixNano(),
		Price:     price,
		Bid:       bid,
		Size:      size,
		Timestamp: time.Now().UnixNano(),
	}

	match, err := app.exchangeServer.PlaceLimitOrder(r.Context(), &pb.PlaceLimitOrderRequest{Price: price, Order: order})
	if err != nil {
		app.respondOrderError(w, r, symbol, http.StatusInternalServerError, err.Error())
		return
	}

	message := fmt.Sprintf("limit order placed at %s for %s", engine.FormatPrice(price), engine.FormatQuantity(size))
	app.respondOrderSuccess(w, r, symbol, message, match)
}

func (app *application) ensureMarketDepth(bid bool, size int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	snapshot, err := app.exchangeServer.StreamOrderBook(ctx, &pb.OrderBookRequest{Market: "default"})
	if err != nil {
		return fmt.Errorf("failed to read orderbook depth")
	}

	var total int64
	if bid {
		for _, limit := range snapshot.Asks {
			total += limit.TotalVolume
		}
	} else {
		for _, limit := range snapshot.Bids {
			total += limit.TotalVolume
		}
	}

	if total < size {
		return fmt.Errorf("not enough depth for market order")
	}

	return nil
}

func (app *application) bestPrice(bid bool) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	snapshot, err := app.exchangeServer.StreamOrderBook(ctx, &pb.OrderBookRequest{Market: "default"})
	if err != nil {
		return 0, fmt.Errorf("failed to read orderbook snapshot")
	}

	if bid {
		if len(snapshot.Bids) == 0 {
			return 0, fmt.Errorf("no bid liquidity")
		}
		return snapshot.Bids[0].Price, nil
	}

	if len(snapshot.Asks) == 0 {
		return 0, fmt.Errorf("no ask liquidity")
	}

	return snapshot.Asks[0].Price, nil
}

func (app *application) safePlaceMarketOrder(ctx context.Context, order *pb.Order) (match *pb.Match, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("market order rejected: %v", rec)
		}
	}()

	return app.exchangeServer.PlaceMarketOrder(ctx, order)
}

func parseOrderInput(r *http.Request, requirePrice bool) (bool, int64, int64, error) {
	err := r.ParseForm()
	if err != nil {
		return false, 0, 0, fmt.Errorf("invalid form payload")
	}

	sideRaw := strings.TrimSpace(r.FormValue("side"))
	bidRaw := strings.TrimSpace(r.FormValue("bid"))
	if sideRaw == "" && bidRaw == "" {
		return false, 0, 0, fmt.Errorf("missing order side")
	}

	bid := false
	if bidRaw != "" {
		parsedBid, err := strconv.ParseBool(bidRaw)
		if err != nil {
			return false, 0, 0, fmt.Errorf("invalid bid flag")
		}
		bid = parsedBid
	} else {
		side := strings.ToLower(sideRaw)
		switch side {
		case "buy", "bid":
			bid = true
		case "sell", "ask":
			bid = false
		default:
			return false, 0, 0, fmt.Errorf("invalid side")
		}
	}

	size, err := parseScaledDecimal(r.FormValue("size"), engine.QuantityScale)
	if err != nil || size <= 0 {
		return false, 0, 0, fmt.Errorf("invalid size")
	}

	if !requirePrice {
		return bid, size, 0, nil
	}

	price, err := parseScaledDecimal(r.FormValue("price"), engine.PriceScale)
	if err != nil || price <= 0 {
		return false, 0, 0, fmt.Errorf("invalid price")
	}

	return bid, size, price, nil
}

func parseScaledDecimal(raw string, scale int64) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("empty decimal value")
	}

	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}
	if f <= 0 {
		return 0, fmt.Errorf("decimal value must be positive")
	}

	return int64(math.Round(f * float64(scale))), nil
}

func (app *application) respondOrderError(w http.ResponseWriter, r *http.Request, symbol string, code int, message string) {
	if wantsJSON(r) {
		writeJSON(w, code, map[string]any{
			"ok":      false,
			"message": message,
		})
		return
	}

	app.sessionManager.Put(r.Context(), "flash", message)
	http.Redirect(w, r, "/trade/"+symbol, http.StatusSeeOther)
}

func (app *application) respondOrderSuccess(w http.ResponseWriter, r *http.Request, symbol, message string, match *pb.Match) {
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"message": message,
			"match":   match,
		})
		return
	}

	app.sessionManager.Put(r.Context(), "flash", message)
	http.Redirect(w, r, "/trade/"+symbol, http.StatusSeeOther)
}

func wantsJSON(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	return strings.Contains(accept, "application/json") || strings.Contains(ct, "application/json")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
