package main

import (
	"context"
	"log"
	"net/http"

	"github.com/kevinssheva/expenses-tracker/internal/config"
	"github.com/kevinssheva/expenses-tracker/internal/handler"
	"github.com/kevinssheva/expenses-tracker/internal/service"
	"github.com/kevinssheva/expenses-tracker/internal/sheets"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	sheetsClient, err := sheets.NewClient(context.Background(), cfg.GoogleCredentialsFile, cfg.GoogleSheetID, cfg.SheetRange, cfg.Location)
	if err != nil {
		log.Fatalf("create sheets client: %v", err)
	}
	expenseService := service.NewExpenseService(sheetsClient, cfg.Location)

	mux := http.NewServeMux()
	mux.Handle("/expenses", handler.NewExpensesHandler(cfg.APIKey, expenseService))
	mux.Handle("/expenses/", handler.NewExpensesHandler(cfg.APIKey, expenseService))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	addr := ":" + cfg.Port
	log.Printf("starting finance service on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
