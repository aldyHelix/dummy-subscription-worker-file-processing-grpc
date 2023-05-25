package copyfto

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	math "math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/citradigital/toldata"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"
	"google.golang.org/api/option"
)

const (
	FUNC_TAPERA_ORDER = "Dummy Order"
	ACTION_INSERT     = "d0bdf085-84a9-45cf-9bca-91c0787c17c7"
	// R067
	STATUS_TERSIMPAN  = "1c2bc45d-f9df-4a92-a074-26c58e988629"
	STATUS_TERKENDALA = "aa650e95-163b-47bd-996b-685a53df5d17"
	STATUS_TERSEDIA   = "994459d5-7ecb-4d8d-9c8d-801ca3f25c4d"

	// R068
	NEW       = "b09cec43-5c2b-4860-bb04-e112e8db394c"
	ERROR     = "c396eccb-a160-47e8-83da-207b63891269"
	UNMATCHED = "18a16fd5-93d2-485d-af5d-aa489d93b16d"
	INVALID   = "e93f5414-af74-435a-8025-e123c3905f20"

	SUBS = "26a3ef20-86c3-4ad6-9a1d-ab8acd2b5998"
	REDM = "b637ab18-247b-487d-898b-d97fff1691ae"
	SWTC = "590e60a5-1051-4fb3-9061-18dfc579d137"

	ERR_DB_QUERY_MSG = "Err DB Query"
	ERR_DB_SCAN_MSG  = "Err DB Row Scan"

	K8_SVC_NAME = "newabc-insert-fto-grpc"
)

func createCopyftoAPI() *CopyftoAPI {
	return &CopyftoAPI{}
}

func (c *CopyftoAPI) CopyFileDummyOrder(ctx context.Context, req *Empty) (*GenericReponse, error) {
	fmt.Println("ABC")
	res := &GenericReponse{
		Success:   true,
		RequestId: uuid.NewString(),
	}
	go c.Work(res.RequestId)
	return nil, nil
}

func (c *CopyftoAPI) getSavedFTO(logctx context.Context, reqInfo *RequestInfo, processLimit int) ([]*DummyOrderInfo, error) {

	var results []*DummyOrderInfo

	rows, err := c.Db.Query(logctx, `
		SELECT 
			id, nama_file, COALESCE(ukuran_file_normal,0), ukuran_file_zip,
			COALESCE(jumlah_transaksi, 0), tanggal_perolehan, status
		FROM dummy_order_info
		WHERE status = $1 LIMIT $2	
	`, STATUS_TERSIMPAN, processLimit)
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, ERR_DB_QUERY_MSG, "NOK", err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item DummyOrderInfo
		var obtainedDate time.Time
		err = rows.Scan(
			&item.Id,
			&item.FileName,
			&item.NormalFileSize,
			&item.ZipFileSize,
			&item.NumberOfTransaction,
			&obtainedDate,
			&item.FileStatus,
		)

		if err != nil {
			Log(logctx, zerolog.ErrorLevel, reqInfo, ERR_DB_SCAN_MSG, "NOK", err.Error())
			return nil, err
		}

		item.ObtainedDate = obtainedDate.Unix()
		results = append(results, &item)
		msg := fmt.Sprintf("Got File %s, ID %s", item.FileName, item.Id)
		Log(logctx, zerolog.InfoLevel, reqInfo, msg, "OK", "")
	}
	return results, nil
}

func (c *CopyftoAPI) GetFileFTOAli(logctx context.Context, reqInfo *RequestInfo, fileName string) (*FTOInfo, error) {
	bucket, err := prepareAliBucket()
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Reading Ali Bucket", "NOK", err.Error())
		return nil, err
	}

	fmt.Println(bucket.GetObject(fileName))
	return nil, nil
}

