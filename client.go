package main

import (
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync/atomic"
	"time"
)

const (
	timeout    = 0
	interrupt  = 1
	serverDone = 2
)

func runDurationTimer(d time.Duration, toStop chan int) {
	go func() {
		dSeconds := uint64(d.Seconds())
		if dSeconds == 0 {
			return
		}
		time.Sleep(d)
		toStop <- timeout
	}()
}

func runClient(testParam ThunTestParam, clientParam thunClientParam, server string) {
	initClient()
	test, err := establishSession(testParam, server)
	if err != nil {
		return
	}
	runTest(test, clientParam.duration)
}

func initClient() {
	initClientUI()
}

func handleCtrlC(toStop chan int) {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, os.Kill)
	go func() {
		sig := <-sigChan
		switch sig {
		case os.Interrupt:
			fallthrough
		case os.Kill:
			toStop <- interrupt
		}
	}()
}

func clientWatchControlChannel(test *thunTest, toStop chan int) {
	go func() {
		waitForChannelStop := make(chan bool, 1)
		watchControlChannel(test, waitForChannelStop)
		<-waitForChannelStop
		toStop <- serverDone
	}()
}

func establishSession(testParam ThunTestParam, server string) (test *thunTest, err error) {
	conn, err := net.Dial("tcp", server+":"+ctrlPort)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()
	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)
	thunMsg := createSynMsg(testParam)
	err = sendSessionMsg(enc, thunMsg)
	if err != nil {
		return
	}
	rserver, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	server = "[" + rserver + "]"
	test, err = newTest(server, conn, testParam, enc, dec)
	if err != nil {
		thunMsg = createFinMsg(err.Error())
		sendSessionMsg(enc, thunMsg)
		return
	}
	thunMsg = recvSessionMsg(test.dec)
	if thunMsg.Type != ThunAck {
		if thunMsg.Type == ThunFin {
			err = fmt.Errorf("%s", thunMsg.Fin.Message)
		} else {
			err = fmt.Errorf("Unexpected control message received. %v", thunMsg)
		}
		deleteTest(test)
		return nil, err
	}
	gCert = thunMsg.Ack.Cert
	napDuration := thunMsg.Ack.NapDuration
	time.Sleep(napDuration)
	// TODO: Enable this in future, right now there is not much value coming
	// from this.
	/**
		thunMsg = createAckMsg()
		err = sendSessionMsg(test.enc, thunMsg)
		if err != nil {
			os.Exit(1)
		}
	    **/
	return
}

func runTest(test *thunTest, d time.Duration) {
	startStatsTimer()
	go runUDPTest(test)
	test.isActive = true
	toStop := make(chan int, 1)
	runDurationTimer(d, toStop)
	clientWatchControlChannel(test, toStop)
	handleCtrlC(toStop)
	reason := <-toStop
	close(test.done)
	sendSessionMsg(test.enc, &ThunMsg{})
	test.ctrlConn.Close()
	stopStatsTimer()
	switch reason {
	case timeout:
		fmt.Printf("Thun done, duration: " + d.String() + ".")
	case interrupt:
		ui.printMsg("Thun done, received interrupt signal.")
	case serverDone:
		ui.printMsg("Thun done, server terminated the session.")
	}
}

func runUDPTest(test *thunTest) {
	server := test.session.remoteAddr
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		go func() {
			buff := make([]byte, test.testParam.BufferSize)
			conn, err := net.Dial("udp", server+":"+udpPort)
			if err != nil {
				ui.printDbg("Unable to dial UDP, error: %v", err)
				return
			}
			defer conn.Close()
			ec := test.newConn(conn)
			rserver, rport, _ := net.SplitHostPort(conn.RemoteAddr().String())
			lserver, lport, _ := net.SplitHostPort(conn.LocalAddr().String())
			ui.printMsg("[%3d] local %s port %s connected to %s port %s",
				ec.fd, lserver, lport, rserver, rport)
			blen := len(buff)
		ExitForLoop:
			for {
				select {
				case <-test.done:
					break ExitForLoop
				default:
					n, err := conn.Write(buff)
					if err != nil {
						ui.printDbg("%v", err)
						continue
					}
					if n < blen {
						ui.printDbg("Partial write: %d", n)
						continue
					}
					atomic.AddUint64(&ec.data, uint64(n))
					atomic.AddUint64(&test.testResult.bpsdata, uint64(n))
					atomic.AddUint64(&test.testResult.ppsdata, 1)
				}
			}
		}()
	}
}
