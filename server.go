package main

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"runtime"
	"sync/atomic"
	"time"
)

var gCert []byte

func runServer(testParam ThunTestParam, serverParam ethrServerParam) {
	defer stopStatsTimer()
	initServer(serverParam.showUI)
	l := runControlChannel()
	defer l.Close()
	startStatsTimer()
	for {
		conn, err := l.Accept()
		if err != nil {
			ui.printErr("Error accepting new control connection: %v", err)
			continue
		}
		go handleRequest(conn)
	}

}

func initServer(showUI bool) {
	initServerUI(showUI)
}

func runControlChannel() net.Listener {
	l, err := net.Listen("tcp", hostAddr+":"+ctrlPort)
	if err != nil {
		finiServer()
		fmt.Printf("Fatal error listening for control connections: %v", err)
		os.Exit(1)
	}
	ui.printMsg("Listening on " + ctrlPort + " for control plane")
	return l
}

func handleRequest(conn net.Conn) {
	defer conn.Close()
	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)
	ethrMsg := recvSessionMsg(dec)
	if ethrMsg.Type != EthrSyn {
		return
	}
	testParam := ethrMsg.Syn.TestParam
	server, port, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		ui.printDbg("RemoteAddr: Split host port failed: %v", err)
		return
	}
	_, _, err = net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		ui.printDbg("LocalAddr: Split host port failed: %v", err)
		return
	}
	ui.printMsg("New control connection from " + server + ", port " + port)
	test, err := newTest(server, conn, testParam, enc, dec)
	cleanupFunc := func() {
		test.ctrlConn.Close()
		close(test.done)
		deleteTest(test)
	}
	ui.emitTestHdr()
	if test.testParam.TestID.Protocol == UDP {
		if test.testParam.TestID.Type == Bandwidth {
			err = runUDPBandwidthServer(test)
		} else if test.testParam.TestID.Type == Pps {
			err = runUDPPpsServer(test)
		}
		if err != nil {
			ui.printDbg("Error encounterd in running UDP test (%s): %v",
				testToString(testParam.TestID.Type), err)
			cleanupFunc()
			return
		}
	}
	delay := timeToNextTick()
	ethrMsg = createAckMsg(gCert, delay)
	err = sendSessionMsg(enc, ethrMsg)
	if err != nil {
		ui.printErr("send session message: %v", err)
		cleanupFunc()
		return
	}
	time.Sleep(delay)
	test.isActive = true
	waitForChannelStop := make(chan bool, 1)
	serverWatchControlChannel(test, waitForChannelStop)
	<-waitForChannelStop
	test.isActive = false
	ui.printMsg("Ending " + testToString(testParam.TestID.Type) + " test from " + server)
	cleanupFunc()
	if len(gSessionKeys) > 0 {
		ui.emitTestHdr()
	}
}

func serverWatchControlChannel(test *thunTest, waitForChannelStop chan bool) {
	watchControlChannel(test, waitForChannelStop)
}

func finiServer() {
	ui.fini()
	logFini()
}

func runUDPBandwidthServer(test *thunTest) error {
	udpAddr, err := net.ResolveUDPAddr("udp", hostAddr+":"+udpBandwidthPort)
	if err != nil {
		ui.printDbg("Unable to resolve UDP address: %v", err)
		return err
	}
	l, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		ui.printDbg("Error listening on %s for UDP pkt/s tests: %v", udpPpsPort, err)
		return err
	}
	go func(l *net.UDPConn) {
		defer l.Close()
		//
		// We use NumCPU here instead of NumThreads passed from client. The
		// reason is that for UDP, there is no connection, so all packets come
		// on same CPU, so it isn't clear if there are any benefits to running
		// more threads than NumCPU(). TODO: Evaluate this in future.
		//
		for i := 0; i < runtime.NumCPU(); i++ {
			go runUDPBandwidthHandler(test, l)
		}
		<-test.done
	}(l)
	return nil
}

func runUDPBandwidthHandler(test *thunTest, conn *net.UDPConn) {
	buffer := make([]byte, test.testParam.BufferSize)
	n, remoteAddr, err := 0, new(net.UDPAddr), error(nil)
	for err == nil {
		n, remoteAddr, err = conn.ReadFromUDP(buffer)
		if err != nil {
			ui.printDbg("Error receiving data from UDP for bandwidth test: %v", err)
			continue
		}
		ethrUnused(n)
		server, port, _ := net.SplitHostPort(remoteAddr.String())
		test := getTest(server, UDP, Bandwidth)
		if test != nil {
			atomic.AddUint64(&test.testResult.data, uint64(n))
		} else {
			ui.printDbg("Received unsolicited UDP traffic on port %s from %s port %s", udpPpsPort, server, port)
		}
	}
}

func runUDPPpsServer(test *thunTest) error {
	udpAddr, err := net.ResolveUDPAddr("udp", test.session.remoteAddr+":"+udpPpsPort)
	if err != nil {
		ui.printDbg("Unable to resolve UDP address: %v", err)
		return err
	}
	l, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		ui.printDbg("Error listening on %s for UDP pkt/s tests: %v", udpPpsPort, err)
		return err
	}
	go func(l *net.UDPConn) {
		defer l.Close()

		for i := 0; i < runtime.NumCPU(); i++ {
			go runUDPPpsHandler(test, l)
		}
		<-test.done
	}(l)

	return nil
}

func runUDPPpsHandler(test *thunTest, conn *net.UDPConn) {
	buffer := make([]byte, test.testParam.BufferSize)
	_, remoteAddr, err := 0, new(net.UDPAddr), error(nil)
	for err == nil {
		_, remoteAddr, err = conn.ReadFromUDP(buffer)
		if err != nil {
			ui.printDbg("Error receiving data from UDP for pkt/s test: %v\n", err)
			continue
		}
		server, port, _ := net.SplitHostPort(remoteAddr.String())
		test := getTest(server, UDP, Pps)
		if test != nil {
			atomic.AddUint64(&test.testResult.data, 1)
		} else {
			ui.printDbg("Received unsolicited UDP traffic on port %s from %s port %s\n", udpPpsPort, server, port)
		}
	}
}
