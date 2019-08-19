package main

import (
	"flag"
)

func main() {
	/**
	Get data from the command line   add ip port
	*/
	isServer := flag.Bool("s", false, "")
	thCount := flag.Int("n", 1, "")
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

	thunderParam := ThunderParam{*hostStr, *portStr}
	serverParam := EthrTestParam{thunderParam, uint32(bufLen), uint32(*thCount)}
	clientParam := EthrTestParam{thunderParam, uint32(bufLen), uint32(*thCount)}
	switch mode {
	case ethrModeServer:
		runServer(serverParam)
	case ethrModeClient:
		runClient(clientParam)
	}
}
