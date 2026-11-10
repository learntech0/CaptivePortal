package backend

import (
	"net"
	"regexp"
)

var phoneRe = regexp.MustCompile(`^(?:0[17]\d{8}|254[17]\d{8}|[17]\d{8})$`)
var macRe   = regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`)

func isValidPhone(s string) bool {
	return phoneRe.MatchString(s)
}

func isValidMAC(s string) bool {
	return macRe.MatchString(s) && s != "00:00:00:00:00:00"
}

func isValidRouterIP(s string) bool {
	ip := net.ParseIP(s)
	if ip == nil || ip.To4() == nil {
		return false
	}
	return ip.IsPrivate()
}