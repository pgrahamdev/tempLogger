package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"tempLogger/types"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const defaultDbPath = "./tempLogger.db"

type TLDB struct {
	DB     *sql.DB
	Tables map[string]bool
}

func NewDB(dbPath string) (tldb TLDB, err error) {

	tldb.DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		err = fmt.Errorf("NewDB: Error opening %s: %w", dbPath, err)
		return
	}
	sqlStmt := "create table if not exists tltables (id integer not null primary key, name text);"
	_, err = tldb.DB.Exec(sqlStmt)
	if err != nil {
		err = fmt.Errorf("NewDB: Error creating tltables: %s: %w", sqlStmt, err)
		return
	}

	tldb.Tables = make(map[string]bool, 1)
	rows, err := tldb.DB.Query("select id, name from tltables")
	if err != nil {
		err = fmt.Errorf("NewDB: Error querying tltables: %w", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			err = fmt.Errorf("NewDB: Error reading query of tltables: %w", err)
			return
		}
		tldb.Tables[name] = true
	}
	err = rows.Err()
	if err != nil {
		err = fmt.Errorf("NewDB: General error reading query of tltables: %w", err)
		return
	}
	return
}

// NewTable will create a new table with the name tableName and updates the
// Tables field to
func (tldb *TLDB) NewTable(tableName string) (err error) {
	sqlStmt := fmt.Sprint("create table if not exists ", tableName, " (id integer not null primary key, humidity float, tempc float, tempf float, hiC float, hiF float)")
	_, err = tldb.DB.Exec(sqlStmt)
	if err != nil {
		err = fmt.Errorf("NewTable: Error creating table: %s: %w", tableName, err)
		return
	}

	tldb.Tables[tableName] = true
	id := len(tldb.Tables)
	sqlStmt = "insert into tltables(id, name) values(?, ?)"
	_, err = tldb.DB.Exec(sqlStmt, id, tableName)
	if err != nil {
		err = fmt.Errorf("NewTable: Error inserting %s into tltables: %w", tableName, err)
		// The insertion didn't work, so delete the entry from the map
		delete(tldb.Tables, tableName)
		return
	}

	return
}

func (tldb TLDB) InsertLog(fileName string) (err error) {
	logFile, err := os.Open(fileName)
	if err != nil {
		err = fmt.Errorf("InsertLog: Error opening tempLogger file %s: %w",
			fileName, err)
		return
	}
	logData, err := io.ReadAll(logFile)
	if err != nil {
		err = fmt.Errorf("InsertLog: Error reading tempLogger file %s: %w",
			fileName, err)
		return
	}
	logFile.Close()

	scanner := bufio.NewScanner(bytes.NewReader(logData))
	var tlDataList []types.THData
	for scanner.Scan() {
		var tlData types.THData
		line := scanner.Bytes()
		err = json.Unmarshal(line, &tlData)
		if err != nil {
			log.Println("InsertLog: Error parsing line:", string(line))
			continue
		}
		tlDataList = append(tlDataList, tlData)
	}
	if err = scanner.Err(); err != nil {
		log.Printf("InsertLog: Error parsing tempLogger file %s: %s\n",
			fileName, err.Error())
	}
	if len(tlDataList) > 0 {
		if _, ok := tldb.Tables[tlDataList[0].ID]; !ok {
			err = tldb.NewTable(tlDataList[0].ID)
			if err != nil {
				err = fmt.Errorf("InsertLog: Error creating table: %w", err)
				return
			}
		}
		for _, val := range tlDataList {
			if err = tldb.InsertRecord(val); err != nil {
				log.Println("InsertLog:", err.Error())
			}
		}
	}
	return
}

