package main

import (
	"encoding/binary"
	"fmt"
	"net"
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

func runClient(testParam EthrTestParam, clientParam ethrClientParam) {
	startStatsTimer()
	for th := uint32(0); th < testParam.NumThreads; th++ {
		go func() {
			addr, err := net.ResolveUDPAddr("udp", testParam.host+":"+testParam.port)
			if err != nil {
				fmt.Println("Can't resolve address: ", err)
			}
			conn, err := net.DialUDP("udp", nil, addr)
			if err != nil {
				fmt.Println("Can't dial: ", err)
			}
			defer conn.Close()
			buff := make([]byte, testParam.BufferSize)
			_, err = conn.Write(buff)
			for {

				select {
				default:
					_, err := conn.Write(buff)
					data := make([]byte, 4)
					_, err = conn.Read(data)
					if err != nil {
						fmt.Println("failed to read UDP msg because of ", err)
					}
					t := binary.BigEndian.Uint32(data)
					fmt.Println(time.Unix(int64(t), 0).String())
					if err != nil {
						continue
					}
				}
			}
		}()
	}
	toStop := make(chan int, 1)
	runDurationTimer(clientParam.duration, toStop)
	stopStatsTimer()
	reason := <-toStop
	switch reason {
	case timeout:
		fmt.Println("Ethr done,duration: " + clientParam.duration.String() + ".")
	}
}
