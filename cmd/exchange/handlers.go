package main

import (
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/kayden-vs/zaraba/ui/html/pages"
)

func (app *application) PlaceOrderPost(w http.ResponseWriter, r *http.Request) {}

func (app *application) HomeHandler(w http.ResponseWriter, r *http.Request) {}

func (app *application) MarketsHandler(w http.ResponseWriter, r *http.Request) {
	symbolListProps, err := app.fetchCoinMarket()
	if err != nil {
		fmt.Println(err)
		return
	}

	app.RenderPage(w, r, func(flash string, isAuthenticated bool, csrfToken string) templ.Component {
		return pages.MarketsPage(symbolListProps, "", true, "")
	})
}

func (app *application) TradeHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: make sure symbolName is same as coingecko api call
	symbolID := chi.URLParam(r, "symbol")

	var symbolData pages.CoinMarketProps
	symbolData, err := app.FetchSymbolData(symbolID)
	if err != nil {
		fmt.Println(err)
		return
	}

	tvSymbol := GetTvSymbol(symbolData.Symbol)

	app.RenderPage(w, r, func(flash string, isAuthenticated bool, csrfToken string) templ.Component {
		return pages.TradePage(symbolData, tvSymbol, "", true, "")
	})
}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {}

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}
