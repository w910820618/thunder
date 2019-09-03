package main

import (
	"container/list"
	"encoding/gob"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

var wg sync.WaitGroup

type thunServerParam struct {
	showUI bool
}

type thunClientParam struct {
	duration time.Duration
}

// ThunMsgType represents the message type.
type ThunMsgType uint32

const (
	// ThunInv represents the Inv message.
	ThunInv ThunMsgType = iota

	// ThunSyn represents the Syn message.
	ThunSyn

	// ThunAck represents the Ack message.
	ThunAck

	// ThunFin represents the Fin message.
	ThunFin

	// ThunBgn represents the Bgn message.
	ThunBgn

	// ThunEnd represents the End message.
	ThunEnd
)

// ThunMsgVer represents the message version.
type ThunMsgVer uint32

// ThunMsg represents the message entity.
type ThunMsg struct {
	// Version represents the message version.
	Version ThunMsgVer

	// Type represents the message type.
	Type ThunMsgType

	// Syn represents the Syn value.
	Syn *ThunMsgSyn

	// Ack represents the Ack value.
	Ack *ThunMsgAck

	// Fin represents the Fin value.
	Fin *ThunMsgFin

	// Bgn represents the Bgn value.
	Bgn *ThunMsgBgn

	// End represents the End value.
	End *ThunMsgEnd
}

// ThunMsgSyn represents the Syn entity.
type ThunMsgSyn struct {
	// TestParam represents the test parameters.
	TestParam ThunTestParam
}

// ThunMsgAck represents the Ack entity.
type ThunMsgAck struct {
	Cert        []byte
	NapDuration time.Duration
}

// ThunMsgFin represents the Fin entity.
type ThunMsgFin struct {
	// Message represents the message body.
	Message string
}

// ThunMsgBgn represents the Bgn entity.
type ThunMsgBgn struct {
	// UDPPort represents the udp port.
	UDPPort string
}

// ThunMsgEnd represents the End entity.
type ThunMsgEnd struct {
	// Message represents the message body.
	Message string
}

// ThunTestType represents the test type.
type ThunTestType uint32

const (
	// All represents all tests - For now only applicable for servers
	All ThunTestType = iota

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

// ThunProtocol represents the network protocol.
type ThunProtocol uint32

const (
	// TCP represents the tcp protocol.
	TCP ThunProtocol = iota

	// UDP represents the udp protocol.
	UDP

	// HTTP represents using http protocol.
	HTTP

	// HTTPS represents using https protocol.
	HTTPS

	// ICMP represents the icmp protocol.
	ICMP
)

type ThunTestID struct {
	Protocol ThunProtocol

	Type ThunTestType
}

type ThunTestParam struct {
	TestID ThunTestID

	BufferSize uint32

	NumThreads uint32
}

type thunSession struct {
	remoteAddr string
	testCount  uint32
	tests      map[ThunTestID]*thunTest
}

type thunTestResult struct {
	bpsdata uint64
	ppsdata uint64
}

type thunTest struct {
	isActive   bool
	session    *thunSession
	ctrlConn   net.Conn
	refCount   int32
	enc        *gob.Encoder
	dec        *gob.Decoder
	rcvdMsgs   chan *ThunMsg
	testParam  ThunTestParam
	testResult thunTestResult
	done       chan struct{}
	connList   *list.List
}
type thunConn struct {
	data    uint64
	test    *thunTest
	conn    net.Conn
	elem    *list.Element
	fd      uintptr
	retrans uint64
}

type thunMode uint32

const (
	thunModeInv thunMode = iota

	thunModeServer

	thunModeClient
)

var gSessions = make(map[string]*thunSession)
var gSessionKeys = make([]string, 0)
var gSessionLock sync.RWMutex

func newTest(remoteAddr string, conn net.Conn, testParam ThunTestParam, enc *gob.Encoder, dec *gob.Decoder) (*thunTest, error) {
	gSessionLock.Lock()
	defer gSessionLock.Unlock()
	return newTestInternal(remoteAddr, conn, testParam, enc, dec)
}

func newTestInternal(remoteAddr string, conn net.Conn, testParam ThunTestParam, enc *gob.Encoder, dec *gob.Decoder) (*thunTest, error) {
	var session *thunSession
	session, found := gSessions[remoteAddr]
	if !found {
		session = &thunSession{}
		session.remoteAddr = remoteAddr
		session.tests = make(map[ThunTestID]*thunTest)
		gSessions[remoteAddr] = session
		gSessionKeys = append(gSessionKeys, remoteAddr)
	}

	test, found := session.tests[testParam.TestID]
	if found {
		return test, os.ErrExist
	}

	session.testCount++
	test = &thunTest{}
	test.session = session
	test.ctrlConn = conn
	test.refCount = 0
	test.enc = enc
	test.dec = dec
	test.rcvdMsgs = make(chan *ThunMsg)
	test.testParam = testParam
	test.done = make(chan struct{})
	test.connList = list.New()
	session.tests[testParam.TestID] = test
	return test, nil
}

func getTest(remoteAddr string, proto ThunProtocol, testType ThunTestType) (test *thunTest) {
	gSessionLock.RLock()
	defer gSessionLock.RUnlock()
	return getTestInternal(remoteAddr, proto, testType)
}

func getTestInternal(remoteAddr string, proto ThunProtocol, testType ThunTestType) (test *thunTest) {
	test = nil
	session, found := gSessions[remoteAddr]
	if !found {
		return
	}
	test, _ = session.tests[ThunTestID{proto, testType}]
	return
}

func watchControlChannel(test *thunTest, waitForChannelStop chan bool) {
	go func() {
		for {
			thunMsg := recvSessionMsg(test.dec)
			if thunMsg.Type == ThunInv {
				break
			}
			test.rcvdMsgs <- thunMsg
			ui.printDbg("%v", thunMsg)
		}
		waitForChannelStop <- true
	}()
}

func recvSessionMsg(dec *gob.Decoder) (thunMsg *ThunMsg) {
	thunMsg = &ThunMsg{}
	err := dec.Decode(thunMsg)
	if err != nil {
		ui.printDbg("Error receiving message on control channel: %v", err)
		thunMsg.Type = ThunInv
	}
	return
}

func deleteTest(test *thunTest) {
	gSessionLock.Lock()
	defer gSessionLock.Unlock()
	deleteTestInternal(test)
}

func deleteTestInternal(test *thunTest) {
	session := test.session
	testID := test.testParam.TestID
	//
	// TODO fix this, we need to decide where to close this, inside this
	// function or by the caller. The reason we may need it to be done by
	// the caller is, because done is used for test done notification and
	// there may be some time after done that consumers are still accessing it
	//
	// Since we have not added any refCounting on test object, we are doing
	// hacky timeout based solution by closing "done" outside and sleeping
	// for sufficient time. ugh!
	//
	// close(test.done)
	// test.ctrlConn.Close()
	// test.session = nil
	// test.connList = test.connList.Init()
	//
	delete(session.tests, testID)
	session.testCount--

	if session.testCount == 0 {
		deleteKey(session.remoteAddr)
		delete(gSessions, session.remoteAddr)
	}
}

func deleteKey(key string) {
	i := 0
	for _, x := range gSessionKeys {
		if x != key {
			gSessionKeys[i] = x
			i++
		}
	}
	gSessionKeys = gSessionKeys[:i]
}

func getFd(conn net.Conn) uintptr {
	var fd uintptr
	var rc syscall.RawConn
	var err error
	switch ct := conn.(type) {
	case *net.TCPConn:
		rc, err = ct.SyscallConn()
		if err != nil {
			return 0
		}
	case *net.UDPConn:
		rc, err = ct.SyscallConn()
		if err != nil {
			return 0
		}
	default:
		return 0
	}
	fn := func(s uintptr) {
		fd = s
	}
	rc.Control(fn)
	return fd
}

func sendSessionMsg(enc *gob.Encoder, thunMsg *ThunMsg) error {
	err := enc.Encode(thunMsg)
	if err != nil {
		ui.printDbg("Error sending message on control channel. Message: %v, Error: %v", thunMsg, err)
	}
	return err
}

func createAckMsg(cert []byte, d time.Duration) (thunMsg *ThunMsg) {
	thunMsg = &ThunMsg{Version: 0, Type: ThunAck}
	thunMsg.Ack = &ThunMsgAck{}
	thunMsg.Ack.Cert = cert
	thunMsg.Ack.NapDuration = d
	return
}

func createFinMsg(message string) (thunMsg *ThunMsg) {
	thunMsg = &ThunMsg{Version: 0, Type: ThunFin}
	thunMsg.Fin = &ThunMsgFin{}
	thunMsg.Fin.Message = message
	return
}

func createSynMsg(testParam ThunTestParam) (thunMsg *ThunMsg) {
	thunMsg = &ThunMsg{Version: 0, Type: ThunSyn}
	thunMsg.Syn = &ThunMsgSyn{}
	thunMsg.Syn.TestParam = testParam
	return
}

func (test *thunTest) newConn(conn net.Conn) (ec *thunConn) {
	gSessionLock.Lock()
	defer gSessionLock.Unlock()
	ec = &thunConn{}
	ec.test = test
	ec.conn = conn
	ec.fd = getFd(conn)
	ec.elem = test.connList.PushBack(ec)
	return
}

func (test *thunTest) connListDo(f func(*thunConn)) {
	gSessionLock.RLock()
	defer gSessionLock.RUnlock()
	for e := test.connList.Front(); e != nil; e = e.Next() {
		ec := e.Value.(*thunConn)
		f(ec)
	}
}
