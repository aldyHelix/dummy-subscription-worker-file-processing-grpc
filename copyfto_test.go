package copyfto

import (
	"context"
	"testing"

	"github.com/citradigital/toldata"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTolDataHealthCheck(t *testing.T) {
	api := createCopyfto()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := api.ToldataHealthCheck(ctx, &toldata.Empty{})
	assert.Equal(t, nil, err)
}

func TestCopyFileDummyOrder(t *testing.T) {
	api := createCopyfto()
	err := api.Work(uuid.NewString())
	assert.Equal(t, nil, err)
}

// func TestGetFileFTO(t *testing.T) {
// 	api := createCopyfto()

// 	_, err := api.GetFileFTO("ORD-TA-NW-UP-20210225110110-0001.zip")
// 	assert.Equal(t, nil, err)
// }