func (c *CopyftoAPI) GetFileFTO(logctx context.Context, reqInfo *RequestInfo, fileName string) (*FTOInfo, error) {
	ctx := context.Background()
	objPath := os.Getenv("FTO_LOCAL_DIRECTORY")
	gcsBucketName := os.Getenv("OBJECT_STORAGE_BUCKET_NAME")
	if gcsBucketName == "" {
		gcsBucketName = "gs:://dummy-bucket/"
	}
	gcsBucketName = strings.Split(gcsBucketName, "/")[2]
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(os.Getenv("GCS_CREDENTIAL_PATH")))
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Create Storage Client", "NOK", err.Error())
		return nil, err
	}
	defer client.Close()

	path := "dummy/" + objPath
	rc, err := client.Bucket(gcsBucketName).Object(path + "/" + fileName).NewReader(ctx)
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Create Storage Reader For File: "+fileName, "NOK", err.Error())
		return nil, err
	}
	defer rc.Close()

	slurp, err := ioutil.ReadAll(rc)
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Reading File: "+fileName, "NOK", err.Error())
		return nil, err
	}

	err = ioutil.WriteFile("/tmp/"+fileName, slurp, 0644)
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Writing File: "+fileName, "NOK", err.Error())
	}

	err = Unzip("/tmp/"+fileName, "/tmp")
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Unzipping File: "+fileName, "NOK", err.Error())
		return nil, err
	}

	parseFileName := strings.Split(fileName, ".")
	txtName := "/tmp/" + parseFileName[0] + ".txt"
	file, err := os.Open(txtName)
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Opening File: "+txtName, "NOK", err.Error())
		return nil, err
	}
	defer file.Close()
	lines, err := lineCounter(file)
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Counting Line In File: "+txtName, "NOK", err.Error())
		return nil, err
	}
	fileInfo, err := os.Stat(txtName)
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Get File Stat: "+txtName, "NOK", err.Error())
		return nil, err
	}
	msg := fmt.Sprintf("File Size = %d, Line = %d", fileInfo.Size(), lines)
	Log(logctx, zerolog.InfoLevel, reqInfo, msg, "OK", "")
	return &FTOInfo{
		Filename:            txtName,
		NormalFileSize:      fileInfo.Size(),
		NumberOfTransaction: lines - 1,
	}, nil
}
func (c *CopyftoAPI) Work(reqId string) error {
	logctx := context.Background()
	reqInfo := &RequestInfo{RequestId: reqId}
	Log(logctx, zerolog.InfoLevel, reqInfo, "START", "OK", "Starting CFTO Process")
	defer Log(logctx, zerolog.InfoLevel, reqInfo, "STOP", "OK", "Stopping CFTO Process")

	ftoProcessLimit := os.Getenv("FTO_PROCESS_LIMIT")
	processLimit, err := strconv.Atoi(ftoProcessLimit)

	if processLimit < 1 {
		processLimit = 1
	}

	c.ensureDBConn()
	savedDummyOrderInfo, err := c.getSavedFTO(logctx, reqInfo, processLimit)
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Get Saved FTO", "NOK", err.Error())
		return err
	}

	if len(savedDummyOrderInfo) < 1 {
		// sleep
		Log(logctx, zerolog.InfoLevel, reqInfo, "No Records Found", "OK", "")
		return err
	}

	for _, val := range savedDummyOrderInfo {
		fmt.Println(val)
	}
	return nil
}
func (c *CopyftoAPI) UpdateMetadataFTO(logctx context.Context, reqInfo *RequestInfo, id string, status string, data *FTOInfo) error {
	var err error
	if data != nil {
		_, err = c.Db.Exec(logctx, `
			UPDATE dummy_order_info SET 
				ukuran_file_normal = $1,
				jumlah_transaksi = $2,
				status = $3,
				updated_by = 'abc-copy-file-to',
				updated_at = now()
			WHERE id = $4`,
			data.NormalFileSize,
			data.NumberOfTransaction,
			status,
			id,
		)
	} else {
		_, err = c.Db.Exec(logctx, `
			UPDATE dummy_order_info SET 
				status = $1,
				updated_by = 'abc-copy-file-to',
				updated_at = now()
			WHERE id = $2`,
			status,
			id,
		)
	}

	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, ERR_DB_QUERY_MSG, "NOK", err.Error())
		return err
	}
	statusObj, err := getReferenceData(status, false)
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Get Reference Data", "NOK", err.Error())
		return err
	}

	if statusObj.TotalResult < 1 {
		err = errors.New("status " + status + " is not found in data reference")
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Get Reference Data", "NOK", err.Error())
		return err
	}

	_, err = c.Db.Exec(logctx, `
		INSERT INTO riwayat_status_fto_info (
			dummy_order_info,
			status_id,
			status_label,
			created_at,
			created_by,
			updated_at,
			updated_by
		) VALUES ($1, $2, $3, now(), 'abc-copy-file-to', now(), 'abc-copy-file-to')
	`, id, status, statusObj.Result[0].Label)

	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, ERR_DB_QUERY_MSG, "NOK", err.Error())
	}

	return nil
}

