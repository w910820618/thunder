package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

func runClient(clientParam EthrTestParam) {
	done := make(chan struct{})
	for th := uint32(0); th < clientParam.NumThreads; th++ {
		go func() {
			addr, err := net.ResolveUDPAddr("udp", clientParam.thunderParam.host+":"+clientParam.thunderParam.port)
			if err != nil {
				fmt.Println("Can't resolve address: ", err)
			}
			conn, err := net.DialUDP("udp", nil, addr)
			if err != nil {
				fmt.Println("Can't dial: ", err)
			}
			defer conn.Close()
			buff := make([]byte, clientParam.BufferSize)
			_, err = conn.Write(buff)
			for {
				select {
				case <-done:
					break
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
	<-done
}
