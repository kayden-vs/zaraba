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

	app.sessionManager.Put(r.Context(), "flash", "success:Account created Succesfully.")

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
	app.sessionManager.Put(r.Context(), "flash", "success:Logged in succesfully!")

	http.Redirect(w, r, "/markets", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Remove(r.Context(), "authenticatedUserID")
	app.sessionManager.Put(r.Context(), "flash", "success:You've been logged out Succesfully!")
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
	app.safeEnsureDemoLiquidity()

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
		app.sessionManager.Put(r.Context(), "flash", fmt.Sprintf("success:$%d added successfully!", amountUser))
	}

	// debit from wallet
	if transctionType == "withdraw" {
		_, err = app.wallet.DebitWallet(int64(userID), amount)
		app.sessionManager.Put(r.Context(), "flash", fmt.Sprintf("success:$%d withdrawn successfully!", amountUser))
	}

	// TODO: add to portfolio summary

	http.Redirect(w, r, "/user/wallet", http.StatusSeeOther)
}

func (app *application) SseMarketHandler(w http.ResponseWriter, r *http.Request) {
	setSSEHeaders(w)

	flusher, ok := w.(http.Flusher)
	if !ok {
		app.serverError(w, fmt.Errorf("streaming not supported"))
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintf(w, ": connected\n\n"); err != nil {
		return
	}
	if _, err := fmt.Fprintf(w, "data: []\n\n"); err != nil {
		return
	}
	flusher.Flush()

	clientChan := service.PriceBroker.AddClient()
	defer service.PriceBroker.RemoveClient(clientChan)
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case data, open := <-clientChan:
			if !open {
				return
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (app *application) SseOrderBookHandler(w http.ResponseWriter, r *http.Request) {
	app.safeEnsureDemoLiquidity()
	setSSEHeaders(w)

	flusher, ok := w.(http.Flusher)
	if !ok {
		app.serverError(w, fmt.Errorf("streaming not supported"))
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintf(w, ": connected\n\n"); err != nil {
		return
	}
	flusher.Flush()

	clientChan := service.OrderbookBroker.AddClient()
	defer service.OrderbookBroker.RemoveClient(clientChan)

	service.BroadcastOrderBook(app.exchangeServer)
	ticker := time.NewTicker(3 * time.Second)
	heartbeat := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	defer heartbeat.Stop()

	for {
		select {
		case data, open := <-clientChan:
			if !open {
				return
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			app.safeEnsureDemoLiquidity()
			service.BroadcastOrderBook(app.exchangeServer)
		case <-heartbeat.C:
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (app *application) SseTradesHandler(w http.ResponseWriter, r *http.Request) {
	app.safeEnsureDemoLiquidity()
	setSSEHeaders(w)

	flusher, ok := w.(http.Flusher)
	if !ok {
		app.serverError(w, fmt.Errorf("streaming not supported"))
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := fmt.Fprintf(w, ": connected\n\n"); err != nil {
		return
	}
	flusher.Flush()

	clientChan := service.TradeBroker.AddClient()
	defer service.TradeBroker.RemoveClient(clientChan)

	initial := service.RecentTradesPayload()
	if _, err := fmt.Fprintf(w, "data: %s\n\n", initial); err != nil {
		return
	}
	flusher.Flush()
	ticker := time.NewTicker(3 * time.Second)
	heartbeat := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	defer heartbeat.Stop()

	for {
		select {
		case data, open := <-clientChan:
			if !open {
				return
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
		case <-ticker.C:
			app.safeEnsureDemoLiquidity()
		case <-heartbeat.C:
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (app *application) safeEnsureDemoLiquidity() {
	defer func() {
		if rec := recover(); rec != nil {
			app.errorLog.Printf("demo liquidity panic recovered: %v", rec)
		}
	}()

	service.EnsureDemoLiquidity(app.exchangeServer)
}

func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	// Disable proxy buffering for long-lived event streams.
	w.Header().Set("X-Accel-Buffering", "no")
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

	var bestAsk int64
	if bid {
		wallet, err := app.wallet.GetWallet(int64(userID))
		if err != nil {
			app.serverError(w, err)
			return
		}

		bestAsk, err = app.bestPrice(false)
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

	initialPrice := bestAsk
	if !bid {
		if sellBestBid, err := app.bestPrice(true); err == nil {
			initialPrice = sellBestBid
		}
	}

	orderID, err := app.orders.Create(&models.Order{
		UserID:         int64(userID),
		EngineOrderID:  order.Id,
		Symbol:         symbol,
		Side:           orderSide(bid),
		OrderType:      models.OrderTypeMarket,
		Price:          initialPrice,
		Quantity:       size,
		Filled:         0,
		FilledNotional: 0,
		Status:         models.OrderStatusOpen,
	})
	if err != nil {
		app.serverError(w, err)
		return
	}

	match, err := app.safePlaceMarketOrder(r.Context(), order.Order)
	if err != nil {
		_ = app.orders.DeleteByEngineOrderID(int64(userID), order.Id)
		app.respondOrderError(w, r, symbol, http.StatusBadRequest, err.Error())
		return
	}

	filled := size
	if match != nil && match.SizeFilled > 0 {
		filled = match.SizeFilled
	}
	if filled < 0 {
		filled = 0
	}
	if filled > size {
		filled = size
	}

	executionPrice := initialPrice
	if match != nil && match.Price > 0 {
		executionPrice = match.Price
	}

	executedNotional := int64(0)
	if executionPrice > 0 && filled > 0 {
		executedNotional = engine.CalculateNotionalInt(executionPrice, filled)
	}

	status := models.OrderStatusFilled
	if filled == 0 {
		status = models.OrderStatusCancelled
	} else if filled < size {
		status = models.OrderStatusPartiallyFilled
	}

	if bid {
		if executedNotional <= 0 {
			executedNotional = engine.CalculateNotionalInt(bestAsk, filled)
		}

		if _, err = app.wallet.DebitWallet(int64(userID), executedNotional); err != nil {
			_ = app.orders.UpdateExecution(orderID, filled, executedNotional, status)
			app.respondOrderError(w, r, symbol, http.StatusBadRequest, err.Error())
			return
		}
	}

	if err = app.orders.UpdateExecution(orderID, filled, executedNotional, status); err != nil {
		app.serverError(w, err)
		return
	}

	message := "market order accepted"
	if filled > 0 {
		message = fmt.Sprintf("market order filled %s at %s", engine.FormatQuantity(filled), engine.FormatPrice(executionPrice))
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
	originalSize := size
	lockedNotional := int64(0)
	if bid {
		lockedNotional = engine.CalculateNotionalInt(order.Price, order.Size)
		if _, err = app.wallet.LockAmount(int64(userID), lockedNotional); err != nil {
			app.respondOrderError(w, r, symbol, http.StatusBadRequest, err.Error())
			return
		}
	}

	orderID, err := app.orders.Create(&models.Order{
		UserID:         int64(userID),
		EngineOrderID:  order.Id,
		Symbol:         symbol,
		Side:           orderSide(bid),
		OrderType:      models.OrderTypeLimit,
		Price:          price,
		Quantity:       originalSize,
		Filled:         0,
		FilledNotional: 0,
		Status:         models.OrderStatusOpen,
	})
	if err != nil {
		if bid && lockedNotional > 0 {
			_, _ = app.wallet.UnlockAmount(int64(userID), lockedNotional)
		}
		app.serverError(w, err)
		return
	}

	match, err := app.exchangeServer.PlaceLimitOrder(r.Context(), &pb.PlaceLimitOrderRequest{Price: price, Order: order})
	if err != nil {
		_ = app.orders.DeleteByEngineOrderID(int64(userID), order.Id)
		if bid && lockedNotional > 0 {
			_, _ = app.wallet.UnlockAmount(int64(userID), lockedNotional)
		}
		app.respondOrderError(w, r, symbol, http.StatusInternalServerError, err.Error())
		return
	}

	filledSize := originalSize - order.Size
	if filledSize < 0 {
		filledSize = 0
	}
	if filledSize > originalSize {
		filledSize = originalSize
	}
	if match != nil && match.SizeFilled > filledSize {
		filledSize = match.SizeFilled
		if filledSize > originalSize {
			filledSize = originalSize
		}
	}

	executionPrice := price
	if match != nil && match.Price > 0 {
		executionPrice = match.Price
	}

	executedNotional := int64(0)
	if filledSize > 0 && executionPrice > 0 {
		executedNotional = engine.CalculateNotionalInt(executionPrice, filledSize)
	}

	status := models.OrderStatusOpen
	if order.Size == 0 {
		status = models.OrderStatusFilled
	} else if filledSize > 0 {
		status = models.OrderStatusPartiallyFilled
	}

	if bid {
		if filledSize > 0 {
			if _, err = app.wallet.DebitWallet(int64(userID), executedNotional); err != nil {
				_, _ = app.wallet.UnlockAmount(int64(userID), lockedNotional)
				_ = app.orders.UpdateExecution(orderID, filledSize, executedNotional, status)
				app.respondOrderError(w, r, symbol, http.StatusBadRequest, err.Error())
				return
			}

			if _, err = app.wallet.UnlockAmount(int64(userID), executedNotional); err != nil {
				_ = app.orders.UpdateExecution(orderID, filledSize, executedNotional, status)
				app.respondOrderError(w, r, symbol, http.StatusBadRequest, err.Error())
				return
			}
		}
	}

	if err = app.orders.UpdateExecution(orderID, filledSize, executedNotional, status); err != nil {
		app.serverError(w, err)
		return
	}

	message := fmt.Sprintf("limit order placed at %s for %s", engine.FormatPrice(price), engine.FormatQuantity(size))
	app.respondOrderSuccess(w, r, symbol, message, match)
}

func (app *application) ensureMarketDepth(bid bool, size int64) error {
	service.EnsureDemoLiquidityForMarketOrder(app.exchangeServer, bid, size)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	snapshot, err := app.exchangeServer.StreamOrderBook(ctx, &pb.OrderBookRequest{Market: "default"})
	if err != nil {
		return fmt.Errorf("failed to read orderbook depth")
	}

	total := depthForMarketSide(snapshot, bid)
	if total < size {
		service.EnsureDemoLiquidityForMarketOrder(app.exchangeServer, bid, size)
		snapshot, err = app.exchangeServer.StreamOrderBook(ctx, &pb.OrderBookRequest{Market: "default"})
		if err != nil {
			return fmt.Errorf("failed to read orderbook depth")
		}
		total = depthForMarketSide(snapshot, bid)
	}

	if total < size {
		return fmt.Errorf("not enough depth for market order")
	}

	return nil
}

func depthForMarketSide(snapshot *pb.OrderbookSnapshot, bid bool) int64 {
	if snapshot == nil {
		return 0
	}

	var total int64
	if bid {
		for _, limit := range snapshot.Asks {
			total += limit.TotalVolume
		}
		return total
	}

	for _, limit := range snapshot.Bids {
		total += limit.TotalVolume
	}

	return total
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

	app.sessionManager.Put(r.Context(), "flash", "error:"+message)
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

	app.sessionManager.Put(r.Context(), "flash", "success:"+message)
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

func (app *application) OrdersHandler(w http.ResponseWriter, r *http.Request) {
	userID := int64(app.sessionManager.GetInt(r.Context(), "authenticatedUserID"))
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	if err := app.syncUserOpenOrders(userID); err != nil {
		app.errorLog.Printf("orders sync failed for user %d: %v", userID, err)
	}

	allOrders, err := app.orders.ListByUser(userID, nil, 500)
	if err != nil {
		app.serverError(w, err)
		return
	}

	tab, statusFilter := normalizeOrdersTab(r.URL.Query().Get("tab"))
	priceBySymbol, pairBySymbol := app.marketMetadata()

	rows := make([]pages.OrderRowProps, 0, len(allOrders))
	summary := pages.OrderSummaryProps{TotalOrders: len(allOrders)}

	for _, order := range allOrders {
		status := strings.ToUpper(strings.TrimSpace(order.Status))
		switch status {
		case models.OrderStatusOpen, models.OrderStatusPartiallyFilled:
			summary.ActiveOrders++
		case models.OrderStatusFilled:
			summary.FilledOrders++
		case models.OrderStatusCancelled:
			summary.CancelledOrders++
		}

		summary.TotalFilledNotional += order.FilledNotional

		lastPrice := priceBySymbol[strings.ToLower(order.Symbol)]
		pnlValue, pnlPct, hasPnL := calculateUnrealizedPnL(order, lastPrice)
		if hasPnL {
			summary.TotalUnrealizedPnL += pnlValue
		}

		if !matchesStatusFilter(status, statusFilter) {
			continue
		}

		rows = append(rows, pages.OrderRowProps{
			ID:               order.ID,
			EngineOrderID:    order.EngineOrderID,
			Pair:             formatOrderPair(order.Symbol, pairBySymbol),
			Symbol:           strings.ToUpper(order.Symbol),
			Side:             strings.ToUpper(order.Side),
			OrderType:        strings.ToUpper(order.OrderType),
			Status:           status,
			Price:            order.Price,
			Quantity:         order.Quantity,
			Filled:           order.Filled,
			Remaining:        order.Remaining(),
			FillPercent:      filledPercent(order.Filled, order.Quantity),
			FilledNotional:   order.FilledNotional,
			AvgFillPrice:     order.AvgFillPrice,
			LastPrice:        lastPrice,
			UnrealizedPnL:    pnlValue,
			UnrealizedPnLPct: pnlPct,
			HasPnL:           hasPnL,
			CreatedAt:        order.CreatedAt,
			UpdatedAt:        order.UpdatedAt,
			CanCancel:        canCancelOrder(order),
		})
	}

	props := pages.OrdersPageProps{
		ActiveTab: tab,
		Orders:    rows,
		Summary:   summary,
	}

	app.RenderPage(w, r, func(flash string, isAuthenticated bool, csrfToken string) templ.Component {
		return pages.OrdersPage(props, flash, isAuthenticated, csrfToken)
	})
}

func (app *application) CancelOrderPost(w http.ResponseWriter, r *http.Request) {
	userID := int64(app.sessionManager.GetInt(r.Context(), "authenticatedUserID"))
	if userID == 0 {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	orderID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || orderID <= 0 {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	if err = app.syncUserOpenOrders(userID); err != nil {
		app.errorLog.Printf("pre-cancel sync failed for user %d order %d: %v", userID, orderID, err)
	}

	order, err := app.orders.GetByIDForUser(orderID, userID)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w, r)
			return
		}
		app.serverError(w, err)
		return
	}

	if !canCancelOrder(order) {
		app.sessionManager.Put(r.Context(), "flash", "error:order can no longer be cancelled")
		http.Redirect(w, r, "/orders?tab=active", http.StatusSeeOther)
		return
	}

	cancelled, ok := app.exchangeServer.CancelOrderByID(order.EngineOrderID)
	if !ok {
		_ = app.syncUserOpenOrders(userID)
		app.sessionManager.Put(r.Context(), "flash", "error:order is no longer active")
		http.Redirect(w, r, "/orders?tab=active", http.StatusSeeOther)
		return
	}

	if order.IsBuy() && order.Price > 0 {
		releaseQty := order.Remaining()
		if cancelled != nil && cancelled.Size > 0 && cancelled.Size < releaseQty {
			releaseQty = cancelled.Size
		}
		if releaseQty > 0 {
			releaseNotional := engine.CalculateNotionalInt(order.Price, releaseQty)
			if err = app.unlockWalletUpTo(userID, releaseNotional); err != nil {
				app.serverError(w, err)
				return
			}
		}
	}

	if err = app.orders.MarkCancelled(order.ID); err != nil {
		app.serverError(w, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", fmt.Sprintf("success:Order #%d cancelled", order.ID))
	http.Redirect(w, r, "/orders?tab=active", http.StatusSeeOther)
}

func (app *application) syncUserOpenOrders(userID int64) error {
	if userID == 0 {
		return nil
	}

	openOrders, err := app.orders.ListOpenByUser(userID)
	if err != nil {
		return err
	}
	if len(openOrders) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	snapshot, err := app.exchangeServer.StreamOrderBook(ctx, &pb.OrderBookRequest{Market: "default"})
	if err != nil {
		return err
	}

	remainingByID := orderbookRemainingByID(snapshot)

	for _, order := range openOrders {
		if order.EngineOrderID == 0 {
			continue
		}

		remaining := int64(0)
		if v, ok := remainingByID[order.EngineOrderID]; ok {
			remaining = v
		}
		if remaining < 0 {
			remaining = 0
		}
		if remaining > order.Quantity {
			remaining = order.Quantity
		}

		newFilled := order.Quantity - remaining
		if newFilled < 0 {
			newFilled = 0
		}
		if newFilled > order.Quantity {
			newFilled = order.Quantity
		}
		if newFilled < order.Filled {
			newFilled = order.Filled
		}

		fillDelta := newFilled - order.Filled
		newFilledNotional := order.FilledNotional

		if fillDelta > 0 && order.Price > 0 {
			deltaNotional := engine.CalculateNotionalInt(order.Price, fillDelta)
			if order.IsBuy() && deltaNotional > 0 {
				if _, err = app.wallet.DebitWallet(userID, deltaNotional); err != nil {
					app.errorLog.Printf("wallet debit sync failed (user=%d order=%d): %v", userID, order.ID, err)
					continue
				}
				if err = app.unlockWalletUpTo(userID, deltaNotional); err != nil {
					app.errorLog.Printf("wallet unlock sync failed (user=%d order=%d): %v", userID, order.ID, err)
					continue
				}
			}
			newFilledNotional += deltaNotional
		}

		status := statusFromProgress(order.Quantity, newFilled)
		if status == models.OrderStatusOpen && newFilled > 0 {
			status = models.OrderStatusPartiallyFilled
		}

		if newFilled == order.Filled && newFilledNotional == order.FilledNotional && status == order.Status {
			continue
		}

		if err = app.orders.UpdateExecution(order.ID, newFilled, newFilledNotional, status); err != nil {
			return err
		}
	}

	return nil
}

func (app *application) marketMetadata() (map[string]int64, map[string]string) {
	prices := make(map[string]int64)
	pairs := make(map[string]string)

	markets, err := app.fetchCoinMarket()
	if err != nil {
		return prices, pairs
	}

	for _, market := range markets {
		symbolID := strings.ToLower(strings.TrimSpace(market.ID))
		if symbolID == "" {
			continue
		}

		prices[symbolID] = engine.PriceToInt(market.Price)

		ticker := strings.ToUpper(strings.TrimSpace(market.Symbol))
		if ticker != "" {
			pairs[symbolID] = ticker + "/USDT"
		}
	}

	return prices, pairs
}

func orderbookRemainingByID(snapshot *pb.OrderbookSnapshot) map[int64]int64 {
	remainingByID := make(map[int64]int64)
	if snapshot == nil {
		return remainingByID
	}

	for _, limit := range snapshot.Asks {
		for _, order := range limit.Orders {
			remainingByID[order.Id] = order.Size
		}
	}

	for _, limit := range snapshot.Bids {
		for _, order := range limit.Orders {
			remainingByID[order.Id] = order.Size
		}
	}

	return remainingByID
}

func filledPercent(filled, quantity int64) float64 {
	if quantity <= 0 {
		return 0
	}
	pct := (float64(filled) / float64(quantity)) * 100
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

func calculateUnrealizedPnL(order *models.Order, lastPrice int64) (int64, float64, bool) {
	if order == nil || order.Filled <= 0 || lastPrice <= 0 {
		return 0, 0, false
	}

	entryPrice := order.AvgFillPrice
	if entryPrice <= 0 && order.FilledNotional > 0 {
		entryPrice = int64(math.Round((float64(order.FilledNotional) / float64(order.Filled)) * float64(engine.QuantityScale)))
	}
	if entryPrice <= 0 {
		entryPrice = order.Price
	}
	if entryPrice <= 0 {
		return 0, 0, false
	}

	entryNotional := engine.CalculateNotionalInt(entryPrice, order.Filled)
	markNotional := engine.CalculateNotionalInt(lastPrice, order.Filled)
	if entryNotional <= 0 {
		return 0, 0, false
	}

	pnl := markNotional - entryNotional
	if !order.IsBuy() {
		pnl = entryNotional - markNotional
	}

	pct := (float64(pnl) / float64(entryNotional)) * 100
	return pnl, pct, true
}

func normalizeOrdersTab(raw string) (string, map[string]bool) {
	tab := strings.ToLower(strings.TrimSpace(raw))
	switch tab {
	case "completed":
		return "completed", map[string]bool{
			models.OrderStatusFilled:    true,
			models.OrderStatusCancelled: true,
		}
	case "all":
		return "all", nil
	default:
		return "active", map[string]bool{
			models.OrderStatusOpen:            true,
			models.OrderStatusPartiallyFilled: true,
		}
	}
}

func matchesStatusFilter(status string, filter map[string]bool) bool {
	if len(filter) == 0 {
		return true
	}
	return filter[status]
}

func statusFromProgress(quantity, filled int64) string {
	if quantity <= 0 {
		return models.OrderStatusOpen
	}
	if filled >= quantity {
		return models.OrderStatusFilled
	}
	if filled > 0 {
		return models.OrderStatusPartiallyFilled
	}
	return models.OrderStatusOpen
}

func formatOrderPair(symbol string, pairBySymbol map[string]string) string {
	key := strings.ToLower(strings.TrimSpace(symbol))
	if pair, ok := pairBySymbol[key]; ok && pair != "" {
		return pair
	}

	upper := strings.ToUpper(strings.TrimSpace(symbol))
	if strings.HasSuffix(upper, "USDT") && len(upper) > len("USDT") {
		return upper[:len(upper)-len("USDT")] + "/USDT"
	}

	if upper == "" {
		return "UNKNOWN/USDT"
	}

	return upper + "/USDT"
}

func orderSide(bid bool) string {
	if bid {
		return models.OrderSideBuy
	}
	return models.OrderSideSell
}

func canCancelOrder(order *models.Order) bool {
	if order == nil {
		return false
	}
	if order.EngineOrderID == 0 {
		return false
	}
	if !strings.EqualFold(order.OrderType, models.OrderTypeLimit) {
		return false
	}
	status := strings.ToUpper(strings.TrimSpace(order.Status))
	if status != models.OrderStatusOpen && status != models.OrderStatusPartiallyFilled {
		return false
	}
	return order.Remaining() > 0
}

func (app *application) unlockWalletUpTo(userID, amount int64) error {
	if amount <= 0 {
		return nil
	}

	wallet, err := app.wallet.GetWallet(userID)
	if err != nil {
		return err
	}

	unlockAmount := amount
	if wallet.Locked < unlockAmount {
		unlockAmount = wallet.Locked
	}
	if unlockAmount <= 0 {
		return nil
	}

	_, err = app.wallet.UnlockAmount(userID, unlockAmount)
	return err
}
