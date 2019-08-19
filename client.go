package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"time"
)

func runClient(clientParam thunderClientParam) {
	addr, err := net.ResolveUDPAddr("udp", clientParam.host+":"+clientParam.port)
	if err != nil {
		fmt.Println("Can't resolve address: ", err)
		os.Exit(1)
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Println("Can't dial: ", err)
		os.Exit(1)
	}
	defer conn.Close()
	buff := make([]byte, clientParam.bufLen)
	_, err = conn.Write(buff)
	if err != nil {
		fmt.Println("failed:", err)
		os.Exit(1)
	}
	data := make([]byte, clientParam.bufLen)
	_, err = conn.Read(data)
	if err != nil {
		fmt.Println("failed to read UDP msg because of ", err)
		os.Exit(1)
	}
	t := binary.BigEndian.Uint32(data)
	fmt.Println(time.Unix(int64(t), 0).String())
	os.Exit(0)
}
