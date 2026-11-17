package netinfo

import (
	"net"
	"testing"
)

func ipnet(s string) net.Addr {
	ip, n, _ := net.ParseCIDR(s)
	n.IP = ip
	return n
}

func TestFilter(t *testing.T) {
	up := net.FlagUp
	ifs := []iface{
		{name: "lo", flags: up | net.FlagLoopback, addrs: []net.Addr{ipnet("127.0.0.1/8")}},
		{name: "docker0", flags: up, addrs: []net.Addr{ipnet("172.17.0.1/16")}},
		{name: "vEthernet (WSL)", flags: up, addrs: []net.Addr{ipnet("172.28.0.1/20")}},
		{name: "eth0", flags: up, addrs: []net.Addr{ipnet("10.0.0.5/24"), ipnet("fe80::1/64")}},
		{name: "wlan0", flags: up, addrs: []net.Addr{ipnet("192.168.1.20/24"), ipnet("169.254.3.3/16")}},
		{name: "eth1", flags: 0, addrs: []net.Addr{ipnet("192.168.9.9/24")}},
	}
	got := filter(ifs)
	want := []Addr{{IP: "192.168.1.20", Iface: "wlan0"}, {IP: "10.0.0.5", Iface: "eth0"}}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	}
}

func TestListRuns(t *testing.T) {
	if _, err := List(); err != nil {
		t.Fatal(err)
	}
}
