package main

import (
	"flag"
	"fmt"
	"runtime"
	"strings"
	"thunder/internal/cmd"
	"time"
)

const defaultLogFileName = "./thuns.log for server, ./thunc.log for client"

func main() {
	runtime.GOMAXPROCS(1024)

	isServer := flag.Bool("s", false, "Run in server mode,default client.")
	clientDest := flag.String("c", "", "Server is specified using name, FQDN or IP address.")
	testTypePtr := flag.String("t", "", "")
	thCount := flag.Int("n", 1, "Number of Parallel Sessions (and Threads).0: Equal to number of CPUs Default: 1")
	showUI := flag.Bool("ui", false, "Show output in text UI.")
	duration := flag.Duration("d", 10*time.Second, "Duration for the test (format: <num>[ms | s | m | h]")
	bufLenStr := flag.String("l", "16KB", "Length of buffer to use (format: <num>[KB | MB | GB]) Only valid for Bandwidth tests. Max 1GB. Default: 16KB")
	portStr := flag.String("ports", "", "Use custom port numbers instead of default ones. A comma separated list of key=value pair is used.Key specifies the protocol, and value specifies the port Default: 'tcp=9999,http=9899,https=9799'")
	protocol := flag.String("p", "tcp", "Protocol ('tcp', 'udp', 'http', 'https', or 'icmp')")
	outputFile := flag.String("o", defaultLogFileName, "Name of log file. By default, following file names are used: Server mode: 'thuns.log' Client mode: 'thunc.log'")
	debug := flag.Bool("debug", false, "Enable debug information in logging output.")
	noOutput := flag.Bool("no", false, "Disable logging to file. Logging to file is enabled by default.")
	rttCount := flag.Int("i", 1000, "Number of round trip iterations for each latency measurement. Default: 1000")
	use4 := flag.Bool("4", false, "Use only IP v4 version")
	use6 := flag.Bool("6", false, "Use only IP v6 version")
	gap := flag.Duration("g", 0, "Time interval between successive measurements (format: <num>[ms | s | m | h] 0: No gap Default: 1s")
	reverse := flag.Bool("r", false, "For Bandwidth tests, send data from server to client.")
	ncs := flag.Bool("ncs", false, "No Connection Stats would be printed if this flag is specified. This is useful for running with large number of connections as specified by -n option.")
	ic := flag.Bool("ic", false, "")
	flag.Parse()

	gNoConnectionStats = *ncs

	gIgnoreCert = *ic

	mode := thunModeInv

	if *isServer {
		if *clientDest != "" {
			cmd.PrintUsageError("Invalid arguments,\"-c\" cannot be used with \"-s\".")
		}
		mode = thunModeServer
	} else if *clientDest != "" {
		mode = thunModeClient
	} else {
		cmd.PrintUsageError("Invalid arguments, use either \"-s\" or \"-c\".")
	}

	if *reverse && mode != thunModeClient {
		cmd.PrintUsageError("Invalid arguments, \"-r\" can only be used in client mode.")
	}

	if *use4 && !*use6 {
		ipVer = thunIPv4
	} else if *use6 && !*use4 {
		ipVer = thunIPv6
	}

	bufLen := unitToNumber(*bufLenStr)

	if bufLen == 0 {
		cmd.PrintUsageError(fmt.Sprintf("Invalid length specified: %s" + *bufLenStr))
	}

	if *rttCount <= 0 {
		cmd.PrintUsageError(fmt.Sprintf("Invalid RTT count for latency test: %d", *rttCount))
	}

	var testType ThunTestType

	switch *testTypePtr {
	case "":
		switch mode {
		case thunModeServer:
			testType = All
		case thunModeClient:
			testType = Bandwidth
		case thunModeExtClient:
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
		cmd.PrintUsageError(fmt.Sprintf("Invalid value \"%s\" specified for parameter \"-t\".\n"+
			"Valid parameters and values are:\n", *testTypePtr))
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
		cmd.PrintUsageError(fmt.Sprintf("Invalid value \"%s\" specified for parameter \"-p\".\n"+
			"Valid parameters and values are:\n", *protocol))
	}

	if *thCount <= 0 {
		*thCount = runtime.NumCPU()
	}

	generatePortNumbers(*portStr)

	testParam := ThunTestParam{ThunTestID{ThunProtocol(proto), testType},
		uint32(bufLen),
		uint32(*thCount),
		uint32(*rttCount),
		*reverse}
	validateTestParam(mode, testParam)

	logFileName := *outputFile
	if !*noOutput {
		if logFileName == defaultLogFileName {
			switch mode {
			case thunModeServer:
				logFileName = "thuns.log"
			case thunModeClient:
				logFileName = "thunc.log"
			case thunModeExtClient:
				logFileName = "thunxc.log"
			}
		}
		logInit(logFileName, *debug)
	}

	clientParam := thunClientParam{*duration, *gap}
	serverParam := thunServerParam{*showUI}

	switch mode {
	case thunModeServer:
		runServer(testParam, serverParam)
	case thunModeClient:
		runClient(testParam, clientParam, *clientDest)
	}
}

func emitUnsupportedTest(testParam ThunTestParam) {
	cmd.PrintUsageError(fmt.Sprintf("\"%s\" test for \"%s\" is not supported.\n",
		testToString(testParam.TestID.Type), protoToString(testParam.TestID.Protocol)))
}

func printReverseModeError() {
	cmd.PrintUsageError("Reverse mode (-r) is only supported for TCP Bandwidth tests.")
}

func validateTestParam(mode thunMode, testParam ThunTestParam) {
	testType := testParam.TestID.Type
	protocol := testParam.TestID.Protocol
	if mode == thunModeServer {
		if testType != All || protocol != TCP {
			emitUnsupportedTest(testParam)
		}
	} else if mode == thunModeClient {
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
					cmd.PrintUsageError("Maximum supported buffer size for UDP is 64K\n")
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
	} else if mode == thunModeExtClient {
		if (protocol != TCP) || (testType != ConnLatency && testType != Bandwidth) {
			emitUnsupportedTest(testParam)
		}
	}
}
