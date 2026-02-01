package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bookstore/internal/httpapi"
	"bookstore/internal/seed"
	"bookstore/internal/store"
	"bookstore/internal/worker"
)

func main() {
	s := store.New()
	seed.MustSeed(s)

	orderQueue := make(chan int, 100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p := worker.OrderProcessor{Store: s, OrderJobs: orderQueue}
	go p.Run(ctx)

	api := &httpapi.API{Store: s, OrderQueue: orderQueue}
	r := httpapi.NewRouter()
	api.RegisterRoutes(r)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Println("listening on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	cancel()

	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelTimeout()
	_ = srv.Shutdown(ctxTimeout)
}
