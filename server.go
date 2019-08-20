package main

import (
	"fmt"
	"net"
	"runtime"
	"sync/atomic"
)

func runServer(testParam ThunTestParam) {
	defer stopStatsTimer()
	startStatsTimer()
	test, err := newTest(hostAddr, testParam)
	if err != nil {
		return
	}
	runUDPPpsServer(test)
}

func runUDPPpsServer(test *thunTest) error {
	udpAddr, err := net.ResolveUDPAddr("udp", test.session.remoteAddr+":"+udpPpsPort)
	if err != nil {
		fmt.Println("Unable to resolve UDP address: %v", err)
		return err
	}
	l, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Printf("Error listening on %s for UDP pkt/s tests: %v", udpPpsPort, err)
		return err
	}

	defer l.Close()
	for {
		for i := 0; i < runtime.NumCPU(); i++ {
			go runUDPPpsHandler(test, l)
		}
		<-test.done
	}

	return nil
}

func runUDPPpsHandler(test *thunTest, conn *net.UDPConn) {
	buffer := make([]byte, test.testParam.BufferSize)
	_, remoteAddr, err := 0, new(net.UDPAddr), error(nil)
	for err == nil {
		_, remoteAddr, err = conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Printf("Error receiving data from UDP for pkt/s test: %v\n", err)
			continue
		}
		server, port, _ := net.SplitHostPort(remoteAddr.String())
		test := getTest(hostAddr, UDP, Pps)
		if test != nil {
			atomic.AddUint64(&test.testResult.data, 1)
		} else {
			fmt.Printf("Received unsolicited UDP traffic on port %s from %s port %s\n", udpPpsPort, server, port)
		}
	}
}
