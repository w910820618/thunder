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
	thCount := flag.Int("n", 1, "")
	duration := flag.Duration("d", 10*time.Second, "")
	bufLenStr := flag.String("len", "", "")
	hostStr := flag.String("hosts", "", "")
	portStr := flag.String("ports", "", "")
	flag.Parse()

	mode := ethrModeInv

	if *isServer {

		mode = ethrModeServer

	} else {

		mode = ethrModeClient

	}

	bufLen := unitToNumber(*bufLenStr)

	clientParam := ethrClientParam{*duration}
	testParam := EthrTestParam{*hostStr, *portStr, uint32(bufLen), uint32(*thCount)}
	switch mode {
	case ethrModeServer:
		runServer(testParam)
	case ethrModeClient:
		runClient(testParam, clientParam)
	}
}
