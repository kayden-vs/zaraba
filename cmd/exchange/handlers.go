package main

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/kayden-vs/zaraba/ui/html/pages"
)

func (app *application) PlaceOrderPost(w http.ResponseWriter, r *http.Request) {}

func (app *application) HomeHandler(w http.ResponseWriter, r *http.Request) {}

func (app *application) MarketsHandler(w http.ResponseWriter, r *http.Request) {
	symbols := make([]pages.SymbolData, 0)

	BTCdata := pages.SymbolData{
		Name:         "BTCUSDT",
		Price:        65000,
		OneDayChange: "3.2%",
		Volume:       4.54,
	}

	ETHdata := pages.SymbolData{
		Name:         "ETHUSDT",
		Price:        4000,
		OneDayChange: "4.5%",
		Volume:       8343.00,
	}

	symbols = append(symbols, BTCdata)
	symbols = append(symbols, ETHdata)

	app.RenderPage(w, r, func(flash string, isAuthenticated bool, csrfToken string) templ.Component {
		return pages.MarketsPage(symbols, "", true, "")
	})
}

func (app *application) TradeHandler(w http.ResponseWriter, r *http.Request) {
	BTCdata := pages.SymbolData{
		Name:         "BTCUSDT",
		Price:        65000,
		OneDayChange: "3.2%",
		Volume:       4.54,
	}
	app.RenderPage(w, r, func(flash string, isAuthenticated bool, csrfToken string) templ.Component {
		return pages.TradePage(BTCdata, "", true, "")
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
