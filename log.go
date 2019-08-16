package main

import (
	"github.com/op/go-logging"
	"os"
)

type logMessage struct {
	Time    string
	Type    string
	Message string
}

var log = logging.MustGetLogger("thunder")

var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

func logInit() *logMessage {
	backend := logging.NewLogBackend(os.Stderr, "", 0)

	backendFormatter := logging.NewBackendFormatter(backend, format)

	backendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(logging.INFO, "")

	logging.SetBackend(backendLeveled, backendFormatter)
	return nil
}

func (message *logMessage) logInfo() {
}

func (message *logMessage) logErr() {
}

func (message *logMessage) logWarn() {
}
