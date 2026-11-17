// Package netinfo finds the LAN addresses other devices can reach this PC on.
package netinfo

import (
	"net"
	"sort"
	"strings"
)

// Addr is a usable LAN IPv4 address and the interface it belongs to.
type Addr struct {
	IP    string `json:"ip"`
	Iface string `json:"iface"`
}

type iface struct {
	name  string
	flags net.Flags
	addrs []net.Addr
}

// Interface name fragments for virtual adapters that phones can't reach.
var virtualNames = []string{
	"docker", "br-", "veth", "virbr", "vmnet", "vboxnet", "tun", "tap",
	"wg", "tailscale", "zt", "utun", "vethernet", "virtualbox", "vmware",
	"hyper-v", "loopback", "lxc", "lxd", "podman", "cni", "flannel",
}

var wirelessNames = []string{"wl", "wi-fi", "wifi", "wireless", "wlan"}

// List returns reachable LAN IPv4 addresses, best candidates first.
func List() ([]Addr, error) {
	sysIfs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	ifs := make([]iface, 0, len(sysIfs))
	for _, si := range sysIfs {
		addrs, err := si.Addrs()
		if err != nil {
			continue
		}
		ifs = append(ifs, iface{name: si.Name, flags: si.Flags, addrs: addrs})
	}
	return filter(ifs), nil
}

func hasAny(name string, frags []string) bool {
	n := strings.ToLower(name)
	for _, f := range frags {
		if strings.HasPrefix(n, f) || (len(f) > 3 && strings.Contains(n, f)) {
			return true
		}
	}
	return false
}

func filter(ifs []iface) []Addr {
	type scored struct {
		Addr
		score int
	}
	var out []scored
	for _, i := range ifs {
		if i.flags&net.FlagUp == 0 || i.flags&net.FlagLoopback != 0 || hasAny(i.name, virtualNames) {
			continue
		}
		for _, a := range i.addrs {
			n, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip := n.IP.To4()
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			s := 0
			if hasAny(i.name, wirelessNames) {
				s += 2
			}
			if ip.IsPrivate() {
				s++
			}
			out = append(out, scored{Addr{IP: ip.String(), Iface: i.name}, s})
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].score > out[b].score })
	res := make([]Addr, len(out))
	for i, s := range out {
		res[i] = s.Addr
	}
	return res
}
