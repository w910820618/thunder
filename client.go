package main

import (
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
	test, err := establishSession(testParam, server)
	if err != nil {
		return
	}
	runTest(test, clientParam.duration)
}

func establishSession(testParam ThunTestParam, server string) (test *thunTest, err error) {
	test, err = newTest(server, testParam)
	return test, nil
}

func runTest(test *thunTest, d time.Duration) {
	startStatsTimer()
	go runUDPPpsTest(test)
	test.isActive = true
	toStop := make(chan int, 1)
	runDurationTimer(d, toStop)
	reason := <-toStop
	close(test.done)
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
					fmt.Printf("-----------\n")
					atomic.AddUint64(&test.testResult.data, 1)
				}
			}
		}()
	}
}
