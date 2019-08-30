package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"sync/atomic"
	"time"
)

const (
	timeout    = 0
	interrupt  = 1
	serverDone = 2
)

var gIgnoreCert bool

func clientWatchControlChannel(test *thunTest, toStop chan int) {
	go func() {
		waitForChannelStop := make(chan bool, 1)
		watchControlChannel(test, waitForChannelStop)
		<-waitForChannelStop
		toStop <- serverDone
	}()
}

func handleCtrlC(toStop chan int) {
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, os.Kill)
	go func() {
		sig := <-sigChan
		switch sig {
		case os.Interrupt:
			fallthrough
		case os.Kill:
			toStop <- interrupt
		}
	}()
}

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
	initClient()
	test, err := establishSession(testParam, server)
	if err != nil {
		return
	}
	runTest(test, clientParam.duration)
}

func initClient() {
	initClientUI()
}

func establishSession(testParam ThunTestParam, server string) (test *thunTest, err error) {
	conn, err := net.Dial("tcp", server+":"+ctrlPort)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			conn.Close()
		}
	}()
	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)
	thunMsg := createSynMsg(testParam)
	err = sendSessionMsg(enc, thunMsg)
	if err != nil {
		return
	}
	rserver, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	server = "[" + rserver + "]"
	test, err = newTest(server, conn, testParam, enc, dec)
	if err != nil {
		thunMsg = createFinMsg(err.Error())
		sendSessionMsg(enc, thunMsg)
		return
	}
	thunMsg = recvSessionMsg(test.dec)
	if thunMsg.Type != ThunAck {
		if thunMsg.Type == ThunFin {
			err = fmt.Errorf("%s", thunMsg.Fin.Message)
		} else {
			err = fmt.Errorf("Unexpected control message received. %v", thunMsg)
		}
		deleteTest(test)
		return nil, err
	}
	gCert = thunMsg.Ack.Cert
	napDuration := thunMsg.Ack.NapDuration
	time.Sleep(napDuration)
	// TODO: Enable this in future, right now there is not much value coming
	// from this.
	/**
		thunMsg = createAckMsg()
		err = sendSessionMsg(test.enc, thunMsg)
		if err != nil {
			os.Exit(1)
		}
	    **/
	return
}

func runTest(test *thunTest, d time.Duration) {
	startStatsTimer()
	if test.testParam.TestID.Protocol == TCP {
		if test.testParam.TestID.Type == Bandwidth {
			go runTCPBandwidthTest(test)
		} else if test.testParam.TestID.Type == Cps {
			go runTCPCpsTest(test)
		} else if test.testParam.TestID.Type == Latency {
			ui.emitLatencyHdr()
			go runTCPLatencyTest(test)
		}
	} else if test.testParam.TestID.Protocol == UDP {
		go runUDPTest(test)
	} else if test.testParam.TestID.Protocol == HTTP {
		if test.testParam.TestID.Type == Bandwidth {
			go runHTTPBandwidthTest(test)
		} else if test.testParam.TestID.Type == Latency {
			ui.emitLatencyHdr()
			go runHTTPLatencyTest(test)
		}
	} else if test.testParam.TestID.Protocol == HTTPS {
		if test.testParam.TestID.Type == Bandwidth {
			go runHTTPSBandwidthTest(test)
		}
	}
	test.isActive = true
	toStop := make(chan int, 1)
	runDurationTimer(d, toStop)
	clientWatchControlChannel(test, toStop)
	handleCtrlC(toStop)
	reason := <-toStop
	close(test.done)
	sendSessionMsg(test.enc, &ThunMsg{})
	test.ctrlConn.Close()
	stopStatsTimer()
	switch reason {
	case timeout:
		fmt.Printf("Thun done, duration: " + d.String() + ".")
	case interrupt:
		ui.printMsg("Thun done, received interrupt signal.")
	case serverDone:
		ui.printMsg("Thun done, server terminated the session.")
	}
}

func runTCPBandwidthTest(test *thunTest) {
	server := test.session.remoteAddr
	ui.printMsg("Connecting to host %s, port %s", server, tcpBandwidthPort)
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		buff := make([]byte, test.testParam.BufferSize)
		for i := uint32(0); i < test.testParam.BufferSize; i++ {
			buff[i] = byte(i)
		}
		go func() {
			conn, err := net.Dial("tcp", server+":"+tcpBandwidthPort)
			if err != nil {
				ui.printErr("%v", err)
				os.Exit(1)
				return
			}
			defer conn.Close()
			ec := test.newConn(conn)
			rserver, rport, _ := net.SplitHostPort(conn.RemoteAddr().String())
			lserver, lport, _ := net.SplitHostPort(conn.LocalAddr().String())
			ui.printMsg("[%3d] local %s port %s connected to %s port %s",
				ec.fd, lserver, lport, rserver, rport)
			blen := len(buff)
		ExitForLoop:
			for {
				select {
				case <-test.done:
					break ExitForLoop
				default:
					n := 0
					if test.testParam.Reverse {
						n, err = io.ReadFull(conn, buff)
					} else {
						n, err = conn.Write(buff)
					}
					if err != nil || n < blen {
						ui.printDbg("Error sending/receiving data on a connection for bandwidth test: %v", err)
						continue
					}
					atomic.AddUint64(&ec.data, uint64(blen))
					atomic.AddUint64(&test.testResult.bandwidth, uint64(blen))
				}
			}
		}()
	}
}

