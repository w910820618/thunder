package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"os"
	"runtime"
	"sort"
	"sync/atomic"
	"time"
)

var gCert []byte

func runServer(testParam ThunTestParam, serverParam thunServerParam) {
	defer stopStatsTimer()
	initServer(serverParam.showUI)
	runTCPBandwidthServer()
	runTCPCpsServer()
	runTCPLatencyServer()
	runHTTPBandwidthServer()
	runHTTPSBandwidthServer()
	runHTTPLatencyServer()
	l := runControlChannel()
	defer l.Close()
	startStatsTimer()
	for {
		conn, err := l.Accept()
		if err != nil {
			ui.printErr("Error accepting new control connection: %v", err)
			continue
		}
		go handleRequest(conn)
	}
}

func initServer(showUI bool) {
	initServerUI(showUI)
}

func runControlChannel() net.Listener {
	l, err := net.Listen("tcp", hostAddr+":"+ctrlPort)
	if err != nil {
		finiServer()
		fmt.Printf("Fatal error listening for control connections: %v", err)
		os.Exit(1)
	}
	ui.printMsg("Listening on " + ctrlPort + " for control plane")
	return l
}

func handleRequest(conn net.Conn) {
	defer conn.Close()
	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)
	thunMsg := recvSessionMsg(dec)
	if thunMsg.Type != ThunSyn {
		return
	}
	testParam := thunMsg.Syn.TestParam
	server, port, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		ui.printDbg("RemoteAddr: Split host port failed: %v", err)
		return
	}
	_, _, err = net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		ui.printDbg("LocalAddr: Split host port failed: %v", err)
		return
	}
	ui.printMsg("New control connection from " + server + ", port " + port)
	test, err := newTest(server, conn, testParam, enc, dec)
	cleanupFunc := func() {
		test.ctrlConn.Close()
		close(test.done)
		deleteTest(test)
	}
	ui.emitTestHdr()
	if test.testParam.TestID.Protocol == UDP {
		err = runUDPServer(test)
		if err != nil {
			ui.printDbg("Error encounterd in running UDP test (%s): %v",
				testToString(testParam.TestID.Type), err)
			cleanupFunc()
			return
		}
	}
	delay := timeToNextTick()
	thunMsg = createAckMsg(gCert, delay)
	err = sendSessionMsg(enc, thunMsg)
	if err != nil {
		ui.printErr("send session message: %v", err)
		cleanupFunc()
		return
	}
	time.Sleep(delay)
	test.isActive = true
	waitForChannelStop := make(chan bool, 1)
	serverWatchControlChannel(test, waitForChannelStop)
	<-waitForChannelStop
	test.isActive = false
	ui.printMsg("Ending " + testToString(testParam.TestID.Type) + " test from " + server)
	cleanupFunc()
	if len(gSessionKeys) > 0 {
		ui.emitTestHdr()
	}
}

func serverWatchControlChannel(test *thunTest, waitForChannelStop chan bool) {
	watchControlChannel(test, waitForChannelStop)
}

func finiServer() {
	ui.fini()
	logFini()
}

func runHTTPSBandwidthServer() {
	cert, err := genX509KeyPair()
	if err != nil {
		ui.printErr("Unable to start HTTPS server. Error in X509 certificate: %v", err)
		return
	}
	config := &tls.Config{}
	config.NextProtos = []string{"http/1.1"}
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0] = cert
	sm := http.NewServeMux()
	sm.HandleFunc("/", runHTTPSBandwidthHandler)
	l, err := net.Listen("tcp", ":"+httpsBandwidthPort)
	if err != nil {
		ui.printErr("Unable to start HTTPS server. Error in listening on socket: %v", err)
		return
	}
	ui.printMsg("Listening on " + httpsBandwidthPort + " for HTTPS bandwidth tests")
	tl := tls.NewListener(tcpKeepAliveListener{l.(*net.TCPListener)}, config)
	go runHTTPServer(tl, sm)
}

func runHTTPSBandwidthHandler(w http.ResponseWriter, r *http.Request) {
	runHTTPandHTTPSHandler(w, r, HTTPS, Bandwidth)
}

func runHTTPLatencyServer() {
	sm := http.NewServeMux()
	sm.HandleFunc("/", runHTTPLatencyHandler)
	l, err := net.Listen("tcp", ":"+httpLatencyPort)
	if err != nil {
		ui.printErr("Unable to start HTTP server. Error in listening on socket: %v", err)
		return
	}
	ui.printMsg("Listening on " + httpLatencyPort + " for HTTP latency tests")
	go runHTTPServer(tcpKeepAliveListener{l.(*net.TCPListener)}, sm)
}

func runHTTPLatencyHandler(w http.ResponseWriter, r *http.Request) {
	runHTTPandHTTPSHandler(w, r, HTTP, Latency)
}

