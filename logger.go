package copyfto

import (
	"context"
	defaultlog "log"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/peer"
)

const (
	LevelKey = "LOG_LEVEL"
)

func InitLogger() {
	lvl, err := strconv.Atoi(os.Getenv(LevelKey))
	if err != nil || lvl < 0 || lvl > 7 {
		lvl = 7
	}
	defaultlog.Println("log level: ", lvl, zerolog.Level(lvl).String())
	zerolog.SetGlobalLevel(zerolog.Level(lvl))
	zerolog.TimestampFieldName = "timestamp"
	zerolog.LevelFieldName = "log_level"
	zerolog.LevelFieldMarshalFunc = func(l zerolog.Level) string { return strings.ToUpper(l.String()) }
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05.345 +0700"
	log.Info().Msg("log configured")
}
func Log(ctx context.Context, level zerolog.Level, reqinfo *RequestInfo, message, status, description string) {
	var event *zerolog.Event
	logger := log.Output(os.Stdout)
	switch level {
	case zerolog.DebugLevel:
		event = logger.Debug()
	case zerolog.InfoLevel:
		event = logger.Info()
	case zerolog.WarnLevel:
		event = logger.Warn()
	case zerolog.ErrorLevel:
		logger = logger.Output(os.Stderr)
		event = logger.Error()
	case zerolog.FatalLevel:
		logger = logger.Output(os.Stderr)
		event = logger.Fatal()
	case zerolog.PanicLevel:
		logger = logger.Output(os.Stderr)
		event = logger.Panic()
	default:
		event = logger.Info()
	}
	p, ok := peer.FromContext(ctx)
	if ok && p != nil {
		ip := p.Addr.String()
		if ip != "" {
			reqinfo.IpAddress = ip
		}
	}
	event.
		Str("ip_address", reqinfo.IpAddress).
		Str("hostname", os.Getenv("HOSTNAME")).
		Str("endpoint", reqinfo.Endpoint).
		Str("request_id", reqinfo.RequestId).
		Str("username", reqinfo.Username).
		Str("system", "INTEGRASI").
		Str("subsystem", "SINT").
		Str("app", K8_SVC_NAME).
		Str("resource", reqinfo.Resource).
		Str("action", reqinfo.Action).
		Str("status", status).
		Str("description", description).
		Msg(message)
}
func naiveAtoI(s string) (i int) {
	i, _ = strconv.Atoi(s)
	return
}
