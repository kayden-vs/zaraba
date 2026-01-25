package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/dashboard", app.dashboard)
	r.Get("/placeorder", app.PlaceOrder)
	r.Post("/placeorder", app.PlaceOrderPost)

	return r
}
