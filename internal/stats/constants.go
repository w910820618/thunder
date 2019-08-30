package stats

type ThunNetStats struct {
	NetDevStats []ThunNetDevStat
	TCPStats    thunTCPStat
}

type ThunNetDevStat struct {
	InterfaceName string
	RxBytes       uint64
	TxBytes       uint64
	RxPkts        uint64
	TxPkts        uint64
}

type thunTCPStat struct {
	SegRetrans uint64
}
