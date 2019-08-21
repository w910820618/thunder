package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type logMessage struct {
	Time    string
	Type    string
	Message string
}

type logLatencyData struct {
	Time       string
	Type       string
	RemoteAddr string
	Protocol   string
	Avg        string
	Min        string
	P50        string
	P90        string
	P95        string
	P99        string
	P999       string
	P9999      string
	Max        string
}

type logTestResults struct {
	Time                 string
	Type                 string
	RemoteAddr           string
	Protocol             string
	BitsPerSecond        string
	ConnectionsPerSecond string
	PacketsPerSecond     string
	AverageLatency       string
}

var loggingActive = false
var logDebug = false
var logChan = make(chan string, 64)

func logInit(fileName string, debug bool) {
	if fileName == "" {
		return
	}
	logDebug = debug
	logFile, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Printf("Unable to open the log file %s, Error: %v", fileName, err)
		return
	}
	log.SetFlags(0)
	log.SetOutput(logFile)
	loggingActive = true
	go runLogger(logFile)
}

func logFini() {
	loggingActive = false
}

func runLogger(logFile *os.File) {
	for loggingActive {
		s := <-logChan
		log.Println(s)
	}
	logFile.Close()
}

func _log(prefix, msg string) {
	if loggingActive {
		logData := logMessage{}
		logData.Time = time.Now().UTC().Format(time.RFC3339)
		logData.Type = prefix
		logData.Message = msg
		logJSON, _ := json.Marshal(logData)
		logChan <- string(logJSON)
	}
}

func logMsg(msg string) {
	_log("INFO", msg)
}

func logErr(msg string) {
	_log("ERROR", msg)
}

func logDbg(msg string) {
	if logDebug {
		_log("DEBUG", msg)
	}
}

func logResults(s []string) {
	if loggingActive {
		logData := logTestResults{}
		logData.Time = time.Now().UTC().Format(time.RFC3339)
		logData.Type = "TestResult"
		logData.RemoteAddr = s[0]
		logData.Protocol = s[1]
		logData.BitsPerSecond = s[2]
		logData.ConnectionsPerSecond = s[3]
		logData.PacketsPerSecond = s[4]
		logData.AverageLatency = s[5]
		logJSON, _ := json.Marshal(logData)
		logChan <- string(logJSON)
	}
}

func logLatency(remoteAddr, proto string, avg, min, p50, p90, p95, p99, p999, p9999, max time.Duration) {
	if loggingActive {
		logData := logLatencyData{}
		logData.Time = time.Now().UTC().Format(time.RFC3339)
		logData.Type = "LatencyResult"
		logData.RemoteAddr = remoteAddr
		logData.Protocol = proto
		logData.Avg = durationToString(avg)
		logData.Min = durationToString(min)
		logData.P50 = durationToString(p50)
		logData.P90 = durationToString(p90)
		logData.P95 = durationToString(p95)
		logData.P99 = durationToString(p99)
		logData.P999 = durationToString(p999)
		logData.P9999 = durationToString(p9999)
		logData.Max = durationToString(max)
		logJSON, _ := json.Marshal(logData)
		logChan <- string(logJSON)
	}
}
