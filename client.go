package main

import (
	"net"
	"sync/atomic"
	"time"
)

func runClient(testParam EthrTestParam, clientParam ethrClientParam, server string) {
	server = "[" + server + "]"
	test, err := establishSession(testParam, server)
	if err != nil {
		/**
		ui.printErr("%v", err)
		*/

		return
	}
	runTest(test, clientParam.duration)

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
				/**
				ui.printDbg("Unable to dial UDP, error: %v", err)
				*/
				return
			}
			defer conn.Close()

			/**
			rserver, rport, _ := net.SplitHostPort(conn.RemoteAddr().String())
			lserver, lport, _ := net.SplitHostPort(conn.LocalAddr().String())
			ui.printMsg("[udp] local %s port %s connected to %s port %s",
				lserver, lport, rserver, rport)
			*/

			blen := len(buff)
		ExitForLoop:
			for {
				select {
				case <-test.done:
					break ExitForLoop
				default:
					n, err := conn.Write(buff)
					if err != nil {
						/**
						ui.printDbg("%v", err)
						*/
						continue
					}
					if n < blen {
						/**
						ui.printDbg("Partial write: %d", n)
						*/
						continue
					}
					atomic.AddUint64(&test.testResult.data, 1)
				}
			}
		}()
	}
}