func runUDPServer(test *thunTest) error {
	udpAddr, err := net.ResolveUDPAddr("udp", hostAddr+":"+udpBandwidthPort)
	if err != nil {
		ui.printDbg("Unable to resolve UDP address: %v", err)
		return err
	}
	l, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		ui.printDbg("Error listening on %s for UDP pkt/s tests: %v", udpPpsPort, err)
		return err
	}
	go func(l *net.UDPConn) {
		defer l.Close()
		for i := 0; i < runtime.NumCPU(); i++ {
			go runUDPHandler(test, l)
		}
		<-test.done
	}(l)
	return nil
}

func runUDPHandler(test *thunTest, conn *net.UDPConn) {
	buffer := make([]byte, test.testParam.BufferSize)
	n, remoteAddr, err := 0, new(net.UDPAddr), error(nil)
	for err == nil {
		n, remoteAddr, err = conn.ReadFromUDP(buffer)
		if err != nil {
			ui.printDbg("Error receiving data from UDP for bandwidth test: %v", err)
			continue
		}
		thunUnused(n)
		server, port, _ := net.SplitHostPort(remoteAddr.String())
		test := getTest(server, UDP, Bandwidth)
		if test != nil {
			atomic.AddUint64(&test.testResult.bandwidth, uint64(n))
			atomic.AddUint64(&test.testResult.data, 1)
		} else {
			ui.printDbg("Received unsolicited UDP traffic on port %s from %s port %s", udpPpsPort, server, port)
		}
	}
}
func closeConn(conn net.Conn) {
	err := conn.Close()
	if err != nil {
		ui.printDbg("Failed to close TCP connection, error: %v", err)
	}
}

func runHTTPBandwidthServer() {
	sm := http.NewServeMux()
	sm.HandleFunc("/", runHTTPBandwidthHandler)
	l, err := net.Listen("tcp", ":"+httpBandwidthPort)
	if err != nil {
		ui.printErr("Unable to start HTTP server. Error in listening on socket: %v", err)
		return
	}
	ui.printMsg("Listening on " + httpBandwidthPort + " for HTTP bandwidth tests")
	go runHTTPServer(tcpKeepAliveListener{l.(*net.TCPListener)}, sm)
}

func runHTTPBandwidthHandler(w http.ResponseWriter, r *http.Request) {
	runHTTPandHTTPSHandler(w, r, HTTP, Bandwidth)
}

