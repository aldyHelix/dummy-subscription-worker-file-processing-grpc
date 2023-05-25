package copyfto

import (
	"context"
	"log"
	"os"
	"testing"
	"time"
)

const (
	HTTP_PORT_TEST = ":12921"
)

var d *CopyftoAPI

var httpFixtures *TestFixtures

func TestMain(m *testing.M) {
	log.SetOutput(os.Stdout)
	log.Println("Init Logger")
	InitLogger()
	log.Println("Done Init Logger")
	ctx, cancel := context.WithCancel(context.Background())
	api := createCopyfto()

	log.Println("init")
	startHttp(ctx, api)
	code := m.Run()
	cancel()
	os.Exit(code)
}

func startHttp(ctx context.Context, api *CopyftoAPI) {
	started := make(chan bool, 1)
	mux, api := SetupAPI()
	httpFixtures = CreateFixtures("dummy-api")
	api.Fixtures = httpFixtures

	go StartHTTP(ctx, HTTP_PORT_TEST, &started, mux, api)

	log.Println("waiting-start-http-test")
	_ = <-started
	time.Sleep(1000)
}
