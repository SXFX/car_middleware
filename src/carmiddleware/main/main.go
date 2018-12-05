package main

import (
	"carmiddleware/log"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/robfig/cron"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	carAdmissionUrl   = "https://parkinglot.sxfs0351.com/api/createOrder"
	carComeOutUrl     = "https://parkinglot.sxfs0351.com/api/orderFinish"
	cleanDirtyDataUrl = "https://parkinglot.sxfs0351.com/api/questionOrderPush"
	recordTempSelect  = "SELECT id, car_no, in_time, out_time, status, type FROM TC.record_temp"
	recordTempUpdate  = "UPDATE TC.record_temp SET status = ? WHERE id = ?"
	recordTempDelete  = "DELETE FROM TC.record_temp WHERE id = ?"
)

var (
	server   = "47.94.162.149"
	port     = 10034
	user     = "bestbang"
	password = "bestbang"
	database = "1.3.1"
)

func main() {
	log.InitLogger("car_middleware")

	carMiddlewareCon := cron.New()
	spec := "*/3 * * * * ?"
	carMiddlewareCon.AddFunc(spec, func() {
		connectDatabase()
	})
	carMiddlewareCon.Start()
	log.Info("17泊车中间件启动")
	defer carMiddlewareCon.Stop()

	select {}
}

func connectDatabase() {

	connString := fmt.Sprintf("server=%s;port=%d;database=%s;user id=%s;password=%s;encrypt=disable", server, port, database, user, password)

	conn, err := sql.Open("mssql", connString)
	if err != nil {
		log.Fatal("Open Connection failed:", err.Error())
	}
	defer conn.Close()

	stmt, err := conn.Prepare(recordTempSelect)
	if err != nil {
		log.Fatal("Prepare failed:", err.Error())
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		log.Fatal("Query failed:", err.Error())
	}

	cols, err := rows.Columns()
	var colsData = make([]interface{}, len(cols))
	for i := 0; i < len(cols); i++ {
		colsData[i] = new(interface{})
	}

	for rows.Next() {
		_ = rows.Scan(colsData...)
		carMiddleware(colsData, conn)
	}
	defer rows.Close()
}

func carMiddleware(colsData []interface{}, dbConn *sql.DB) {
	recordTempId := (*(colsData[0].(*interface{}))).(int64)
	carNo := (*(colsData[1].(*interface{}))).(string)

	inTime := (*(colsData[2].(*interface{}))).(time.Time)

	carStatus := (*(colsData[4].(*interface{}))).(int64)
	noSenseType := (*(colsData[5].(*interface{}))).(int64)

	inTimeStr := inTime.Format("2006-01-02 15:04:05")

	noSenseTypeStr := strconv.FormatInt(noSenseType, 10)
	carStatusStr := strconv.FormatInt(carStatus, 10)

	if strings.Compare(carNo, "晋-JHN977") == 0 {
		fmt.Println(carNo)
		//return
		if carStatus == 0 {

			var carAdmissionInfo map[string]interface{}

			resultStr := carAdmission(carNo, inTimeStr, noSenseTypeStr)

			_ = json.Unmarshal([]byte(string(resultStr)), &carAdmissionInfo)

			if carAdmissionInfo["success"] == true {

				updateResult := updateRecordTemp(1, recordTempId, dbConn)

				if updateResult == 1 {
					log.Info("updateRecordTemp:", carNo+"入场成功")
				} else {
					log.Error("updateRecordTemp:", carNo+"状态修改失败(入场)")
				}

			} else {
				log.Error("carAdmission:", carNo+carAdmissionInfo["msg"].(string))
			}
		} else if carStatus == 1 {

			outTime := (*(colsData[3].(*interface{}))).(time.Time)
			outTimeStr := outTime.Format("2006-01-02 15:04:05")

			var carOutInfo map[string]interface{}

			resultStr := carComeOut(carNo, inTimeStr, outTimeStr, noSenseTypeStr)

			_ = json.Unmarshal([]byte(string(resultStr)), &carOutInfo)

			if carOutInfo["success"] == true {

				deleteResult := deleteRecordTemp(recordTempId, dbConn)
				if deleteResult == 1 {
					log.Info("deleteResult:", carNo+"出场成功")
				} else {
					log.Error("deleteResult:删除", carNo+"停车信息失败")
				}

			} else {
				log.Error("carComeOut", carNo+carOutInfo["msg"].(string))
			}
		} else if carStatus == -1 || carStatus == -2 {

			var cleanDirtyDataInfo map[string]interface{}

			resultStr := cleanDirtyData(carNo, inTimeStr, carStatusStr)

			_ = json.Unmarshal([]byte(string(resultStr)), &cleanDirtyDataInfo)

			if cleanDirtyDataInfo["success"] == true {

				deleteResult := deleteRecordTemp(recordTempId, dbConn)
				if deleteResult == 1 {
					log.Info("deleteResult:", carNo+"脏数据处理成功")
				} else {
					log.Error("deleteResult:删除", carNo+"脏数据失败")
				}

			} else {
				log.Error("cleanDirtyData", carNo+cleanDirtyDataInfo["msg"].(string))
			}
		}
	}
}

