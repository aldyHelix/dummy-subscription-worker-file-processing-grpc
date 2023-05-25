package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	grpc "google.golang.org/grpc"

	api "copyfto"

	"github.com/citradigital/toldata"
	"github.com/rs/zerolog"
)

func main() {
	log.SetOutput(os.Stdout)
	log.Println("Init Logger")
	api.InitLogger()
	log.Println("Done Init Logger")
	grpcServer := grpc.NewServer()
	ctx, cancel := context.WithCancel(context.Background())
	_, grpcapi := api.SetupAPI()

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		api.Log(context.Background(), zerolog.FatalLevel, &api.RequestInfo{}, "Err Invalid ENV", "NOK", "Please specify GRPC_PORT as port where this GRPC server should serve")
	}

	conn, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		api.Log(context.Background(), zerolog.FatalLevel, &api.RequestInfo{}, "Err Failed to Listen", "NOK", err.Error())
	}

	grpc, err := api.NewCopyftoGRPC(ctx, toldata.ServiceConfiguration{URL: ""})
	if err != nil {
		api.Log(context.Background(), zerolog.FatalLevel, &api.RequestInfo{}, "Err Failed to Create Service", "NOK", err.Error())
	}
	grpc.Service.SetBuslessObject(grpcapi)

	api.RegisterCopyftoServer(grpcServer, grpc)
	api.Log(context.Background(), zerolog.InfoLevel, &api.RequestInfo{}, "Starting GRPC Server", "OK", "")
	go grpcServer.Serve(conn)
	api.Log(context.Background(), zerolog.InfoLevel, &api.RequestInfo{}, "GRPC server started!", "OK", "")

	// scheduler COPY FILE FTO
	if os.Getenv("NEWTKP_INSERT_CFTO_WROKER") == "true" {
		log.Println("starting worker insert cfto")
		grpcapi.CopyFileDummyOrder(ctx, &api.Empty{})
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		_ = <-sigs
		done <- true
	}()

	api.Log(context.Background(), zerolog.InfoLevel, &api.RequestInfo{}, "Starting GRPC API", "OK", "")
	<-done
	cancel()
	api.Log(context.Background(), zerolog.InfoLevel, &api.RequestInfo{}, "Stopped GRPC API", "OK", "")
}
