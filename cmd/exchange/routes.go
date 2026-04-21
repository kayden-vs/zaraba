package main

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/kayden-vs/zaraba/ui"
)

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(secureHeaders)

	staticFS, err := fs.Sub(ui.Files, "static")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(staticFS))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	r.Get("/ping", ping)
	r.Head("/ping", ping)

	r.Get("/sse/markets", app.SseMarketHandler)
	r.Get("/sse/orderbook", app.SseOrderBookHandler)
	r.Get("/sse/trades", app.SseTradesHandler)

	r.Group(func(r chi.Router) {
		r.Use(app.sessionManager.LoadAndSave)
		r.Use(noSurf)
		r.Use(app.authenticate)

		r.Get("/", app.HomeHandler)
		r.Get("/markets", app.MarketsHandler)

		r.Get("/trade/{symbol}", app.TradeHandler)

		r.Get("/user/signup", app.userSignup)
		r.Post("/user/signup", app.userSignupPost)
		r.Get("/user/login", app.userLogin)
		r.Post("/user/login", app.userLoginPost)

		//  -- authenticated only routes --
		r.Group(func(r chi.Router) {
			r.Use(app.requireAuthentication)

			r.Post("/user/logout", app.userLogoutPost)
			r.Post("/trade/{symbol}/placemarketorder", app.PlaceMarketOrderPost)
			r.Post("/trade/{symbol}/placelimitorder", app.PlaceLimitOrderPost)
			r.Get("/orders", app.OrdersHandler)
			r.Post("/orders/{id}/cancel", app.CancelOrderPost)
			r.Get("/user/wallet", app.WalletHandler)
			r.Post("/api/wallet/transactions", app.WalletHandlerPost)
		})
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w, r)
	})

	return r
}
