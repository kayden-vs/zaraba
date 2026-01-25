package main

import (
	"fmt"
	"net/http"

	"github.com/kayden-vs/zaraba/ui/html/pages"
)

func (app *application) dashboard(w http.ResponseWriter, r *http.Request) {
	err := pages.HomePage().Render(r.Context(), w)
	if err != nil {
		fmt.Println(err)
	}
}

func (app *application) PlaceOrder(w http.ResponseWriter, r *http.Request) {

}

func (app *application) PlaceOrderPost(w http.ResponseWriter, r *http.Request) {

}
