package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"tempLogger/types"
	"time"

	"github.com/tarm/serial"
)

const swVer = 3

func main() {
	fmt.Println("tempLogger Version", swVer)
	cfg := flag.String("config", "tempLogger.json", "Path to the JSON configuration file")
	flag.Parse()

	var s *serial.Port
	var err error
	var connectRetries int

	cfgFile, err := os.Open(*cfg)
	if err != nil {
		log.Fatalln("tempLogger: Could not open file", *cfg, ":", err.Error())
	}

	cfBytes, err := io.ReadAll(cfgFile)
	if err != nil {
		log.Fatalln("tempLogger: Could not read file", *cfg, ":", err.Error())
	}

	var tlCfg types.TLCfg
	err = json.Unmarshal(cfBytes, &tlCfg)
	if err != nil {
		log.Fatalln("tempLogger: Could not parse file", *cfg, ":", err.Error())
	}

	c := &serial.Config{Name: tlCfg.SerialPath, Baud: tlCfg.Baud}
	for {
		s, err = serial.OpenPort(c)
		if err != nil {
			log.Println("tempLogger: Error opening serial port", tlCfg.SerialPath, ":", err.Error())
			if connectRetries >= tlCfg.Retries {
				log.Fatalln("tempLogger: Retried opening serial port", tlCfg.Retries, "times. Exiting...")
			}
			connectRetries++
			// Sleep for 10 seconds to allow time for the port to show up
			time.Sleep(10 * time.Second)
			continue
		}
		// We have a port, so quit loop
		break
	}
	defer s.Close()

	tmpReader := bufio.NewReader(s)

	var tmpData types.THData
	for {
		var bytes []byte
		// Read all of the values but only use the last one
		for j := 0; j < 60; j++ {
			bytes, _ = tmpReader.ReadBytes('\n')
		}
		// n, err := s.Write([]byte("test"))
		// if err != nil {
		// 	log.Fatal(err)
		// }

		err = json.Unmarshal(bytes, &tmpData)
		if err != nil {
			log.Println("tempLogger: Error reading data:", err.Error())
			continue
		}
		tmpTimeStamp := time.Now()
		tmpData.TimeStamp = tmpTimeStamp.Format(time.RFC3339)
		tmpData.ID = tlCfg.ID
		//log.Println(string(bytes))
		bytes, err = json.Marshal(tmpData)
		if err != nil {
			log.Println("tempLogger: Error marshalling data", err.Error())
			continue
		}
		fileName := fmt.Sprintf("tempLogger-%04d%02d%02d.log",
			tmpTimeStamp.Year(),
			tmpTimeStamp.Month(),
			tmpTimeStamp.Day(),
		)
		f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Println("tempLogger: Error opening file", fileName, ":", err.Error())
			continue
		}
		if _, err = f.WriteString(string(bytes) + "\n"); err != nil {
			f.Close() // ignore error; Write error takes precedence
			log.Println("tempLogger: Error writing string to file", fileName, ":", err.Error())
		} else {
			if err = f.Close(); err != nil {
				log.Println("tempLogger: Error closing file", fileName, ":", err.Error())
			}
		}
	}
}
