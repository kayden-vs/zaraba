package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/alexedwards/scs/v2"
)

type Config struct {
	port string
}

type application struct {
	sessionManager *scs.SessionManager
}

func main() {
	config := Config{}
	app := application{}
	port := flag.String("port", ":8080", "port for http server")
	flag.Parse()

	config.port = *port

	fmt.Printf("Listening on port %s\n", config.port)
	err := http.ListenAndServe(config.port, app.routes())
	if err != nil {
		fmt.Println(err)
	}
}
