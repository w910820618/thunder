package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var customPortRegex = regexp.MustCompile("(\\w+)=([0-9]+)")

var ctrlPort string
var tcpBandwidthPort, tcpCpsPort, tcpPpsPort, tcpLatencyPort string
var udpBandwidthPort, udpCpsPort, udpPpsPort, udpLatencyPort string
var httpBandwidthPort, httpCpsPort, httpPpsPort, httpLatencyPort string
var httpsBandwidthPort, httpsCpsPort, httpsPpsPort, httpsLatencyPort string

var ctrlBasePort = 8888
var tcpBasePort = 9999
var udpBasePort = 9999
var httpBasePort = 9899
var httpsBasePort = 9799

func generatePortNumbers(customPortString string) {
	portsStr := strings.ToUpper(customPortString)
	data := customPortRegex.FindAllStringSubmatch(portsStr, -1)
	for _, kv := range data {
		k := kv[1]
		v := kv[2]
		p := toInt(v)
		if p == 0 {
			continue
		}
		switch k {
		case "TCP":
			tcpBasePort = p
		case "UDP":
			udpBasePort = p
		case "HTTP":
			httpBasePort = p
		case "HTTPS":
			httpsBasePort = p
		case "CONTROL":
			ctrlBasePort = p
		default:
			//ui.printErr("Ignoring unexpected key in custom ports: %s", k)
		}
	}
	ctrlPort = toString(ctrlBasePort)
	tcpBandwidthPort = toString(tcpBasePort)
	tcpCpsPort = toString(tcpBasePort - 1)
	tcpPpsPort = toString(tcpBasePort - 2)
	tcpLatencyPort = toString(tcpBasePort - 3)
	udpBandwidthPort = toString(udpBasePort)
	udpCpsPort = toString(udpBasePort - 1)
	udpPpsPort = toString(udpBasePort - 2)
	udpLatencyPort = toString(udpBasePort - 3)
	httpBandwidthPort = toString(httpBasePort)
	httpCpsPort = toString(httpBasePort - 1)
	httpPpsPort = toString(httpBasePort - 2)
	httpLatencyPort = toString(httpBasePort - 3)
	httpsBandwidthPort = toString(httpsBasePort)
	httpsCpsPort = toString(httpsBasePort - 1)
	httpsPpsPort = toString(httpsBasePort - 2)
	httpsLatencyPort = toString(httpsBasePort - 3)
}

const (
	// UNO represents 1 unit.
	UNO = 1

	// KILO represents k.
	KILO = 1000

	// MEGA represents m.
	MEGA = 1000 * 1000

	// GIGA represents g.
	GIGA = 1000 * 1000 * 1000

	// TERA represents t.
	TERA = 1000 * 1000 * 1000 * 1000
)

func unitToNumber(s string) uint64 {
	s = strings.TrimSpace(s)
	s = strings.ToUpper(s)

	i := strings.IndexFunc(s, unicode.IsLetter)

	if i == -1 {
		bytes, err := strconv.ParseFloat(s, 64)
		if err != nil || bytes <= 0 {
			return 0
		}
		return uint64(bytes)
	}

	bytesString, multiple := s[:i], s[i:]
	bytes, err := strconv.ParseFloat(bytesString, 64)
	if err != nil || bytes <= 0 {
		return 0
	}

	switch multiple {
	case "T", "TB", "TIB":
		return uint64(bytes * TERA)
	case "G", "GB", "GIB":
		return uint64(bytes * GIGA)
	case "M", "MB", "MIB":
		return uint64(bytes * MEGA)
	case "K", "KB", "KIB":
		return uint64(bytes * KILO)
	case "B":
		return uint64(bytes)
	default:
		return 0
	}
}

func testToString(testType EthrTestType) string {
	switch testType {
	case Bandwidth:
		return "Bandwidth"
	case Cps:
		return "Connections/s"
	case Pps:
		return "Packets/s"
	case Latency:
		return "Latency"
	default:
		return "Invalid"
	}
}

func protoToString(proto EthrProtocol) string {
	switch proto {
	case TCP:
		return "TCP"
	case UDP:
		return "UDP"
	case HTTP:
		return "HTTP"
	case HTTPS:
		return "HTTPS"
	case ICMP:
		return "ICMP"
	}
	return ""
}

func toString(n int) string {
	return fmt.Sprintf("%d", n)
}

func toInt(s string) int {
	res, err := strconv.Atoi(s)
	if err != nil {
		//ui.printDbg("Error in string conversion: %v", err)
		return 0
	}
	return res
}
