package copyfto

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/citradigital/toldata"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
)

const DefaultReadInterval = 30

func (c *CopyftoAPI) initDb() {
	if os.Getenv("UNIT_TEST") == "1" {
		return
	}

	connString := os.Getenv("DB_CONNECTION_STRING_TKP")
	backupConnString := os.Getenv("DB_CONNECTION_STRING")
	if connString == "" {
		connString = backupConnString
	}
	if connString == "" {
		Log(context.Background(), zerolog.FatalLevel, &RequestInfo{}, "Err Invalid ENV", "NOK", "please provide DB_CONNECTION_STRING_TKP or DB_CONNECTION_STRING")
	}

	pool, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		log.Println(err)
		connString = backupConnString
		pool, err = pgxpool.Connect(context.Background(), connString)
		if err != nil {
			log.Panic(err)
			return
		}
	}
	c.Db = pool
	msg := fmt.Sprintf("Connected to %s", connString)
	splitted := strings.Split(connString, "@")
	if len(splitted) > 1 {
		msg = fmt.Sprintf("Connected to %s", splitted[1])
	}
	Log(context.Background(), zerolog.InfoLevel, &RequestInfo{}, msg, "OK", "")
}

func SetupAPI() (*http.ServeMux, *CopyftoAPI) {
	api := createCopyftoAPI()
	api.initDb()
	go func() {
		if os.Getenv("UNIT_TEST") == "1" {
			return
		}
		if os.Getenv("DUMMY_CFTO_DISABLE_SCHEDULER") == "true" {
			Log(context.Background(), zerolog.InfoLevel, &RequestInfo{}, "DUMMY_CFTO_DISABLE_SCHEDULER is set to true, disabling scheduler", "OK", "")
			return
		}
		for {
			interval, err := strconv.Atoi(os.Getenv("FTO_READ_INTERVAL"))
			if err != nil {
				interval = DefaultReadInterval
				msg := fmt.Sprintln(err.Error(), "FTO_READ_INTERVAL invalid, fallback to default: "+fmt.Sprint(interval))
				Log(context.Background(), zerolog.ErrorLevel, &RequestInfo{}, "Err Invalid ENV", "NOK", msg)
			}
			api.Work(uuid.NewString())
			time.Sleep(time.Duration(interval) * time.Minute)
		}
	}()

	r := http.NewServeMux()
	//add here for custom handle function

	return r, api
}

func StartHTTP(ctx context.Context, port string, started *chan bool, mux *http.ServeMux, api *CopyftoAPI) *http.Server {
	rest, err := NewCopyftoREST(ctx, toldata.ServiceConfiguration{URL: ""})
	if err != nil {
		log.Println(err)
		return nil
	}

	rest.InstallCopyftoMux(mux)
	rest.Service.SetBuslessObject(api)

	srv := &http.Server{
		Handler:      mux,
		Addr:         port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	go func() {
		log.Fatal(srv.ListenAndServe())
	}()

	log.Println("API started in localhost", port)
	*started <- true

	return srv
}
