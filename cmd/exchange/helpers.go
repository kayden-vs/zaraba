package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/a-h/templ"
	"github.com/go-playground/form"
	"github.com/justinas/nosurf"
	"github.com/kayden-vs/zaraba/ui/html/pages"
)

func (app *application) serverError(w http.ResponseWriter, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	app.errorLog.Output(2, trace) //to get the correct line number of err and avoid err reference to this file

	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}

// RenderPage injects flash and isAuthenticated into the page component.
func (app *application) RenderPage(
	w http.ResponseWriter,
	r *http.Request,
	renderFunc func(flash string, isAuthenticated bool, csrfToken string) templ.Component,
) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	flash := app.sessionManager.PopString(r.Context(), "flash")
	isAuth := app.isAuthenticated(r)
	csrfToken := nosurf.Token(r)
	err := renderFunc(flash, isAuth, csrfToken).Render(r.Context(), w)
	if err != nil {
		// client navigated away mid-render — not a real error
		if errors.Is(err, context.Canceled) {
			return
		}
		app.errorLog.Println(err.Error())
		app.serverError(w, err)
		return
	}
}

func (app *application) decodePostForm(r *http.Request, dst any) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	err = app.formDecoder.Decode(dst, r.PostForm)
	if err != nil {
		var invalidDecoderError *form.InvalidDecoderError

		if errors.As(err, &invalidDecoderError) {
			panic(err)
		}
		return err
	}

	return nil
}

func (app *application) isAuthenticated(r *http.Request) bool {
	isAuthenticated, ok := r.Context().Value(isAuthenticatedContextKey).(bool)
	if !ok {
		return false
	}
	return isAuthenticated
}

func (app *application) fetchCoinMarket() ([]pages.CoinMarketProps, error) {
	url := "https://api.coingecko.com/api/v3/coins/markets" +
		"?vs_currency=usd" +
		"&order=market_cap_desc" +
		"&per_page=10" +
		"&page=1" +
		"&price_change_percentage=24h" +
		fmt.Sprintf("&x_cg_demo_api_key=%s", os.Getenv("API_KEY"))

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var markets []pages.CoinMarketProps
	if err := json.NewDecoder(resp.Body).Decode(&markets); err != nil {
		return nil, err
	}

	return markets, nil
}

func (app *application) FetchSymbolData(symbolID string) (pages.CoinMarketProps, error) {
	url := "https://api.coingecko.com/api/v3/coins/markets" +
		"?vs_currency=usd" +
		"&ids=" + symbolID +
		"&price_change_percentage=24h" +
		fmt.Sprintf("&x_cg_demo_api_key=%s", os.Getenv("API_KEY"))

	resp, err := http.Get(url)
	if err != nil {
		return pages.CoinMarketProps{}, err
	}
	defer resp.Body.Close()
	symbolData := []pages.CoinMarketProps{}

	if err := json.NewDecoder(resp.Body).Decode(&symbolData); err != nil {
		return pages.CoinMarketProps{}, err
	}

	if len(symbolData) == 0 {
		return pages.CoinMarketProps{}, fmt.Errorf("no data found for symbol: %s", symbolID)
	}

	return symbolData[0], nil
}

func GetTvSymbol(symbol string) string {
	var SymbolMap = map[string]string{
		"btc":   "BINANCE:BTCUSDT",
		"eth":   "BINANCE:ETHUSDT",
		"usdt":  "BINANCE:USDTUSD",
		"bnb":   "BINANCE:BNBUSDT",
		"xrp":   "BINANCE:XRPUSDT",
		"usdc":  "BINANCE:USDCUSDT",
		"sol":   "BINANCE:SOLUSDT",
		"trx":   "BINANCE:TRXUSDT",
		"steth": "BINANCE:STETHUSDT",
		"doge":  "BINANCE:DOGEUSDT",
	}

	tvSymbol := SymbolMap[symbol]
	return tvSymbol
}
