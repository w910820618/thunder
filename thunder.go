package main

import (
	"flag"
	"runtime"
	"time"
)

func main() {
	/**
	Get data from the command line
	*/
	isServer := flag.Bool("s", false, "")
	clientDest := flag.String("c", "", "")
	thCount := flag.Int("n", 1, "")
	bufLenStr := flag.String("l", "16KB", "")
	rttCount := flag.Int("i", 1000, "")
	reverse := flag.Bool("r", false, "")
	gap := flag.Duration("g", 0, "")
	duration := flag.Duration("d", 10*time.Second, "")
	portStr := flag.String("ports", "", "")

	flag.Parse()

	xMode := false

	mode := ethrModeInv

	if *isServer {
		if *clientDest != "" {
			/**
			cmd.PrintUsageError("Invalid arguments, \"-c\" cannot be used with \"-s\".")
			*/
		}
		if xMode {
			mode = ethrModeExtServer
		} else {
			mode = ethrModeServer
		}
	} else if *clientDest != "" {
		if xMode {
			mode = ethrModeExtClient
		} else {
			mode = ethrModeClient
		}
	} else {
		/**
		cmd.PrintUsageError("Invalid arguments, use either \"-s\" or \"-c\".")
		*/
	}

	bufLen := unitToNumber(*bufLenStr)
	if bufLen == 0 {
		/**
		cmd.PrintUsageError(fmt.Sprintf("Invalid length specified: %s" + *bufLenStr))
		*/
	}

	if *rttCount <= 0 {
		/**
		cmd.PrintUsageError(fmt.Sprintf("Invalid RTT count for latency test: %d", *rttCount))
		*/
	}

	/**
	Judge the type of test pps
	*/
	var testType EthrTestType
	testType = Pps

	/**
	Judge the protocol of test UDP
	*/
	proto := UDP

	if *thCount <= 0 {
		*thCount = runtime.NumCPU()
	}

	//
	// For Pkt/s, we always override the buffer size to be just 1 byte.
	// TODO: Evaluate in future, if we need to support > 1 byte packets for
	//       Pkt/s testing.
	//
	if testType == Pps {
		bufLen = 1
	}

	testParam := EthrTestParam{EthrTestID{EthrProtocol(proto), testType},
		uint32(*thCount),
		uint32(bufLen),
		uint32(*rttCount),
		*reverse}
	validateTestParam(mode, testParam)

	generatePortNumbers(*portStr)
	clientParam := ethrClientParam{*duration, *gap}

	switch mode {
	case ethrModeServer:
		runServer(testParam)
	case ethrModeClient:
		runClient(testParam, clientParam, *clientDest)
	}
}

func emitUnsupportedTest(testParam EthrTestParam) {
	/**
	cmd.PrintUsageError(fmt.Sprintf("\"%s\" test for \"%s\" is not supported.\n",
		testToString(testParam.TestID.Type), protoToString(testParam.TestID.Protocol)))
	*/
}

func printReverseModeError() {
	/**
	cmd.PrintUsageError("Reverse mode (-r) is only supported for TCP Bandwidth tests.")
	*/
}

func validateTestParam(mode ethrMode, testParam EthrTestParam) {
	testType := testParam.TestID.Type
	protocol := testParam.TestID.Protocol
	if mode == ethrModeServer {
		if testType != All || protocol != TCP {
			emitUnsupportedTest(testParam)
		}
	} else if mode == ethrModeClient {
		if testType != Bandwidth && testType != Pps {
			emitUnsupportedTest(testParam)
		}
		if testType == Bandwidth {
			if testParam.BufferSize > (64 * 1024) {
				/**
				cmd.PrintUsageError("Maximum supported buffer size for UDP is 64K\n")
				*/
			}
		}
		if testParam.Reverse {
			printReverseModeError()
		}
	} else if mode == ethrModeExtClient {
		if (protocol != TCP) || (testType != ConnLatency && testType != Bandwidth) {
			emitUnsupportedTest(testParam)
		}
	}
}