func (c *CopyftoAPI) InsertFTODetail(logctx context.Context, reqInfo *RequestInfo, idDummyOrderInfo string, info *FTOInfo) error {
	file, err := os.Open(info.Filename)
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Opening File "+info.Filename, "NOK", err.Error())
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	//scan first to skip header
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		Log(logctx, zerolog.FatalLevel, reqInfo, "Err Scanning File "+info.Filename, "NOK", err.Error())
	}
	maxInsert := 1000 // max insert per batch
	trx := 0
	for i := 0; i < int(math.Ceil(float64(info.NumberOfTransaction)/float64(maxInsert))); i++ {
		var processedRecords int
		processedRecords, err = c.processBatch(logctx, reqInfo, idDummyOrderInfo, info, maxInsert, scanner)
		trx += processedRecords
		desc := fmt.Sprintf("transaction recorded %d", trx)
		Log(logctx, zerolog.DebugLevel, reqInfo, "Checkpoint", "OK", desc)
	}
	msg := fmt.Sprintf("transaction recorded %d", trx)
	Log(logctx, zerolog.DebugLevel, reqInfo, msg, "OK", "Done Inserting FTO Detail")

	return nil
}

func (c *CopyftoAPI) processBatch(
	logctx context.Context,
	reqInfo *RequestInfo,
	idDummyOrderInfo string,
	info *FTOInfo,
	maxInsert int,
	scanner *bufio.Scanner,
) (int, error) {
	processedRecords := 0
	var history []*DummyOrderInfoHistory
	var unPairedSwitchings [][]string
	var pairedSwtcID []string
	valueStrings := []string{}
	valueArgs := []interface{}{}
	mapper := make(map[string]int)

	query := `
	INSERT INTO dummy_order_detail (
		dummy_order_info,
		transaction_date,
		transaction_type,
		instruction_type,
		sa_code,
		ifua,
		fund_code,
		amount_nominal,
		amount_unit,
		amount_all_unit,
		fee_nominal,
		fee_unit,
		fee_percent,
		redem_payment_ac_sequential_code,
		redem_payment_bank_bic_code,
		redem_payment_bank_bi_member_code,
		redm_payment_ac_no,
		payment_date,
		transfer_type,
		reference_no,
		number_of_units_allocated,
		nav_per_unit,
		gross_transaction_amount,
		transaction_fee_nominal,
		net_transaction_amount,
		outstanding_number_of_units,
		status_transaksi, 
		status_proses, 
		deskripsi_status,
		created_at,
		created_by,
		updated_at,
		updated_by
	)`
	if info.NumberOfTransaction < maxInsert {
		maxInsert = info.NumberOfTransaction
	}

	for j := 0; j < maxInsert; j++ {
		var newQueryLine string
		var params []interface{}
		var err error
		newQueryLine, params, unPairedSwitchings, pairedSwtcID, err = c.createBatchQuery(logctx, reqInfo, scanner, idDummyOrderInfo, len(valueArgs), mapper, unPairedSwitchings, pairedSwtcID)
		if err != nil {
			Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Create Batch Query", "NOK", err.Error())
			return processedRecords, err
		}
		if len(newQueryLine) > 0 {
			valueStrings = append(valueStrings, newQueryLine)
			valueArgs = append(valueArgs, params...)
			processedRecords++
		}
	}
	for _, unPairedSwitching := range unPairedSwitchings {
		newQueryLine, params := c.processRecord(logctx, reqInfo, idDummyOrderInfo, SWTC, unPairedSwitching, len(valueArgs), "", true, mapper)
		valueStrings = append(valueStrings, newQueryLine)
		valueArgs = append(valueArgs, params...)
		processedRecords++
	}
	stmt := fmt.Sprintf("%s VALUES %s RETURNING id, status_transaksi, status_proses, deskripsi_status", query, strings.Join(valueStrings, ","))
	Log(logctx, zerolog.DebugLevel, reqInfo, "START: Insert fto_detail query", "OK", "")
	rows, err := c.Db.Query(logctx, stmt, valueArgs...)
	Log(logctx, zerolog.DebugLevel, reqInfo, "STOP: Insert fto_detail query", "OK", "")
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, ERR_DB_QUERY_MSG, "NOK", err.Error())
		return processedRecords, err
	}
	defer rows.Close()

	for rows.Next() {
		var historyItem DummyOrderInfoHistory
		err = rows.Scan(
			&historyItem.DummyOrderDetail,
			&historyItem.StatusId,
			&historyItem.StatusProcess,
			&historyItem.StatusDescription,
		)

		if err != nil {
			Log(logctx, zerolog.ErrorLevel, reqInfo, ERR_DB_SCAN_MSG, "NOK", err.Error())
			return processedRecords, err
		}

		if historyItem.StatusId == NEW {
			historyItem.StatusLabel = "NEW"
		} else if historyItem.StatusId == ERROR {
			historyItem.StatusLabel = "ERROR"
		}

		history = append(history, &historyItem)
	}
	rows.Close()
	// Insert to history
	err = c.InsertHistoryFTODetail(logctx, reqInfo, history)
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Insert History FTO Detail", "NOK", err.Error())
		return processedRecords, err
	}
	if len(pairedSwtcID) > 0 {
		Log(logctx, zerolog.DebugLevel, reqInfo, "deleting paired swtc", "OK", "")
		_, err = c.Db.Exec(logctx, fmt.Sprintf("DELETE FROM dummy_order_detail WHERE id IN ('%s')", strings.Join(pairedSwtcID, "','")))
	}
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Delete paired swtc records", "NOK", err.Error())
		return processedRecords, err
	}
	Log(logctx, zerolog.DebugLevel, reqInfo, "Memory Usage Before Clean", "OK", GetMemUsage())
	history = nil
	valueStrings = nil
	valueArgs = nil
	// Force GC to clear up, should see a memory drop
	runtime.GC()
	Log(logctx, zerolog.DebugLevel, reqInfo, "Memory Usage After Clean", "OK", GetMemUsage())
	return processedRecords, err
}