/**
车辆入场
*/
func carAdmission(carNo string, inTimeStr string, noSenseTypeStr string) string {
	resp, err := http.PostForm(carAdmissionUrl,
		url.Values{"carNo": {carNo}, "inTime": {inTimeStr}, "noSenseType": {noSenseTypeStr}})

	if resp.StatusCode != 200 {
		log.Error("carAdmission:", carNo+"入场信息发送失败")
		log.Error("carAdmission PostForm:" + resp.Status)
	} else {
		log.Info("carAdmission:", carNo+"入场信息发送成功")
	}

	if err != nil {
		log.Error("carAdmission PostForm:", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("carAdmission ReadAll:", err)
	}
	return string(body)
}

/**
车辆出场
*/
func carComeOut(carNo string, inTimeStr string, outTimeStr string, noSenseTypeStr string) string {
	resp, err := http.PostForm(carComeOutUrl,
		url.Values{"carNo": {carNo}, "inTime": {inTimeStr}, "outTime": {outTimeStr}, "noSenseType": {noSenseTypeStr}})

	if resp.StatusCode != 200 {
		log.Error("carComeOut:", carNo+"出场信息发送失败")
		log.Error("carComeOut PostForm:" + resp.Status)
	} else {
		log.Info("carComeOut:", carNo+"出场信息发送成功")
	}

	if err != nil {
		log.Info("carComeOut PostForm:", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Info("carComeOut ReadAll:", err)
	}
	return string(body)
}

/**
清理脏数据
*/
func cleanDirtyData(carNo string, inTimeStr string, carStatusStr string) string {
	resp, err := http.PostForm(cleanDirtyDataUrl,
		url.Values{"carNo": {carNo}, "inTime": {inTimeStr}, "status": {carStatusStr}})

	if resp.StatusCode != 200 {
		log.Error("cleanDirtyData:", carNo+"脏数据处理信息发送失败")
		log.Error("cleanDirtyData PostForm:" + resp.Status)
	} else {
		log.Info("carComeOut:", carNo+"脏数据处理信息发送成功")
	}

	if err != nil {
		log.Info("cleanDirtyData PostForm:", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Info("cleanDirtyData ReadAll:", err)
	}
	return string(body)
}

/**
  修改中间表数据
*/
func updateRecordTemp(carStatus int64, recordTempId int64, dbConn *sql.DB) int {
	result := 0
	stmt, err := dbConn.Prepare(recordTempUpdate)
	if err != nil {
		log.Fatal("Prepare failed:", err.Error())
	}

	defer stmt.Close()

	updateRecord, err := stmt.Exec(carStatus, recordTempId)
	if err != nil {
		log.Fatal("Exec failed:", err.Error())
	}

	rowsAffect, err := updateRecord.RowsAffected()

	if err != nil {
		log.Fatal("updateRecord failed:", err.Error())
	}

	result = int(rowsAffect)
	return result
}

/**
  删除中间表数据
*/
func deleteRecordTemp(recordTempId int64, dbConn *sql.DB) int {
	result := 0
	stmt, err := dbConn.Prepare(recordTempDelete)
	if err != nil {
		log.Fatal("Prepare failed:", err.Error())
	}

	defer stmt.Close()

	deleteRecord, err := stmt.Exec(recordTempId)
	if err != nil {
		log.Fatal("Exec failed:", err.Error())
	}

	rowsAffect, err := deleteRecord.RowsAffected()

	if err != nil {
		log.Fatal("deleteRecord failed:", err.Error())
	}

	result = int(rowsAffect)
	return result
}