func (tldb TLDB) InsertRecord(thd types.THData) (err error) {
	ts, err := time.Parse(time.RFC3339, thd.TimeStamp)
	if err != nil {
		err = fmt.Errorf("InsertRecord: Error parsing timestamp: %s: %w",
			thd.TimeStamp, err)
		return
	}
	qStr := fmt.Sprintf("select id from %s where id=?", thd.ID)
	rows, err := tldb.DB.Query(qStr, ts.Unix())
	if err != nil {
		err = fmt.Errorf("InsertRecord: Problem with query: %w", err)
		return
	}
	defer rows.Close()

	exists := false
	// This is a primary key, so it won't be redundant
	if rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil {
			err = fmt.Errorf("InsertRecord: Problem parsing query results: %w", err)
			return
		}
		exists = true
	}
	if err = rows.Err(); err != nil {
		err = fmt.Errorf("InsertRecord: Additional query problem: %w", err)
		return
	}
	if exists {
		// Nothing to do
		return
	}
	// Put the record into the database
	stmtStr := fmt.Sprintf("insert into %s(id, humidity, tempc, tempf, hiC, hiF) values(?, ?, ?, ?, ?, ?)", thd.ID)
	_, err = tldb.DB.Exec(stmtStr, ts.Unix(), thd.Humidity, thd.TempC, thd.TempF, thd.HeatIndexC, thd.HeatIndexF)
	if err != nil {
		err = fmt.Errorf("InsertRecord: Error inserting record into table %s: %w", thd.ID, err)
		return
	}
	return
}

func (tldb TLDB) RecordCount(tableName string) (rowCnt int, err error) {
	rows, err := tldb.DB.Query("select count(id) from " + tableName)
	if err != nil {
		err = fmt.Errorf("RecordCount: Error querying %s: %w", tableName, err)
		return
	}
	defer rows.Close()
	// Should only be one result
	if rows.Next() {
		err = rows.Scan(&rowCnt)
		if err != nil {
			err = fmt.Errorf("RecordCount: Error reading count query of %s: %w", tableName, err)
			return
		}
	}
	err = rows.Err()
	if err != nil {
		err = fmt.Errorf("RecordCount: General error reading query of tltables: %w", err)
		return
	}
	return
}

func (tldb TLDB) RecordCountTime(tableName string, begin time.Time, end time.Time) (rowCnt int, err error) {
	beginU := begin.Unix()
	endU := end.Unix()
	queryStr := fmt.Sprintf("select count(id) from %s where (id > %d and id < %d)",
		tableName, beginU, endU)
	rows, err := tldb.DB.Query(queryStr)
	if err != nil {
		err = fmt.Errorf("RecordCountTime: Error querying %s: %w", tableName, err)
		return
	}
	defer rows.Close()
	// Should only be one result
	if rows.Next() {
		err = rows.Scan(&rowCnt)
		if err != nil {
			err = fmt.Errorf("RecordCountTime: Error reading count query of %s: %w", tableName, err)
			return
		}
	}
	err = rows.Err()
	if err != nil {
		err = fmt.Errorf("RecordCountTime: General error reading query of tltables: %w", err)
		return
	}
	return
}

func (tldb TLDB) RetrieveRecords(tableName string, begin time.Time, end time.Time) (tlList []types.THData, err error) {
	beginU := begin.Unix()
	endU := end.Unix()
	queryStr := fmt.Sprintf("select id, humidity, tempc, tempf, hiC, hiF from %s where (id > %d and id < %d)",
		tableName, beginU, endU)
	rows, err := tldb.DB.Query(queryStr)
	if err != nil {
		err = fmt.Errorf("RetrieveRecords: Error querying %s: %w", tableName, err)
		return
	}
	defer rows.Close()
	// Should only be one result
	for rows.Next() {
		var id int64
		var humidity, tempc, tempf, hiC, hiF float32
		err = rows.Scan(&id, &humidity, &tempc, &tempf, &hiC, &hiF)
		if err != nil {
			err = fmt.Errorf("RetrieveRecords: Error reading count query of %s: %w", tableName, err)
			return
		}
		tlList = append(tlList,
			types.THData{ID: tableName,
				TimeStamp:  time.Unix(id, 0).Format(time.RFC3339),
				Humidity:   humidity,
				TempC:      tempc,
				TempF:      tempf,
				HeatIndexC: hiC,
				HeatIndexF: hiF,
			})
	}
	err = rows.Err()
	if err != nil {
		err = fmt.Errorf("RetrieveRecords: General error reading query of tltables: %w", err)
		return
	}
	return
}

func (tldb *TLDB) Close() {
	tldb.DB.Close()
}
