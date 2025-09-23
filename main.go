package main

import (
	"go-tenders-v3/api"
	"go-tenders-v3/config"
	"go-tenders-v3/storage"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.LoadConfig()

	// Подключаемся к базе данных Postgres по строке подключения из конфигурации
	db, err := sqlx.Connect("postgres", cfg.PostgresConn)
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}

	// Инициализируем слой хранения
	store := storage.NewStorage(db)

	bidHandler := api.NewBidHandler(store)
	tenderHandler := api.NewTenderHandler(store)
	serviceHandler := api.NewServiceHandler()

	handlers := &api.Handlers{
		Bid:     bidHandler,
		Tender:  tenderHandler,
		Service: serviceHandler,
	}

	router := api.NewRouter(handlers)

	log.Printf("Server started at %s", cfg.ServerAddress)
	log.Fatal(http.ListenAndServe(cfg.ServerAddress, router))
}
