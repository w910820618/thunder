package main

import (
	"fmt"
	"net"
)

func runClient(clientParam thunderClientParam) {
	addr, err := net.ResolveUDPAddr("udp", clientParam.host+":"+clientParam.port)
	if err != nil {
		fmt.Println("Can't resolve address: ", err)
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Println("Can't dial: ", err)
	}
	defer conn.Close()
	buff := make([]byte, clientParam.bufLen)
	_, err = conn.Write(buff)
	for {
		_, err := conn.Write(buff)
		if err != nil {
			continue
		}
	}
}
