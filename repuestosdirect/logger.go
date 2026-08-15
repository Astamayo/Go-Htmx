package main

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type logEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"msg"`
	Detail  string `json:"detail,omitempty"`
}

func logInfo(msg string, detail ...string) {
	writeLog("info", msg, detail...)
}

func logError(msg string, detail ...string) {
	writeLog("error", msg, detail...)
}

func writeLog(level, msg string, detail ...string) {
	d := ""
	if len(detail) > 0 {
		d = detail[0]
	}
	if os.Getenv("LOG_JSON") == "1" || isProduction() {
		b, _ := json.Marshal(logEntry{
			Time:    time.Now().UTC().Format(time.RFC3339),
			Level:   level,
			Message: msg,
			Detail:  d,
		})
		log.Println(string(b))
		return
	}
	if d != "" {
		log.Printf("[%s] %s: %s", level, msg, d)
		return
	}
	log.Printf("[%s] %s", level, msg)
}
