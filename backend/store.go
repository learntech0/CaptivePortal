package backend

import "sync"

// paidDevices holds MAC addresses that have successfully paid but have not yet
// been fetched/cleared by the router's polling loop.
var (
	paidMu      sync.Mutex
	paidDevices = make(map[string]bool)
)

// AddPaidDevice records a MAC address as paid and pending router clearance.
func AddPaidDevice(mac string) {
	paidMu.Lock()
	defer paidMu.Unlock()
	paidDevices[mac] = true
}

// GetPaidDevices returns (and clears) the list of paid MAC addresses that the
// router has not yet processed. Clearing after fetch prevents the router from
// re-processing the same addresses on every poll.
func GetPaidDevices() []string {
	paidMu.Lock()
	defer paidMu.Unlock()

	out := make([]string, 0, len(paidDevices))
	for mac := range paidDevices {
		out = append(out, mac)
	}
	// Reset the pending queue once the router has fetched it.
	paidDevices = make(map[string]bool)
	return out
}