func (c *CopyftoAPI) createBatchQuery(
	logctx context.Context,
	reqInfo *RequestInfo,
	scanner *bufio.Scanner,
	idDummyOrderInfo string,
	colIndex int,
	mapper map[string]int,
	unPairedSwitchings [][]string,
	idToBeDeleted []string,
) (newQueryLine string, params []interface{}, newUnPairedSwitchings [][]string, newIdToBeDeleted []string, err error) {
	newIdToBeDeleted = idToBeDeleted
	newUnPairedSwitchings = unPairedSwitchings
	success := scanner.Scan()
	if !success {
		if err = scanner.Err(); err != nil {
			Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Scanning Record", "NOK", err.Error())
			return
		}
	}
	Log(logctx, zerolog.DebugLevel, reqInfo, "Processing line "+fmt.Sprint((colIndex+1)/29), "OK", "")
	record := scanner.Text()

	parseRecord := strings.Split(record, "|")
	if len(parseRecord) > 1 {
		instructionType := getInstructionType(parseRecord[17])
		var pairedSwitching []string
		var pairedToInfo, pairedToID string
		if instructionType == SWTC {
			pairedToID, pairedToInfo, pairedSwitching, newUnPairedSwitchings = c.pairSwitching(logctx, reqInfo, idDummyOrderInfo, parseRecord, newUnPairedSwitchings)
			if len(pairedSwitching) == 0 {
				return
			}
			desc := c.validateSWTCAmount(parseRecord, pairedSwitching)
			newQueryLine, params = c.processRecord(logctx, reqInfo, idDummyOrderInfo, instructionType, parseRecord, colIndex, desc, false, mapper)
			colIndex += 29
			newPairedQueryLine, pairedParams := c.processRecord(logctx, reqInfo, pairedToInfo, instructionType, pairedSwitching, colIndex, desc, false, mapper)
			newQueryLine = fmt.Sprintf("%s, %s", newQueryLine, newPairedQueryLine)
			params = append(params, pairedParams...)
			if len(pairedToID) > 0 {
				newIdToBeDeleted = append(newIdToBeDeleted, pairedToID)
			}
		} else {
			newQueryLine, params = c.processRecord(logctx, reqInfo, idDummyOrderInfo, instructionType, parseRecord, colIndex, "", false, mapper)
		}
	}
	return
}

