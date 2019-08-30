//-----------------------------------------------------------------------------
// Copyright (C) Microsoft. All rights reserved.
// Licensed under the MIT license.
// See LICENSE.txt file in the project root for full license information.
//-----------------------------------------------------------------------------
package stats

type OSStats interface {
	GetNetDevStats() ([]ThunNetDevStat, error)
	GetTCPStats() (thunTCPStat, error)
}

// GetOSStats returns thun-relevant OS statistics, dependant on the build flag
func GetOSStats() OSStats {
	return osStats{}
}
