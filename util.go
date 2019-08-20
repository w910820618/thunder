package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var customPortRegex = regexp.MustCompile("(\\w+)=([0-9]+)")

const hostAddr = ""

var udpPpsPort string

func generatePortNumbers(customPortString string) {
	udpPpsPort = customPortString
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

func testToString(testType ThunTestType) string {
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

func protoToString(proto ThunProtocol) string {
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
