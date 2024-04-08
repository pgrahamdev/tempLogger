package main

import (
	"os"
	"testing"
	"time"
)

const testDbPath = "test/test.db"

func TestCreateDB(t *testing.T) {
	_, err := os.Stat(testDbPath)
	if err == nil {
		err = os.Remove(testDbPath)
		if err != nil {
			t.Fatalf("Could not remove file %s: %s", testDbPath, err.Error())
		}
	}
	tldb, err := NewDB(testDbPath)
	if err != nil {
		t.Fatalf("Could not create %s: %s", testDbPath, err.Error())
	}
	defer tldb.Close()
	if len(tldb.Tables) != 0 {
		t.Error("Tables struct member non-zero.")
	}
}

func TestInsertLog(t *testing.T) {
	_, err := os.Stat(testDbPath)
	if err == nil {
		err = os.Remove(testDbPath)
		if err != nil {
			t.Fatalf("Could not remove file %s: %s", testDbPath, err.Error())
		}
	}
	tldb, err := NewDB(testDbPath)
	if err != nil {
		t.Fatalf("Could not create %s: %s", testDbPath, err.Error())
	}
	defer tldb.Close()
	if len(tldb.Tables) != 0 {
		t.Error("Tables struct member non-zero.")
	}
	err = tldb.InsertLog("test/tempLogger-20240114-1.log")
	if err != nil {
		t.Fatalf("Error inserting log into database: %s", err.Error())
	}
	if len(tldb.Tables) != 1 {
		t.Error("Tables struct member not equal to 1.")
	}
	rowCnt, err := tldb.RecordCount("sensor1")
	if err != nil {
		t.Errorf("Could not read row count for %s", "sensor1")
	}
	if rowCnt != 702 {
		t.Errorf("Incorrect number of records: Expected %d, Actual %d", 702, rowCnt)
	}
	begin := time.Unix(1705298580, 0)
	end := time.Unix(1705300900, 0)
	rowCnt, err = tldb.RecordCountTime("sensor1", begin, end)
	if err != nil {
		t.Errorf("Could not read row count for %s", "sensor1")
	}
	if rowCnt != 19 {
		t.Errorf("Incorrect number of time range records: Expected %d, Actual %d", 19, rowCnt)
	}
	tlList, err := tldb.RetrieveRecords("sensor1", begin, end)
	if err != nil {
		t.Errorf("Could not read records from %s", "sensor1")
	}
	if len(tlList) != 19 {
		t.Errorf("Incorrect number of time range records: Expected %d, Actual %d", 19, len(tlList))
	}
}

func TestInsertLog2(t *testing.T) {
	_, err := os.Stat(testDbPath)
	if err == nil {
		err = os.Remove(testDbPath)
		if err != nil {
			t.Fatalf("Could not remove file %s: %s", testDbPath, err.Error())
		}
	}
	tldb, err := NewDB(testDbPath)
	if err != nil {
		t.Fatalf("Could not create %s: %s", testDbPath, err.Error())
	}
	defer tldb.Close()
	if len(tldb.Tables) != 0 {
		t.Error("Tables struct member non-zero.")
	}
	err = tldb.InsertLog("test/tempLogger-20240114-1.log")
	if err != nil {
		t.Fatalf("Error inserting log into database: %s", err.Error())
	}
	if len(tldb.Tables) != 1 {
		t.Error("Tables struct member not equal to 1.")
	}
	rowCnt, err := tldb.RecordCount("sensor1")
	if err != nil {
		t.Errorf("Could not read row count for %s", "sensor1")
	}
	if rowCnt != 702 {
		t.Errorf("Incorrect number of records: Expected %d, Actual %d", 702, rowCnt)
	}
	err = tldb.InsertLog("test/tempLogger-20240114-2.log")
	if err != nil {
		t.Fatalf("Error inserting log into database: %s", err.Error())
	}
	if len(tldb.Tables) != 1 {
		t.Error("Tables struct member not equal to 1.")
	}
	rowCnt, err = tldb.RecordCount("sensor1")
	if err != nil {
		t.Errorf("Could not read row count for %s", "sensor1")
	}
	if rowCnt != 703 {
		t.Errorf("Incorrect number of records: Expected %d, Actual %d", 702, rowCnt)
	}
}
