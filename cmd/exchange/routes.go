package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/", app.HomeHandler)
	r.Get("/markets", app.MarketsHandler)
	r.Get("/trade/{symbol}", app.TradeHandler)

	r.Post("/trade/{symbol}/placeorder", app.PlaceOrderPost)

	// -- auth --
	r.Get("/user/signup", app.userSignup)
	r.Post("/user/signup", app.userSignupPost)
	r.Get("/user/login", app.userLogin)
	r.Post("/user/login", app.userLoginPost)
	r.Post("/user/logout", app.userLogoutPost)

	return r
}
