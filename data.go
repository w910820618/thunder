package main

import (
	"time"
)

type ethrTestResult struct {
	data uint64
}

//type ethrSession struct {
//	remoteAddr string
//	testCount  uint32
//	tests      map[EthrTestID]*ethrTest
//}

// EthrMsgSyn represents the Syn entity.
type EthrMsgSyn struct {
	// TestParam represents the test parameters.
	TestParam EthrTestParam
}

// EthrMsgAck represents the Ack entity.
type EthrMsgAck struct {
	Cert        []byte
	NapDuration time.Duration
}

// EthrMsgFin represents the Fin entity.
type EthrMsgFin struct {
	// Message represents the message body.
	Message string
}

// EthrMsgBgn represents the Bgn entity.
type EthrMsgBgn struct {
	// UDPPort represents the udp port.
	UDPPort string
}

// EthrMsgEnd represents the End entity.
type EthrMsgEnd struct {
	// Message represents the message body.
	Message string
}

// EthrMsgType represents the message type.
type EthrMsgType uint32

const (
	// EthrInv represents the Inv message.
	EthrInv EthrMsgType = iota

	// EthrSyn represents the Syn message.
	EthrSyn

	// EthrAck represents the Ack message.
	EthrAck

	// EthrFin represents the Fin message.
	EthrFin

	// EthrBgn represents the Bgn message.
	EthrBgn

	// EthrEnd represents the End message.
	EthrEnd
)

// EthrMsgVer represents the message version.
type EthrMsgVer uint32

// EthrMsg represents the message entity.
type EthrMsg struct {
	// Version represents the message version.
	Version EthrMsgVer

	// Type represents the message type.
	Type EthrMsgType

	// Syn represents the Syn value.
	Syn *EthrMsgSyn

	// Ack represents the Ack value.
	Ack *EthrMsgAck

	// Fin represents the Fin value.
	Fin *EthrMsgFin

	// Bgn represents the Bgn value.
	Bgn *EthrMsgBgn

	// End represents the End value.
	End *EthrMsgEnd
}

type ethrClientParam struct {
	duration time.Duration
}

// EthrTestType represents the test type.
type EthrTestType uint32

const (
	// All represents all tests - For now only applicable for servers
	All EthrTestType = iota

	// Bandwidth represents the bandwidth test.
	Bandwidth

	// Cps represents connections/s test.
	Cps

	// Pps represents packets/s test.
	Pps

	// Latency represents the latency test.
	Latency

	// ConnLatency represents connection setup latency.
	ConnLatency
)

// EthrProtocol represents the network protocol.
type EthrProtocol uint32

const (
	// TCP represents the tcp protocol.
	TCP EthrProtocol = iota

	// UDP represents the udp protocol.
	UDP

	// HTTP represents using http protocol.
	HTTP

	// HTTPS represents using https protocol.
	HTTPS

	// ICMP represents the icmp protocol.
	ICMP
)

// EthrTestParam represents the parameters used for the test.
type EthrTestParam struct {
	host string

	port string

	BufferSize uint32

	NumThreads uint32
}

type ethrMode uint32

const (
	ethrModeInv ethrMode = iota
	ethrModeServer
	ethrModeExtServer
	ethrModeClient
	ethrModeExtClient
)