func runTCPCpsTest(test *thunTest) {
	server := test.session.remoteAddr
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		go func() {
		ExitForLoop:
			for {
				select {
				case <-test.done:
					break ExitForLoop
				default:
					conn, err := net.Dial("tcp", server+":"+tcpCpsPort)
					if err == nil {
						atomic.AddUint64(&test.testResult.data, 1)
						tcpconn, ok := conn.(*net.TCPConn)
						if ok {
							tcpconn.SetLinger(0)
						}
						conn.Close()
					}
				}
			}
		}()
	}
}

func runTCPLatencyTest(test *thunTest) {
	server := test.session.remoteAddr
	conn, err := net.Dial("tcp", server+":"+tcpLatencyPort)
	if err != nil {
		ui.printErr("Error dialing the latency connection: %v", err)
		os.Exit(1)
		return
	}
	defer conn.Close()
	buffSize := test.testParam.BufferSize
	// TODO Override buffer size to 1 for now. Evaluate if we need to allow
	// client to specify the buffer size in future.
	buffSize = 1
	buff := make([]byte, buffSize)
	for i := uint32(0); i < buffSize; i++ {
		buff[i] = byte(i)
	}
	blen := len(buff)
	rttCount := test.testParam.RttCount
	latencyNumbers := make([]time.Duration, rttCount)
ExitForLoop:
	for {
	ExitSelect:
		select {
		case <-test.done:
			break ExitForLoop
		default:
			for i := uint32(0); i < rttCount; i++ {
				s1 := time.Now()
				n, err := conn.Write(buff)
				if err != nil {
					// ui.printErr(err)
					// return
					break ExitSelect
				}
				if n < blen {
					// ui.printErr("Partial write: " + strconv.Itoa(n))
					// return
					break ExitSelect
				}
				_, err = io.ReadFull(conn, buff)
				if err != nil {
					// ui.printErr(err)
					// return
					break ExitSelect
				}
				e2 := time.Since(s1)
				latencyNumbers[i] = e2
			}
			// TODO temp code, fix it better, this is to allow server to do
			// server side latency measurements as well.
			_, _ = conn.Write(buff)

			calcLatency(test, rttCount, latencyNumbers)
		}
	}
}

func runHTTPBandwidthTest(test *thunTest) {
	uri := test.session.remoteAddr
	ui.printMsg("uri=%s", uri)
	uri = "http://" + uri + ":" + httpBandwidthPort
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		buff := make([]byte, test.testParam.BufferSize)
		for i := uint32(0); i < test.testParam.BufferSize; i++ {
			buff[i] = 'x'
		}
		tr := &http.Transport{DisableCompression: true}
		client := &http.Client{Transport: tr}
		go runHTTPandHTTPSBandwidthTest(test, client, uri, buff)
	}

}

func runHTTPSBandwidthTest(test *thunTest) {
	uri := test.session.remoteAddr
	uri = "https://" + uri + ":" + httpsBandwidthPort
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		buff := make([]byte, test.testParam.BufferSize)
		for i := uint32(0); i < test.testParam.BufferSize; i++ {
			buff[i] = 'x'
		}
		c, err := x509.ParseCertificate(gCert)
		if err != nil {
			ui.printErr("Failed to parse certificate: %v", err)
		}
		clientCertPool := x509.NewCertPool()
		clientCertPool.AddCert(c)

		tlsConfig := &tls.Config{
			InsecureSkipVerify: gIgnoreCert,
			// Certificates: []tls.Certificate{cert},
			RootCAs: clientCertPool,
		}
		//tlsConfig.BuildNameToCertificate()
		tr := &http.Transport{DisableCompression: true, TLSClientConfig: tlsConfig}
		client := &http.Client{Transport: tr}
		go runHTTPandHTTPSBandwidthTest(test, client, uri, buff)
	}

}

