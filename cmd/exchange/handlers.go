package main

import (
	"fmt"
	"net/http"

	"github.com/kayden-vs/zaraba/ui/html/pages"
)

func (app *application) markets(w http.ResponseWriter, r *http.Request) {
	err := pages.HomePage().Render(r.Context(), w)
	if err != nil {
		fmt.Println(err)
	}
}

func (app *application) PlaceOrderPost(w http.ResponseWriter, r *http.Request) {

}

func (app *application) HomeHandler(w http.ResponseWriter, r *http.Request) {}

func (app *application) MarketsHandler(w http.ResponseWriter, r *http.Request) {}

func (app *application) TradeHandler(w http.ResponseWriter, r *http.Request) {}

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {}

func ping(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}
