//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package stats

type ThunNetStats struct {
	NetDevStats []ThunNetDevStat
	TCPStats    ThunTCPStat
}

type ThunNetDevStat struct {
	InterfaceName string
	RxBytes       uint64
	TxBytes       uint64
	RxPkts        uint64
	TxPkts        uint64
}

type ThunTCPStat struct {
	SegRetrans uint64
}
