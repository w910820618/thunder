package main

import (
	"flag"
	"runtime"
	"time"
)

func main() {
	isServer := flag.Bool("s", false, "")
	hostAddrStr := flag.String("h", "127.0.0.1", "")
	thCount := flag.Int("n", 1, "")
	showUI := flag.Bool("ui", false, "")
	duration := flag.Duration("d", 10*time.Second, "")
	bufLenStr := flag.String("len", "64B", "")
	flag.Parse()

	mode := thunModeInv

	if *isServer {
		mode = thunModeServer
	} else {
		mode = thunModeClient
	}

	bufLen := unitToNumber(*bufLenStr)

	var testType ThunTestType

	switch mode {
	case thunModeServer:
		testType = All
	case thunModeClient:
		testType = Bandwidth
	}

	if *thCount <= 0 {
		*thCount = runtime.NumCPU()

	}

	generateAddr(*hostAddrStr)
	testParam := ThunTestParam{ThunTestID{ThunProtocol(UDP), testType},
		uint32(bufLen),
		uint32(*thCount)}

	clientParam := thunClientParam{*duration}
	serverParam := thunServerParam{*showUI}

	switch mode {
	case thunModeServer:
		runServer(testParam, serverParam)
	case thunModeClient:
		runClient(testParam, clientParam, *hostAddrStr)
	}
}