func (c *CopyftoAPI) processRecord(logctx context.Context, reqInfo *RequestInfo, idDummyOrderInfo, instructionType string, record []string, colIndex int, additionalDesc string, isUnpairedSwitching bool, recordMap map[string]int) (newQueryLine string, params []interface{}) {

	dateLayout := "2006-01-02"
	newQueryLine = fmt.Sprintf(`(
		$%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d,
		$%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, now(), 'abc-copy-file-to', now(), 'abc-copy-file-to'
	)`,
		colIndex+1, colIndex+2, colIndex+3, colIndex+4, colIndex+5,
		colIndex+6, colIndex+7, colIndex+8, colIndex+9, colIndex+10,
		colIndex+11, colIndex+12, colIndex+13, colIndex+14, colIndex+15,
		colIndex+16, colIndex+17, colIndex+18, colIndex+19, colIndex+20,
		colIndex+21, colIndex+22, colIndex+23, colIndex+24, colIndex+25,
		colIndex+26, colIndex+27, colIndex+28, colIndex+29,
	)
	params = append(params, idDummyOrderInfo)
	params = append(params, record[0])
	params = append(params, record[1])
	params = append(params, instructionType)
	params = append(params, record[2])
	params = append(params, record[3])
	params = append(params, record[4])

	amountNominal, _ := strconv.ParseFloat(record[5], 64)
	params = append(params, int64(amountNominal))

	amountUnit, _ := strconv.ParseFloat(record[6], 64)
	params = append(params, roundUp(amountUnit, 4))

	params = append(params, record[7])

	feeNominal, _ := strconv.ParseFloat(record[8], 64)
	params = append(params, feeNominal)

	feeUnit, _ := strconv.ParseFloat(record[9], 64)
	params = append(params, feeUnit)

	feePercent, _ := strconv.ParseFloat(record[10], 64)
	params = append(params, feePercent)

	params = append(params, record[11])
	params = append(params, record[12])
	params = append(params, record[13])
	params = append(params, record[14])

	paymentDate, _ := time.Parse(dateLayout, record[15])
	params = append(params, paymentDate)

	transferType, _ := strconv.Atoi(record[16])
	params = append(params, transferType)

	params = append(params, record[17])

	numberOfUnitsAllocated, _ := strconv.ParseFloat(record[18], 64)
	params = append(params, roundUp(numberOfUnitsAllocated, 4))

	navPerUnit, _ := strconv.ParseFloat(record[19], 64)
	params = append(params, roundUp(navPerUnit, 4))

	grossTransactionAmount, _ := strconv.ParseFloat(record[20], 64)
	params = append(params, grossTransactionAmount)

	transactionFeeNominal, _ := strconv.ParseFloat(record[21], 64)
	params = append(params, transactionFeeNominal)

	netTransactionAmount, _ := strconv.ParseFloat(record[22], 64)
	params = append(params, netTransactionAmount)

	outstandingNumberOfUnits, _ := strconv.ParseFloat(record[23], 64)
	params = append(params, roundUp(outstandingNumberOfUnits, 4))
	if record[24] != "OK" {
		params = append(params, ERROR)
		params = append(params, "ERROR")
		params = append(params, record[24])
		return
	}

	duplicate, _ := c.isDuplicate(logctx, reqInfo, record[0], record[4], record[17])
	if !duplicate {
		recordMap[record[0]+record[4]+record[17]]++
		duplicate = recordMap[record[0]+record[4]+record[17]] > 1
	}
	if duplicate {
		params = append(params, ERROR)
		params = append(params, "ERROR")
		params = append(params, "[FTO] Duplicate data")
		return
	}
	statusID, status, desc := c.validateFTO(
		record[17],
		instructionType,
		record[1],
		numberOfUnitsAllocated,
		amountNominal,
		navPerUnit,
		netTransactionAmount,
		amountUnit,
		outstandingNumberOfUnits,
	)
	if instructionType == SWTC && len(additionalDesc) > 0 && record[1] == "1" {
		statusID = INVALID
		status = "ERROR"
		desc = additionalDesc
	}
	if isUnpairedSwitching {
		statusID = UNMATCHED
		status = "OK"
		desc = ""
	}
	params = append(params, statusID)
	params = append(params, status)
	params = append(params, desc)
	return
}
func (c *CopyftoAPI) isDuplicate(logctx context.Context, reqInfo *RequestInfo, trxDate, ifua, refNo string) (duplicate bool, err error) {
	count := 0
	Log(logctx, zerolog.DebugLevel, reqInfo, "START: Duplicate check query", "OK", "")
	err = c.Db.QueryRow(logctx, `
		SELECT COUNT(1) 
		FROM dummy_order_detail 
		WHERE transaction_date=$1::date AND ifua=$2 AND reference_no=$3 AND status_proses=$4`,
		trxDate, ifua, refNo, "OK").Scan(&count)
	Log(logctx, zerolog.DebugLevel, reqInfo, "STOP: Duplicate check query", "OK", "")
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, ERR_DB_QUERY_MSG, "NOK", "")
		return
	}
	duplicate = count > 0
	return
}
func (c *CopyftoAPI) InsertHistoryFTODetail(logctx context.Context, reqInfo *RequestInfo, history []*DummyOrderInfoHistory) error {
	query := `
	INSERT INTO riwayat_status_fto_detail (
		dummy_order_detail,
		status_id,
		status_label,
		status_proses,
		deskripsi_status,
		created_at,
		created_by,
		updated_at,
		updated_by
	)`

	valueStrings := []string{}
	valueArgs := []interface{}{}

	for index, val := range history {
		colIndex := index * 5

		valueStrings = append(valueStrings, fmt.Sprintf(`(
			$%d, $%d, $%d, $%d, $%d, now(), 'abc-copy-file-to', now(), 'abc-copy-file-to'
		)`,
			colIndex+1, colIndex+2, colIndex+3, colIndex+4, colIndex+5,
		))

		valueArgs = append(valueArgs, val.DummyOrderDetail)
		valueArgs = append(valueArgs, val.StatusId)
		valueArgs = append(valueArgs, val.StatusLabel)
		valueArgs = append(valueArgs, val.StatusProcess)
		valueArgs = append(valueArgs, val.StatusDescription)
	}

	stmt := fmt.Sprintf("%s VALUES %s", query, strings.Join(valueStrings, ","))
	Log(logctx, zerolog.DebugLevel, reqInfo, "START: Insert riwayat_status_fto_detail query", "OK", "")
	_, err := c.Db.Exec(logctx, stmt, valueArgs...)
	Log(logctx, zerolog.DebugLevel, reqInfo, "STOP: Insert riwayat_status_fto_detail query", "OK", "")
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, ERR_DB_QUERY_MSG, "NOK", "")
		return err
	}

	return nil
}

