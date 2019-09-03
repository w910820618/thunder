package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var customPortRegex = regexp.MustCompile("(\\w+)=([0-9]+)")

var hostAddr string

var udpPpsPort, ctrlPort, udpPort string

var ctrlBasePort = 8888
var udpBasePort = 9999

func generateAddr(hostAddrStr string) {
	ctrlPort = toString(ctrlBasePort)
	udpPort = toString(udpBasePort)
	hostAddr = hostAddrStr
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

func truncateString(str string, num int) string {
	s := str
	l := len(str)
	if l > num {
		if num > 3 {
			s = "..." + str[l-num+3:l]
		} else {
			s = str[l-num : l]
		}
	}
	return s
}

func splitString(longString string, maxLen int) []string {
	splits := []string{}

	var l, r int
	for l, r = 0, maxLen; r < len(longString); l, r = r, r+maxLen {
		for !utf8.RuneStart(longString[r]) {
			r--
		}
		splits = append(splits, longString[l:r])
	}
	splits = append(splits, longString[l:])
	return splits
}

func durationToString(d time.Duration) string {
	if d < 0 {
		return d.String()
	}
	ud := uint64(d)
	val := float64(ud)
	unit := ""
	if ud < uint64(60*time.Second) {
		switch {
		case ud < uint64(time.Microsecond):
			unit = "ns"
		case ud < uint64(time.Millisecond):
			val = val / 1000
			unit = "us"
		case ud < uint64(time.Second):
			val = val / (1000 * 1000)
			unit = "ms"
		default:
			val = val / (1000 * 1000 * 1000)
			unit = "s"
		}

		result := strconv.FormatFloat(val, 'f', 2, 64)
		return result + unit
	}

	return d.String()
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

func thunUnused(vals ...interface{}) {
	for _, val := range vals {
		_ = val
	}
}

func bytesToRate(bytes uint64) string {
	bits := bytes * 8
	result := numberToUnit(bits)
	return result
}

func ppsToString(pps uint64) string {
	result := numberToUnit(pps)
	return result
}

func numberToUnit(num uint64) string {
	unit := ""
	value := float64(num)

	switch {
	case num >= TERA:
		unit = "T"
		value = value / TERA
	case num >= GIGA:
		unit = "G"
		value = value / GIGA
	case num >= MEGA:
		unit = "M"
		value = value / MEGA
	case num >= KILO:
		unit = "K"
		value = value / KILO
	}

	result := strconv.FormatFloat(value, 'f', 2, 64)
	result = strings.TrimSuffix(result, ".00")
	return result + unit
}
