package main

import (
	"database/sql"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form"
	"github.com/kayden-vs/zaraba/internal/models"
	"github.com/kayden-vs/zaraba/internal/service"
	"github.com/kayden-vs/zaraba/pb"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

type application struct {
	errorLog       *log.Logger
	infoLog        *log.Logger
	formDecoder    *form.Decoder
	users          models.UserModelInterface
	wallet         models.WalletModelInterface
	sessionManager *scs.SessionManager
}

func main() {
	addr := flag.String("addr", ":8080", "HTTP server port")
	dsn := flag.String("dsn", "postgres://rohit:eren@/exchange?host=/var/run/postgresql&sslmode=disable", "PostgreSQL data source name")
	flag.Parse()

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	db, err := openDB(*dsn)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer db.Close()

	formDecoder := form.NewDecoder()

	sessionManager := scs.New()
	sessionManager.Store = postgresstore.New(db)
	sessionManager.Lifetime = 12 * time.Hour

	// Start WebSocket price fetcher
	go service.StartPriceFetcher()

	app := &application{
		errorLog:       errorLog,
		infoLog:        infoLog,
		formDecoder:    formDecoder,
		users:          &models.UserModel{DB: db},
		wallet:         &models.WalletModel{DB: db},
		sessionManager: sessionManager,
	}

	srv := &http.Server{
		Addr:         *addr,
		ErrorLog:     errorLog,
		Handler:      app.routes(),
		IdleTimeout:  1 * time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start gRPC server in a goroutine
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		errorLog.Fatalf("failed to listen on port 50051: %v", err)
	}

	grpcServer := grpc.NewServer()
	exchangeServer := service.NewExchangeServer()
	pb.RegisterExchangeServer(grpcServer, exchangeServer)

	go func() {
		infoLog.Printf("Starting gRPC server on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			errorLog.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	infoLog.Printf("Starting HTTP server on %s", *addr)
	err = srv.ListenAndServe()
	errorLog.Fatal(err)
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
