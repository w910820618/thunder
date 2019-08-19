package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"runtime"
	"time"
)

func runServer(serverParam EthrTestParam) {
	addr, err := net.ResolveUDPAddr("udp", serverParam.thunderParam.host+":"+serverParam.thunderParam.port)
	if err != nil {
		fmt.Println("Can't resolve address: ", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error listening:", err)
	}
	done := make(chan struct{})
	for i := 0; i < runtime.NumCPU(); i++ {
		go handleClient(conn)
	}
	<-done
}

func handleClient(conn *net.UDPConn) {
	for {
		data := make([]byte, 1024)
		n, remoteAddr, err := conn.ReadFromUDP(data)
		if err != nil {
			fmt.Println("failed to read UDP msg because of ", err.Error())
			return
		}
		daytime := time.Now().Unix()
		fmt.Println(n, remoteAddr)
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(daytime))
		conn.WriteToUDP(b, remoteAddr)
	}

}
