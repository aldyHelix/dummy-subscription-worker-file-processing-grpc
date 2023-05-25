package copyfto

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

type CopyftoAPI struct {
	Db       PgxIface
	Fixtures Fixtures
}

type PgxIface interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}
type FTOInfo struct {
	Filename            string
	NormalFileSize      int64
	NumberOfTransaction int
}

type GenericRefDataResponse struct {
	Result      []GenericRefDataItem `json:"result"`
	TotalResult int                  `json:"totalResult"`
}

type GenericRefDataItem struct {
	ID    string `json:"id"`
	Induk string `json:"induk"`
	Kode  string `json:"kode"`
	Label string `json:"label"`
	Nilai string `json:"nilai"`
}

type CreateTicketResponse struct {
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}
type SwitchingValidationData struct {
	ReferenceNo          string
	TrxType              string
	NetTransactionAmount float64
	AmountNominal        float64
}