func runHTTPLatencyTest(test *thunTest) {
	uri := test.session.remoteAddr
	uri = "http://" + uri + ":" + httpLatencyPort

	buff := make([]byte, test.testParam.BufferSize)
	for i := uint32(0); i < test.testParam.BufferSize; i++ {
		buff[i] = 'x'
	}

	rttCount := test.testParam.RttCount
	latencyNumbers := make([]time.Duration, rttCount)
	tr := &http.Transport{DisableCompression: true}
	client := &http.Client{Transport: tr}
ExitForLoop:
	for {
	ExitSelect:
		select {
		case <-test.done:
			break ExitForLoop
		default:
			for i := uint32(0); i < rttCount; i++ {
				s1 := time.Now()
				response, err := client.Post(uri, "text/plain", bytes.NewBuffer(buff))
				if err != nil {
					break ExitSelect
				} else {
					if response.StatusCode != http.StatusOK {
						break ExitSelect
					}
					_, err := ioutil.ReadAll(response.Body)
					response.Body.Close()
					if err != nil {
						break ExitSelect
					}
				}
				e2 := time.Since(s1)
				latencyNumbers[i] = e2
			}

			calcLatency(test, rttCount, latencyNumbers)
		}
	}
}

func runHTTPandHTTPSBandwidthTest(test *thunTest, client *http.Client, uri string, buff []byte) {
ExitForLoop:
	for {
		select {
		case <-test.done:
			break ExitForLoop
		default:
			// response, err := http.Get(uri)
			response, err := client.Post(uri, "text/plain", bytes.NewBuffer(buff))
			if err != nil {
				ui.printDbg("Error in HTTP request: %v", err)
				continue
			} else {
				ui.printDbg("Status received: %v", response.StatusCode)
				if response.StatusCode != http.StatusOK {
					ui.printDbg("Error in HTTP request, received status: %v", response.StatusCode)
					continue
				}
				_, err := ioutil.ReadAll(response.Body)
				response.Body.Close()
				if err != nil {
					ui.printDbg("Error in receving HTTP response: %v", err)
					continue
				}
			}
			atomic.AddUint64(&test.testResult.data, uint64(test.testParam.BufferSize))
		}
	}
}

func runUDPTest(test *thunTest) {
	server := test.session.remoteAddr
	for th := uint32(0); th < test.testParam.NumThreads; th++ {
		go func() {
			buff := make([]byte, test.testParam.BufferSize)
			conn, err := net.Dial("udp", server+":"+udpBandwidthPort)
			if err != nil {
				ui.printDbg("Unable to dial UDP, error: %v", err)
				return
			}
			defer conn.Close()
			ec := test.newConn(conn)
			rserver, rport, _ := net.SplitHostPort(conn.RemoteAddr().String())
			lserver, lport, _ := net.SplitHostPort(conn.LocalAddr().String())
			ui.printMsg("[%3d] local %s port %s connected to %s port %s",
				ec.fd, lserver, lport, rserver, rport)
			blen := len(buff)
		ExitForLoop:
			for {
				select {
				case <-test.done:
					break ExitForLoop
				default:
					n, err := conn.Write(buff)
					if err != nil {
						ui.printDbg("%v", err)
						continue
					}
					if n < blen {
						ui.printDbg("Partial write: %d", n)
						continue
					}
					atomic.AddUint64(&ec.data, uint64(n))
					atomic.AddUint64(&test.testResult.bandwidth, uint64(n))
					atomic.AddUint64(&test.testResult.data, 1)
				}
			}
		}()
	}
}

func calcLatency(test *thunTest, rttCount uint32, latencyNumbers []time.Duration) {
	sum := int64(0)
	for _, d := range latencyNumbers {
		sum += d.Nanoseconds()
	}
	elapsed := time.Duration(sum / int64(rttCount))
	sort.SliceStable(latencyNumbers, func(i, j int) bool {
		return latencyNumbers[i] < latencyNumbers[j]
	})

	rttCountFixed := rttCount
	if rttCountFixed == 1 {
		rttCountFixed = 2
	}
	avg := elapsed
	min := latencyNumbers[0]
	max := latencyNumbers[rttCount-1]
	p50 := latencyNumbers[((rttCountFixed*50)/100)-1]
	p90 := latencyNumbers[((rttCountFixed*90)/100)-1]
	p95 := latencyNumbers[((rttCountFixed*95)/100)-1]
	p99 := latencyNumbers[((rttCountFixed*99)/100)-1]
	p999 := latencyNumbers[uint64(((float64(rttCountFixed)*99.9)/100)-1)]
	p9999 := latencyNumbers[uint64(((float64(rttCountFixed)*99.99)/100)-1)]
	ui.emitLatencyResults(
		test.session.remoteAddr,
		protoToString(test.testParam.TestID.Protocol),
		avg, min, max, p50, p90, p95, p99, p999, p9999)
}