func runHTTPandHTTPSHandler(w http.ResponseWriter, r *http.Request, p ThunProtocol, testType ThunTestType) {
	_, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ui.printDbg("Error reading HTTP body: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	server, _, _ := net.SplitHostPort(r.RemoteAddr)
	test := getTest(server, p, testType)
	if test == nil {
		http.Error(w, "Unauthorized request.", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case "GET":
		w.Write([]byte("OK."))
	case "PUT":
		w.Write([]byte("OK."))
	case "POST":
		w.Write([]byte("OK."))
	default:
		http.Error(w, "Only GET, PUT and POST are supported.", http.StatusMethodNotAllowed)
		return
	}
	if testType == Bandwidth {
		if r.ContentLength > 0 {
			atomic.AddUint64(&test.testResult.data, uint64(r.ContentLength))
		}
	}
}

func runHTTPServer(l net.Listener, handler http.Handler) error {
	err := http.Serve(l, handler)
	if err != nil {
		ui.printErr("Unable to start HTTP server, error: %v", err)
	}
	return err
}

func runTCPBandwidthServer() {
	l, err := net.Listen("tcp", hostAddr+":"+tcpBandwidthPort)
	if err != nil {
		finiServer()
		fmt.Printf("Fatal error listening on "+tcpBandwidthPort+" for TCP bandwidth tests: %v", err)
		os.Exit(1)
	}
	ui.printMsg("Listening on " + tcpBandwidthPort + " for TCP bandwidth tests")
	go func(l net.Listener) {
		defer l.Close()
		for {
			conn, err := l.Accept()
			if err != nil {
				ui.printErr("Error accepting new bandwidth connection: %v", err)
				continue
			}
			server, port, _ := net.SplitHostPort(conn.RemoteAddr().String())
			test := getTest(server, TCP, Bandwidth)
			if test == nil {
				ui.printDbg("Received unsolicited TCP connection on port %s from %s port %s", tcpBandwidthPort, server, port)
				conn.Close()
				continue
			}
			go runTCPBandwidthHandler(conn, test)
		}
	}(l)
}

func runTCPBandwidthHandler(conn net.Conn, test *thunTest) {
	defer closeConn(conn)
	size := test.testParam.BufferSize
	buff := make([]byte, size)
	for i := uint32(0); i < test.testParam.BufferSize; i++ {
		buff[i] = byte(i)
	}
ExitForLoop:
	for {
		select {
		case <-test.done:
			break ExitForLoop
		default:
			var err error
			if test.testParam.Reverse {
				_, err = conn.Write(buff)
			} else {
				_, err = io.ReadFull(conn, buff)
			}
			if err != nil {
				ui.printDbg("Error sending/receiving data on a connection for bandwidth test: %v", err)
				continue
			}
			atomic.AddUint64(&test.testResult.data, uint64(size))
		}
	}
}

func runTCPCpsServer() {
	l, err := net.Listen("tcp", hostAddr+":"+tcpCpsPort)
	if err != nil {
		finiServer()
		fmt.Printf("Fatal error listening on "+tcpCpsPort+" for TCP conn/s tests: %v", err)
		os.Exit(1)
	}
	ui.printMsg("Listening on " + tcpCpsPort + " for TCP conn/s tests")
	go func(l net.Listener) {
		defer l.Close()
		for {
			conn, err := l.Accept()
			if err != nil {
				ui.printDbg("Error accepting new conn/s connection: %v", err)
				continue
			}
			go runTCPCpsHandler(conn)
		}
	}(l)
}

func runTCPCpsHandler(conn net.Conn) {
	defer conn.Close()
	server, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	test := getTest(server, TCP, Cps)
	if test != nil {
		atomic.AddUint64(&test.testResult.data, 1)
	} else {
		ui.printDbg("Error: Unsolicited connection received.")
	}
}

func runTCPLatencyServer() {
	l, err := net.Listen("tcp", hostAddr+":"+tcpLatencyPort)
	if err != nil {
		finiServer()
		fmt.Printf("Fatal error listening on "+tcpLatencyPort+" for TCP latency tests: %v", err)
		os.Exit(1)
	}
	ui.printMsg("Listening on " + tcpLatencyPort + " for TCP latency tests")
	go func(l net.Listener) {
		defer l.Close()
		for {
			conn, err := l.Accept()
			if err != nil {
				ui.printErr("Error accepting new latency connection: %v", err)
				continue
			}
			server, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
			test := getTest(server, TCP, Latency)
			if test == nil {
				conn.Close()
				continue
			}
			ui.emitLatencyHdr()
			go runTCPLatencyHandler(conn, test)
		}
	}(l)
}

func runTCPLatencyHandler(conn net.Conn, test *thunTest) {
	defer conn.Close()
	bytes := make([]byte, test.testParam.BufferSize)
	// TODO Override buffer size to 1 for now. Evaluate if we need to allow
	// client to specify the buffer size in future.
	bytes = make([]byte, 1)
	rttCount := test.testParam.RttCount
	latencyNumbers := make([]time.Duration, rttCount)
	for {
		_, err := io.ReadFull(conn, bytes)
		if err != nil {
			ui.printDbg("Error receiving data for latency test: %v", err)
			return
		}
		for i := uint32(0); i < rttCount; i++ {
			s1 := time.Now()
			_, err = conn.Write(bytes)
			if err != nil {
				ui.printDbg("Error sending data for latency test: %v", err)
				return
			}
			_, err = io.ReadFull(conn, bytes)
			if err != nil {
				ui.printDbg("Error receiving data for latency test: %v", err)
				return
			}
			e2 := time.Since(s1)
			latencyNumbers[i] = e2
		}
		sum := int64(0)
		for _, d := range latencyNumbers {
			sum += d.Nanoseconds()
		}
		elapsed := time.Duration(sum / int64(rttCount))
		sort.SliceStable(latencyNumbers, func(i, j int) bool {
			return latencyNumbers[i] < latencyNumbers[j]
		})
		//
		// Special handling for rttCount == 1. This prevents negative index
		// in the latencyNumber index. The other option is to use
		// roundUpToZero() but that is more expensive.
		//
		rttCountFixed := rttCount
		if rttCountFixed == 1 {
			rttCountFixed = 2
		}
		atomic.SwapUint64(&test.testResult.data, uint64(elapsed.Nanoseconds()))
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
}

func genX509KeyPair() (tls.Certificate, error) {
	now, _ := time.Parse(time.RFC3339, "2019-01-01T00:00:00Z")
	template := &x509.Certificate{
		SerialNumber: big.NewInt(now.Unix()),
		Subject: pkix.Name{
			CommonName:         "localhost",
			Country:            []string{"USA"},
			Organization:       []string{"localhost"},
			OrganizationalUnit: []string{"127.0.0.1"},
		},
		NotBefore:    now,
		NotAfter:     now.AddDate(100, 0, 0), // Valid for 100 years
		SubjectKeyId: []byte{113, 117, 105, 99, 107, 115, 101, 114, 118, 101},
		// IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		IPAddresses:           allLocalIPs(),
		DNSNames:              []string{"localhost", "*"},
		BasicConstraintsValid: true,
		IsCA:        true,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	cert, err := x509.CreateCertificate(rand.Reader, template, template,
		priv.Public(), priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	gCert = cert

	var outCert tls.Certificate
	outCert.Certificate = append(outCert.Certificate, cert)
	outCert.PrivateKey = priv

	return outCert, nil
}

func allLocalIPs() (ipList []net.IP) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil {
				continue
			}
			ipList = append(ipList, ip)
		}
	}
	return
}
