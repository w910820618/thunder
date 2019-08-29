package main

import (
	"flag"
	"runtime"
	"time"
)

func main() {
	/**
	Get data from the command line   add ip port
	*/
	isServer := flag.Bool("s", false, "")
	hostAddrStr := flag.String("h", "127.0.0.1", "")
	clientDest := flag.String("c", "", "")
	//testTypePtr := flag.String("t", "", "")
	thCount := flag.Int("n", 1, "")
	showUI := flag.Bool("ui", false, "")
	duration := flag.Duration("d", 10*time.Second, "")
	bufLenStr := flag.String("len", "512B", "")
	flag.Parse()

	mode := ethrModeInv

	if *isServer {
		mode = ethrModeServer
	} else {
		mode = ethrModeClient
	}

	bufLen := unitToNumber(*bufLenStr)

	var testType ThunTestType

	switch mode {
	case ethrModeServer:
		testType = All
	case ethrModeClient:
		testType = Bandwidth
	}

	//switch *testTypePtr {
	//case "":
	//	switch mode {
	//	case ethrModeServer:
	//		testType = All
	//	case ethrModeClient:
	//		testType = Bandwidth
	//	}
	//case "b":
	//	testType = Bandwidth
	//case "c":
	//	testType = Cps
	//case "p":
	//	testType = Pps
	//case "l":
	//	testType = Latency
	//case "cl":
	//	testType = ConnLatency
	//default:
	//	cmd.PrintUsageError(fmt.Sprintf("Invalid value \"%s\" specified for parameter \"-t\".\n"+
	//		"Valid parameters and values are:\n", *testTypePtr))
	//}

	if *thCount <= 0 {
		*thCount = runtime.NumCPU()
	}

	runtime.GOMAXPROCS(*thCount)
	wg.Add(*thCount)

	generateAddr(*hostAddrStr)
	testParam := ThunTestParam{ThunTestID{ThunProtocol(UDP), testType},
		uint32(bufLen),
		uint32(*thCount)}

	clientParam := thunClientParam{*duration}
	serverParam := ethrServerParam{*showUI}

	//clientParam := ethrClientParam{*duration}
	//testParam := ThunTestParam{*hostStr, *portStr, uint32(bufLen), uint32(*thCount)}
	switch mode {
	case ethrModeServer:
		runServer(testParam, serverParam)
	case ethrModeClient:
		runClient(testParam, clientParam, *clientDest)
	}
}
