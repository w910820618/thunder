package main

import (
	"flag"
)

func main() {
	/**
	Get data from the command line   add ip port
	*/
	isServer := flag.Bool("s", false, "")
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
	//fmt.Printf("%v", bufLen)
	//if bufLen == 0 {
	//	fmt.Printf("Invalid length specified: %s" + *bufLenStr)
	//	/**
	//	cmd.PrintUsageError(fmt.Sprintf("Invalid length specified: %s" + *bufLenStr))
	//	*/
	//}
	//testParam := EthrTestParam{EthrTestID{*hostStr, *portStr}, uint32(bufLen)}

	clientParam := thunderClientParam{*hostStr, *portStr, uint32(bufLen)}
	serverParam := thunderServerParam{*hostStr, *portStr}
	switch mode {
	case ethrModeServer:
		runServer(serverParam)
	case ethrModeClient:
		runClient(clientParam)
	}
}
