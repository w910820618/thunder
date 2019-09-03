//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package stats

type OSStats interface {
	GetNetDevStats() ([]ThunNetDevStat, error)
	GetTCPStats() (ThunTCPStat, error)
}

// GetOSStats returns Thun-relevant OS statistics, dependant on the build flag
func GetOSStats() OSStats {
	return osStats{}
}