func (c *CopyftoAPI) MoveFileToDone(logctx context.Context, reqInfo *RequestInfo, fileName string) error {
	ctx := context.Background()
	objPath := os.Getenv("FTO_LOCAL_DIRECTORY")
	donePath := os.Getenv("FTO_DONE_DIRECTORY")
	gcsBucketName := os.Getenv("OBJECT_STORAGE_BUCKET_NAME")
	if gcsBucketName == "" {
		gcsBucketName = "gs:://dummy-bucket-upload-jakarta/"
	}
	gcsBucketName = strings.Split(gcsBucketName, "/")[2]
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(os.Getenv("GCS_CREDENTIAL_PATH")))
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Create Storage Client", "NOK", err.Error())
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	srcPath := fmt.Sprintf("dummy/%s/%s", objPath, fileName)
	dstPath := fmt.Sprintf("dummy/%s/%s", donePath, fileName)
	src := client.Bucket(gcsBucketName).Object(srcPath)
	dst := client.Bucket(gcsBucketName).Object(dstPath)

	Log(logctx, zerolog.InfoLevel, reqInfo, "Moving file FTO..", "OK", "")
	if _, err := dst.CopierFrom(src).Run(ctx); err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Copying File From "+srcPath+" To "+dstPath, "NOK", err.Error())
		return err
	}
	if err := src.Delete(ctx); err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Deleting File "+srcPath, "NOK", err.Error())
		return err
	}
	Log(logctx, zerolog.InfoLevel, reqInfo, "Done Moving file FTO", "OK", "From "+srcPath+" To "+dstPath)
	return nil
}

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	for _, f := range r.File {
		err := extractAndWriteFile(f, dest)
		if err != nil {
			return err
		}
	}

	return nil
}

func extractAndWriteFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() {
		if err := rc.Close(); err != nil {
			panic(err)
		}
	}()

	path := filepath.Join(dest, f.Name)

	// Check for ZipSlip (Directory traversal)
	if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
		return fmt.Errorf("illegal file path: %s", path)
	}

	if f.FileInfo().IsDir() {
		os.MkdirAll(path, f.Mode())
	} else {
		os.MkdirAll(filepath.Dir(path), f.Mode())
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer func() {
			if err := f.Close(); err != nil {
				panic(err)
			}
		}()

		_, err = io.Copy(f, rc)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *CopyftoAPI) ToldataHealthCheck(ctx context.Context, req *toldata.Empty) (*toldata.ToldataHealthCheckInfo, error) {

	rows, err := c.Db.Query(ctx, `
		SELECT 1
	`)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var one int
		err = rows.Scan(&one)
		if err != nil {
			return nil, err
		}
	}

	ret := &toldata.ToldataHealthCheckInfo{Data: ""}
	return ret, nil
}

