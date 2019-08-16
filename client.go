package main

import (
	"time"
)

func runClient(testParam EthrTestParam, clientParam ethrClientParam, server string) {
	server = "[" + server + "]"
	runTest(testParam, clientParam.duration)

}

func runTest(test *ethrTest, d time.Duration) {
	if test.testParam.TestID.Protocol == UDP {
		if test.testParam.TestID.Type == Pps {
			go runUDPPpsTest(test)
		}
	}
}

func runUDPPpsTest(test *ethrTest) {

}
