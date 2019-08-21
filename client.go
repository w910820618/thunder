package main

import (
	"encoding/gob"
	"fmt"
	"net"
	"sync/atomic"
	"time"
)

const (
	timeout = 0
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
	ethrMsg := createSynMsg(testParam)
	err = sendSessionMsg(enc, ethrMsg)
	if err != nil {
		return
	}
	rserver, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	server = "[" + rserver + "]"
	test, err = newTest(server, conn, testParam, enc, dec)
	if err != nil {
		ethrMsg = createFinMsg(err.Error())
		sendSessionMsg(enc, ethrMsg)
		return
	}
	ethrMsg = recvSessionMsg(test.dec)
	if ethrMsg.Type != EthrAck {
		if ethrMsg.Type == EthrFin {
			err = fmt.Errorf("%s", ethrMsg.Fin.Message)
		} else {
			err = fmt.Errorf("Unexpected control message received. %v", ethrMsg)
		}
		deleteTest(test)
		return nil, err
	}
	gCert = ethrMsg.Ack.Cert
	napDuration := ethrMsg.Ack.NapDuration
	time.Sleep(napDuration)
	// TODO: Enable this in future, right now there is not much value coming
	// from this.
	/**
		ethrMsg = createAckMsg()
		err = sendSessionMsg(test.enc, ethrMsg)
		if err != nil {
			os.Exit(1)
		}
	    **/
	return
}

func runTest(test *thunTest, d time.Duration) {
	startStatsTimer()
	go runUDPPpsTest(test)
	test.isActive = true
	toStop := make(chan int, 1)
	runDurationTimer(d, toStop)
	reason := <-toStop
	close(test.done)
	sendSessionMsg(test.enc, &EthrMsg{})
	test.ctrlConn.Close()
	stopStatsTimer()
	switch reason {
	case timeout:
		fmt.Printf("Ethr done, duration: " + d.String() + ".")
	}
}

func runUDPPpsTest(test *thunTest) {
	server := test.session.remoteAddr
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		go func() {
			conn, err := net.Dial("udp", server+":"+udpPpsPort)
			if err != nil {
				fmt.Println("Can't dial: ", err)
			}
			defer conn.Close()
			buff := make([]byte, test.testParam.BufferSize)
			for {
				select {
				default:
					_, err := conn.Write(buff)
					if err != nil {
						continue
					}
					atomic.AddUint64(&test.testResult.data, 1)
				}
			}
		}()
	}
}
