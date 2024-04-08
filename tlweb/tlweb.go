package main

import (
	_ "embed"
	"encoding/csv"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"tempLogger/types"
	"time"
)

type TLWeb struct {
	Tldb TLDB
}

//go:embed page.html
var page string

func NewSummary(tlData []types.THData) (summary types.TLSummary) {
	tlDataCount := len(tlData)
	var maxIndex, minIndex int
	var maxValue, minValue float32
	maxValue = -1000.0
	minValue = 1000.0
	if tlDataCount > 0 {
		for i := 0; i < tlDataCount; i++ {
			var tmpTemp types.TempRecord
			err := tmpTemp.FromTHData(tlData[i])
			if err != nil {
				log.Println("NewTLPage: Error converting temp:", err.Error())
			} else {
				summary.TLData = append(summary.TLData, tmpTemp)
			}
			if tlData[i].TempF > maxValue {
				maxValue = tlData[i].TempF
				maxIndex = i
			}
			if tlData[i].TempF < minValue {
				minValue = tlData[i].TempF
				minIndex = i
			}
		}
		summary.MinTemp.FromTHData(tlData[minIndex])
		summary.MaxTemp.FromTHData(tlData[maxIndex])
		summary.LastTemp.FromTHData(tlData[tlDataCount-1])
	}
	return
}

func NewTLPage(tldata []types.THData, tl2data []types.THData, outsideData []types.THData, page string) (tlPage types.TLPage) {
	tlPage.Page = page
	tlPage.Summaries = append(tlPage.Summaries, NewSummary(outsideData))
	tlPage.Summaries = append(tlPage.Summaries, NewSummary(tldata))
	tlPage.Summaries = append(tlPage.Summaries, NewSummary(tl2data))
	return
}

func (tlweb TLWeb) ShowDB(w http.ResponseWriter, req *http.Request) {
	end := time.Now()
	begin := end.Add(-time.Hour * 24 * 2)
	tlData, err := tlweb.Tldb.RetrieveRecords("sensor1", begin, end)
	if err != nil {
		log.Println("ShowDB: Error retrieving sensor1 data:", err.Error())
	}
	tl2Data, err := tlweb.Tldb.RetrieveRecords("sensor2", begin, end)
	if err != nil {
		log.Println("ShowDB: Error retrieving sensor2 data:", err.Error())
	}
	outsideData, err := tlweb.Tldb.RetrieveRecords("outside", begin, end)
	if err != nil {
		log.Println("ShowDB: Error retrieving outside data:", err.Error())
	}
	log.Println("Daily", len(tlData), len(tl2Data), len(outsideData))
	tlPage := NewTLPage(tlData, tl2Data, outsideData, "Daily")
	//simple := "Hello, World!"
	t, err := template.New("simple").Parse(page)
	if err != nil {
		log.Println("ShowDB: Error parsing template:", err.Error())
	}
	err = t.Execute(w, tlPage)
	if err != nil {
		log.Println("ShowDB: Error executing template:", err.Error())
	}
}

func (tlweb TLWeb) ShowDBWeek(w http.ResponseWriter, req *http.Request) {
	end := time.Now()
	begin := end.Add(-time.Hour * 24 * 7)
	tlData, err := tlweb.Tldb.RetrieveRecords("sensor1", begin, end)
	if err != nil {
		log.Println("ShowDBWeek: Error retrieving sensor1 data:", err.Error())
	}
	tl2Data, err := tlweb.Tldb.RetrieveRecords("sensor2", begin, end)
	if err != nil {
		log.Println("ShowDBWeek: Error retrieving sensor2 data:", err.Error())
	}
	outsideData, err := tlweb.Tldb.RetrieveRecords("outside", begin, end)
	if err != nil {
		log.Println("ShowDBWeek: Error retrieving outside data:", err.Error())
	}
	log.Println("Weekly", len(tlData), len(tl2Data), len(outsideData))
	tlPage := NewTLPage(tlData, tl2Data, outsideData, "Weekly")
	//simple := "Hello, World!"
	t, err := template.New("simple").Parse(page)
	if err != nil {
		log.Println("ShowDBWeek: Error parsing template:", err.Error())
	}
	err = t.Execute(w, tlPage)
	if err != nil {
		log.Println("ShowDBWeek: Error executing template:", err.Error())
	}
}

func watchFiles(watchPath string, changedFile chan string, done chan bool) {
	cmd := exec.Command("/usr/bin/inotifywait", "-qmrc", "-e", "moved_to", watchPath)

	sout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("watchFiles: Error establishing stdout pipe:", err.Error())
		done <- true
		close(changedFile)
		return
	}

	serr, err := cmd.StderrPipe()
	if err != nil {
		log.Println("watchFiles: Error establishing stderr pipe:", err.Error())
		done <- true
		close(changedFile)
		return
	}

	err = cmd.Start()
	if err != nil {
		log.Println("watchFiles: Error starting inotifywait:", err.Error())
		done <- true
		close(changedFile)
		return
	}

	cReader := csv.NewReader(sout)
	for err == nil {
		fields, err := cReader.Read()
		if err != nil {
			continue
		}
		if len(fields) != 3 {
			log.Println("watchFiles: Unexpected record:", fields)
			continue
		}
		changedFile <- fields[0] + fields[2]
	}
	if err == io.EOF {
		log.Println("watchFiles: inotifywait completed with an unknown error.")
	} else {
		log.Println("watchFiles: inotifywait error:", err.Error())
	}

	serrData, err := io.ReadAll(serr)
	if err != nil {
		log.Println("watchFiles: Error reading stderr:", err.Error())
	}
	if err := cmd.Wait(); err != nil {
		log.Println("watchFiles: Error with inotifywait execution:", err.Error())
		log.Println("StdErr:", string(serrData))
	}

	done <- true
	close(changedFile)
}

func updateDatabase(changedFile chan string, done chan bool, tldb TLDB) {
	for {
		select {
		case file := <-changedFile:
			log.Println("File", file, "has been modified.")
			err := tldb.InsertLog(file)
			if err != nil {
				log.Printf("updateDatabase: Error loading log: %s: %s\n", file, err.Error())
			}
			for key := range tldb.Tables {
				rowCnt, err := tldb.RecordCount(key)
				if err != nil {
					log.Printf("updateDatabase: %s\n", err.Error())
				} else {
					log.Println(key, "has", rowCnt, "records.")
				}
			}
		case <-done:
			return
		}
	}
}

const swVer = "4"

func main() {
	fmt.Println("tlweb, Version", swVer)
	addr := flag.String("addr", ":8080", "http service address")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		log.Fatalln("Must have at least one datapath as an argument")
		os.Exit(1)
	}
	done := make(chan bool, 1)
	changedFile := make(chan string, 10)

	tldb, err := NewDB(defaultDbPath)
	if err != nil {
		log.Fatalln("Error opening database:", err.Error())
	}
	// Watch files from multiple paths
	for _, val := range args {
		go watchFiles(val, changedFile, done)
	}
	go updateDatabase(changedFile, done, tldb)

	tlWeb := TLWeb{Tldb: tldb}
	// http.HandleFunc("/ws", ctx.WsHandler)
	http.HandleFunc("/", tlWeb.ShowDB)
	http.HandleFunc("/weekly", tlWeb.ShowDBWeek)

	log.Fatal(http.ListenAndServe(*addr, nil))

}
