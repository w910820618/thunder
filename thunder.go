package main

import (
	"flag"
	"time"
)

func main() {
	/**
	Get data from the command line   add ip port
	*/
	isServer := flag.Bool("s", false, "")
	clientDest := flag.String("c", "", "")
	thCount := flag.Int("n", 1, "")
	duration := flag.Duration("d", 10*time.Second, "")
	bufLenStr := flag.String("len", "", "")
	portStr := flag.String("ports", "", "")
	flag.Parse()

	mode := ethrModeInv

	if *isServer {

		mode = ethrModeServer

	} else {

		mode = ethrModeClient

	}
	bufLen := unitToNumber(*bufLenStr)
	generatePortNumbers(*portStr)
	testParam := ThunTestParam{ThunTestID{ThunProtocol(UDP), Pps},
		uint32(bufLen),
		uint32(*thCount)}

	clientParam := thunClientParam{*duration}

	//clientParam := ethrClientParam{*duration}
	//testParam := ThunTestParam{*hostStr, *portStr, uint32(bufLen), uint32(*thCount)}
	switch mode {
	case ethrModeServer:
		runServer(testParam)
	case ethrModeClient:
		runClient(testParam, clientParam, *clientDest)
	}
}
