package main

import (
	"encoding/gob"
	"fmt"
	"net"
	"sync/atomic"
	"time"
)

func runClient(testParam EthrTestParam, clientParam ethrClientParam, server string) {
	server = "[" + server + "]"
	test, err := establishSession(testParam, server)
	if err != nil {
		ui.printErr("%v", err)
		return
	}
	runTest(test, clientParam.duration)

}

func establishSession(testParam EthrTestParam, server string) (test *ethrTest, err error) {
	conn, err := net.Dial(tcp(ipVer), server+":"+ctrlPort)
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

func runTest(test *ethrTest, d time.Duration) {
	startStatsTimer()
	if test.testParam.TestID.Protocol == UDP {
		if test.testParam.TestID.Type == Pps {
			go runUDPPpsTest(test)
		}
	}

	test.isActive = true
	close(test.done)
	test.ctrlConn.Close()
	stopStatsTimer()
}

func runUDPPpsTest(test *ethrTest) {
	server := test.session.remoteAddr
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		go func() {
			buff := make([]byte, test.testParam.BufferSize)
			conn, err := net.Dial(udp(ipVer), server+":"+udpPpsPort) // send udp package
			if err != nil {
				ui.printDbg("Unable to dial UDP, error: %v", err)
				return
			}
			defer conn.Close()
			rserver, rport, _ := net.SplitHostPort(conn.RemoteAddr().String())
			lserver, lport, _ := net.SplitHostPort(conn.LocalAddr().String())
			ui.printMsg("[udp] local %s port %s connected to %s port %s",
				lserver, lport, rserver, rport)
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
					atomic.AddUint64(&test.testResult.data, 1)
				}
			}
		}()
	}
}
