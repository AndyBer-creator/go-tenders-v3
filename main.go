package main

import (
	"go-tenders-v3/api"
	"go-tenders-v3/storage"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Подключаемся к базе (укажите свои параметры подключения)
	db, err := sqlx.Connect("postgres", "user=postgres dbname=mydb sslmode=disable")
	if err != nil {
		log.Fatal("DB connection error:", err)
	}

	// Инициализируем слой storage
	store := storage.NewStorage(db)
	log.Printf("Server started")
	bidHandler := api.NewBidHandler(store)
	tenderHandler := api.NewTenderHandler(store)
	serviceHandler := api.NewServiceHandler()

	handlers := &api.Handlers{
		Bid:     bidHandler,
		Tender:  tenderHandler,
		Service: serviceHandler,
	}
	router := api.NewRouter(handlers)

	log.Fatal(http.ListenAndServe(":8080", router))
}