func getReferenceData(id string, isGroup bool) (*GenericRefDataResponse, error) {
	apiUrl := os.Getenv("URL_API_DATA_REFERENSI")
	if os.Getenv("UNIT_TEST") == "1" {
		apiUrl = "http://dev.dummy.com"
	}
	path := "/datareferensi/datareferensi/?id=" + id
	if isGroup {
		path = "/datareferensi/datareferensi?group=" + id
	}
	resp, err := http.Get(apiUrl + path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	data := &GenericRefDataResponse{}
	err = json.Unmarshal(body, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func getInstructionType(refNo string) string {
	switch refNo[:1] {
	case "2":
		return SUBS
	case "3":
		return REDM
	case "4":
		return SWTC
	default:
		return SUBS
	}
}

func (c *CopyftoAPI) CreateFourEyesTicket(logctx context.Context, reqInfo *RequestInfo, id string) (ticketID string, err error) {
	Log(logctx, zerolog.InfoLevel, reqInfo, "Creating Ticket for: "+id, "OK", "")
	body := fmt.Sprintf(
		`{
			"record_id":"%s",
			"action":"%s",
			"function_name":"%s",
			"user_token":"%s",
			"data":%s
		}`,
		id,
		ACTION_INSERT,
		FUNC_TAPERA_ORDER,
		os.Getenv("CFTO_SECRET"),
		strconv.Quote(fmt.Sprintf(`{"id":"%s"}`, id)),
	)
	req, err := http.NewRequest("POST", os.Getenv("URL_API_CREATE_FOUR_EYES_TICKET"), bytes.NewBufferString(body))
	if err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Creating Req", "NOK", err.Error())
		return "", err
	}
	resBytes := []byte(`{"id":"994459d5-7ecb-4d8d-9c8d-801ca3f25c4d"}`)
	if os.Getenv("UNIT_TEST") != "1" {
		Log(logctx, zerolog.DebugLevel, reqInfo, "Sending Request", "OK", body)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Calling: "+os.Getenv("URL_API_CREATE_FOUR_EYES_TICKET"), "NOK", err.Error())
			return "", err
		}
		defer res.Body.Close()
		resBytes, _ = ioutil.ReadAll(res.Body)
		Log(logctx, zerolog.DebugLevel, reqInfo, "Got Response", "OK", string(resBytes))
	}
	ticket := CreateTicketResponse{}
	if err = json.Unmarshal(resBytes, &ticket); err != nil {
		Log(logctx, zerolog.ErrorLevel, reqInfo, "Err Unmarshalling Response", "NOK", err.Error())
		return "", err
	}
	return ticket.Id, nil
}
func (c *CopyftoAPI) validateFTO(refNo, instructionType, trxType string, NumberofUnitsAllocated, AmountNominal, NAVperUnit, NetTransactionAmount, AmountUnit, OutstandingNumberofUnits float64) (statusID, status, description string) {
	statusID = INVALID
	status = "ERROR"
	// switch trxType {
	// case "1":
	// 	if NumberofUnitsAllocated != AmountNominal/NAVperUnit {
	// 		description = fmt.Sprintf("kolom Number of Units Allocated (%v) berbeda dengan hasil perhitungan (%v)", NumberofUnitsAllocated, AmountNominal/NAVperUnit)
	// 		return
	// 	}
	// 	if NetTransactionAmount != NumberofUnitsAllocated*NAVperUnit {
	// 		description = fmt.Sprintf("kolom Net Transaction Amount (%v) berbeda dengan hasil perhitungan (%v)", NetTransactionAmount, NumberofUnitsAllocated*NAVperUnit)
	// 		return
	// 	}
	// case "2":
	// 	if AmountUnit != NumberofUnitsAllocated {
	// 		description = fmt.Sprintf("kolom Amount (Unit) (%v) berbeda kolom Number of Units Allocated (%v)", AmountUnit, NumberofUnitsAllocated)
	// 		return
	// 	}
	// 	if NetTransactionAmount != NumberofUnitsAllocated*NAVperUnit {
	// 		description = fmt.Sprintf("kolom Net Transaction Amount (%v) berbeda dengan hasil perhitungan (%v)", NetTransactionAmount, NumberofUnitsAllocated*NAVperUnit)
	// 		return
	// 	}
	// 	if OutstandingNumberofUnits != 0 {
	// 		description = fmt.Sprintf("kolom Outstanding Number of Units (%v) harusnya 0", OutstandingNumberofUnits)
	// 		return
	// 	}
	// }
	statusID = NEW
	status = "OK"
	return
}
func (c *CopyftoAPI) validateSWTCAmount(firstRecord, secondRecord []string) (description string) {
	var nominal, netAmount string
	switch firstRecord[1] {
	case "1":
		nominal = firstRecord[5]
		netAmount = secondRecord[22]
	case "2":
		nominal = secondRecord[5]
		netAmount = firstRecord[22]
	}
	nominalAmount, err := strconv.ParseFloat(nominal, 64)
	if err != nil {
		description = fmt.Sprintf("kolom Amount (Nominal) transaksi SUBS (%v) tidak valid", nominal)
	}
	nettoAmount, err := strconv.ParseFloat(netAmount, 64)
	if err != nil {
		description = fmt.Sprintf("kolom Net Transaction Amount transaksi REDM (%v) tidak valid", netAmount)
	}
	if nominalAmount != nettoAmount {
		description = fmt.Sprintf("kolom Net Transaction Amount transaksi REDM (%v) berbeda dengan kolom Amount (Nominal) transaksi SUBS (%v)", netAmount, nominal)
		return
	}
	return
}
func (c *CopyftoAPI) pairSwitching(logctx context.Context, reqInfo *RequestInfo, TOInfo string, switchingRecord []string, unPairedSwitchings [][]string) (pairedTOID, pairedTOInfo string, pairedSwitching []string, newUnPairedSwitchings [][]string) {
	for _, unPairedSwitching := range unPairedSwitchings {
		if unPairedSwitching[17] != switchingRecord[17] {
			newUnPairedSwitchings = append(newUnPairedSwitchings, unPairedSwitching)
			continue
		}
		pairedSwitching = unPairedSwitching
		pairedTOInfo = TOInfo
	}
	if len(pairedSwitching) > 0 {
		return
	}
	var transactionDate,
		transactionType,
		saCode,
		ifua,
		fundCode,
		amountNominal,
		amountUnit,
		amountAllUnit,
		feeNominal,
		feeUnit,
		feePercent,
		redemPaymentAcSequentialCode,
		redemPaymentBankBicCode,
		redemPaymentBankBiMemberCode,
		redmPaymentAcNo,
		paymentDate,
		transferType,
		referenceNo,
		numberOfUnitsAllocated,
		navPerUnit,
		grossTransactionAmount,
		transactionFeeNominal,
		netTransactionAmount,
		outstandingNumberOfUnits,
		status string
	err := c.Db.QueryRow(logctx, `
		SELECT
			id,
			dummy_order_info,
			transaction_date::string,
			transaction_type::string,
			sa_code,
			ifua,
			fund_code,
			amount_nominal::string,
			amount_unit::string,
			amount_all_unit::string,
			fee_nominal::string,
			fee_unit::string,
			fee_percent::string,
			redem_payment_ac_sequential_code::string,
			redem_payment_bank_bic_code::string,
			redem_payment_bank_bi_member_code::string,
			redm_payment_ac_no::string,
			payment_date::string,
			transfer_type::string,
			reference_no::string,
			number_of_units_allocated::string,
			nav_per_unit::string,
			gross_transaction_amount::string,
			transaction_fee_nominal::string,
			net_transaction_amount::string,
			outstanding_number_of_units::string, 
			status_proses
		FROM dummy_order_detail 
		WHERE reference_no = $1 AND status_transaksi = $2
	`, switchingRecord[17], UNMATCHED).Scan(
		&pairedTOID,
		&pairedTOInfo,
		&transactionDate,
		&transactionType,
		&saCode,
		&ifua,
		&fundCode,
		&amountNominal,
		&amountUnit,
		&amountAllUnit,
		&feeNominal,
		&feeUnit,
		&feePercent,
		&redemPaymentAcSequentialCode,
		&redemPaymentBankBicCode,
		&redemPaymentBankBiMemberCode,
		&redmPaymentAcNo,
		&paymentDate,
		&transferType,
		&referenceNo,
		&numberOfUnitsAllocated,
		&navPerUnit,
		&grossTransactionAmount,
		&transactionFeeNominal,
		&netTransactionAmount,
		&outstandingNumberOfUnits,
		&status,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			newUnPairedSwitchings = append(unPairedSwitchings, switchingRecord)
		}
		Log(logctx, zerolog.ErrorLevel, reqInfo, ERR_DB_QUERY_MSG, "NOK", err.Error())
		return
	}
	pairedSwitching = []string{
		transactionDate,
		transactionType,
		saCode,
		ifua,
		fundCode,
		amountNominal,
		amountUnit,
		amountAllUnit,
		feeNominal,
		feeUnit,
		feePercent,
		redemPaymentAcSequentialCode,
		redemPaymentBankBicCode,
		redemPaymentBankBiMemberCode,
		redmPaymentAcNo,
		paymentDate,
		transferType,
		referenceNo,
		numberOfUnitsAllocated,
		navPerUnit,
		grossTransactionAmount,
		transactionFeeNominal,
		netTransactionAmount,
		outstandingNumberOfUnits,
		status,
	}
	return
}

func lineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}

// GetMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
func GetMemUsage() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	b, _ := json.Marshal(m)
	return string(b)
}

func roundUp(num float64, precicion float64) float64 {
	precicionStr := fmt.Sprint(precicion)
	str := fmt.Sprintf("%0."+precicionStr+"f", num)
	num, _ = strconv.ParseFloat(str, 64)
	return num
}

func prepareAliBucket() (*oss.Bucket, error) {
	endpoint := os.Getenv("ALI_BUCKET_ENDPOINT")
	accessID := os.Getenv("ALI_BUCKET_ACCESS_ID")
	accessKey := os.Getenv("ALI_BUCKET_ACCESS_KEY")
	bucketName := os.Getenv("ALI_BUCKET_NAME")
	client, err := oss.New(endpoint, accessID, accessKey)
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, err
	}
	return bucket, err
}

// ensureDBConn: check & refreshes DB Connection to ensure no "broken pipe" error
func (c *CopyftoAPI) ensureDBConn() {
	var err error
	for i := 0; i < 3; i++ {
		_, err = c.ToldataHealthCheck(context.Background(), &toldata.Empty{})
		if err == nil {
			return
		}
	}
	c.initDb()
}
