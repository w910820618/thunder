package main

import (
	"os"
	"sync"
	"time"
)

type thunClientParam struct {
	duration time.Duration
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
	data uint64
}

type thunTest struct {
	isActive   bool
	session    *thunSession
	testParam  ThunTestParam
	testResult thunTestResult
	done       chan struct{}
}

type ethrMode uint32

const (
	ethrModeInv ethrMode = iota

	ethrModeServer

	ethrModeClient
)

var gSessions = make(map[string]*thunSession)
var gSessionKeys = make([]string, 0)
var gSessionLock sync.RWMutex

func newTest(remoteAddr string, testParam ThunTestParam) (*thunTest, error) {
	gSessionLock.Lock()
	defer gSessionLock.Unlock()
	return newTestInternal(remoteAddr, testParam)
}

func newTestInternal(remoteAddr string, testParam ThunTestParam) (*thunTest, error) {
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
	test.testParam = testParam
	test.done = make(chan struct{})
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
