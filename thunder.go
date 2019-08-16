package main

import (
	"flag"
	"runtime"
	"strings"
	"time"
)

func main() {
	isServer := flag.Bool("s", false, "")
	clientDest := flag.String("c", "", "")
	testTypePtr := flag.String("t", "", "")
	thCount := flag.Int("n", 1, "")
	bufLenStr := flag.String("l", "16KB", "")
	protocol := flag.String("p", "tcp", "")
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

	var testType EthrTestType
	switch *testTypePtr {
	case "":
		switch mode {
		case ethrModeServer:
			testType = All
		case ethrModeExtServer:
			testType = All
		case ethrModeClient:
			testType = Bandwidth
		case ethrModeExtClient:
			testType = ConnLatency
		}
	case "b":
		testType = Bandwidth
	case "c":
		testType = Cps
	case "p":
		testType = Pps
	case "l":
		testType = Latency
	case "cl":
		testType = ConnLatency
	default:
		/**
		cmd.PrintUsageError(fmt.Sprintf("Invalid value \"%s\" specified for parameter \"-t\".\n"+
			"Valid parameters and values are:\n", *testTypePtr))
		*/
	}

	p := strings.ToUpper(*protocol)
	proto := TCP
	switch p {
	case "TCP":
		proto = TCP
	case "UDP":
		proto = UDP
	case "HTTP":
		proto = HTTP
	case "HTTPS":
		proto = HTTPS
	case "ICMP":
		proto = ICMP
	default:
		/**
		cmd.PrintUsageError(fmt.Sprintf("Invalid value \"%s\" specified for parameter \"-p\".\n"+
			"Valid parameters and values are:\n", *protocol))
		*/
	}

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
		switch protocol {
		case TCP:
			if testType != Bandwidth && testType != Cps && testType != Latency {
				emitUnsupportedTest(testParam)
			}
			if testParam.Reverse && testType != Bandwidth {
				printReverseModeError()
			}
		case UDP:
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
		case HTTP:
			if testType != Bandwidth && testType != Latency {
				emitUnsupportedTest(testParam)
			}
			if testParam.Reverse {
				printReverseModeError()
			}
		case HTTPS:
			if testType != Bandwidth {
				emitUnsupportedTest(testParam)
			}
			if testParam.Reverse {
				printReverseModeError()
			}
		default:
			emitUnsupportedTest(testParam)
		}
	} else if mode == ethrModeExtClient {
		if (protocol != TCP) || (testType != ConnLatency && testType != Bandwidth) {
			emitUnsupportedTest(testParam)
		}
	}
}
