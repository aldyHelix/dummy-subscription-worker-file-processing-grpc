package main

import (
	"context"
	"copyfto"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.SetOutput(os.Stdout)
	log.Println("Init Logger")
	copyfto.InitLogger()
	log.Println("Done Init Logger")

	started := make(chan bool, 1)
	mux, api := copyfto.SetupAPI()

	ctx, cancel := context.WithCancel(context.Background())

	srv := copyfto.StartHTTP(ctx, ":8000", &started, mux, api)

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		_ = <-sigs
		done <- true
	}()
	<-done
	cancel()

	log.Println("server stoped")
	srv.Shutdown(context.Background())
}
