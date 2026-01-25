package main

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kayden-vs/zaraba/ui"
)

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	staticFS, err := fs.Sub(ui.Files, "static")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(staticFS))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	r.Get("/ping", ping)

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
