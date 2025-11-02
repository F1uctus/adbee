package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grandcat/zeroconf"
	"github.com/skip2/go-qrcode"
)

const (
	servicePairing = "_adb-tls-pairing._tcp"
	serviceConnect = "_adb-tls-connect._tcp"

	defaultAttempts     = 30
	defaultQueryTimeout = 2 * time.Second
	defaultSleepBetween = 1 * time.Second
	defaultConnectWait  = 12 * time.Second
)

type Config struct {
	DeviceName   string
	Password     string
	Attempts     int
	QueryTimeout time.Duration
	SleepBetween time.Duration
	ConnectWait  time.Duration
}

func generateID() string { return uuid.New().String()[:8] }

func showQR(name, password string) {
	text := fmt.Sprintf("WIFI:T:ADB;S:%s;P:%s;;", name, password)
	qr, err := qrcode.New(text, qrcode.Medium)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to generate QR code: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(qr.ToSmallString(false))
}

func pairDevice(address string, port int, password string) bool {
	cmd := exec.Command("adb", "pair", fmt.Sprintf("%s:%d", address, port), password)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func resolveServiceHost(e *zeroconf.ServiceEntry) string {
	if len(e.AddrIPv4) > 0 {
		return e.AddrIPv4[0].String()
	}
	if len(e.AddrIPv6) > 0 {
		ip := e.AddrIPv6[0]
		if !ip.IsLinkLocalUnicast() {
			return "[" + ip.String() + "]"
		}
	}
	if e.HostName != "" {
		if ips, err := net.LookupIP(e.HostName); err == nil {
			for _, ip := range ips {
				if v4 := ip.To4(); v4 != nil {
					return v4.String()
				}
			}
		}
	}
	return ""
}

func connectToDevice(address string, port int) bool {
	endpointHost := address
	if strings.Contains(address, ":") && !strings.HasPrefix(address, "[") {
		endpointHost = fmt.Sprintf("[%s]", address)
	}
	endpoint := fmt.Sprintf("%s:%d", endpointHost, port)
	cmd := exec.Command("adb", "connect", endpoint)
	if err := cmd.Run(); err != nil {
		return false
	}
	fmt.Printf("Connected to device: %s\n", endpoint)
	return true
}

func browseServices(
	serviceTypes []string,
	browseTimeout time.Duration,
) []*zeroconf.ServiceEntry {
	results := make([]*zeroconf.ServiceEntry, 0, 32)
	for _, s := range serviceTypes {
		ctx, cancel := context.WithTimeout(context.Background(), browseTimeout)
		entries := make(chan *zeroconf.ServiceEntry, 32)
		var opts []zeroconf.ClientOption
		opts = append(opts, zeroconf.SelectIPTraffic(zeroconf.IPv4))
		resolver, err := zeroconf.NewResolver(opts...)
		if err != nil {
			cancel()
			continue
		}
		go func() {
			for {
				select {
				case e, ok := <-entries:
					if !ok {
						return
					}
					if e != nil {
						results = append(results, e)
					}
				case <-ctx.Done():
					return
				}
			}
		}()
		if err := resolver.Browse(ctx, s, "local.", entries); err != nil {
			cancel()
			continue
		}
		<-ctx.Done()
		cancel()
	}
	return results
}

func waitForConnectService(pairingHost string, pairingPort int, waitTimeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	defer cancel()
	entries := make(chan *zeroconf.ServiceEntry, 32)
	var opts []zeroconf.ClientOption
	opts = append(opts, zeroconf.SelectIPTraffic(zeroconf.IPv4))
	resolver, err := zeroconf.NewResolver(opts...)
	if err != nil {
		return false
	}
	found := make(chan bool, 1)
	go func() {
		for {
			select {
			case e, ok := <-entries:
				if !ok {
					return
				}
				if e == nil {
					continue
				}
				cHost := resolveServiceHost(e)
				if cHost == "" {
					continue
				}
				if cHost == pairingHost && e.Port != pairingPort {
					if connectToDevice(cHost, e.Port) {
						found <- true
						return
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	if err := resolver.Browse(ctx, serviceConnect, "local.", entries); err != nil {
		return false
	}
	select {
	case <-found:
		return true
	case <-ctx.Done():
		return false
	}
}

func runQrPairAndConnect(deviceName string, password string, cfg Config) bool {
	for attempt := 0; attempt < cfg.Attempts; attempt++ {
		services := browseServices([]string{servicePairing}, cfg.QueryTimeout)
		var pairingEntry *zeroconf.ServiceEntry
		for _, entry := range services {
			if entry.Instance == deviceName {
				pairingEntry = entry
				break
			}
		}
		if pairingEntry != nil {
			pairingHost := resolveServiceHost(pairingEntry)
			if pairingHost == "" {
				time.Sleep(cfg.SleepBetween)
				continue
			}
			if !pairDevice(pairingHost, pairingEntry.Port, password) {
				time.Sleep(cfg.SleepBetween)
				continue
			}
			if waitForConnectService(pairingHost, pairingEntry.Port, cfg.ConnectWait) {
				return true
			}
		}
		time.Sleep(cfg.SleepBetween)
	}
	return false
}

func parseFlags() Config {
	name := flag.String("name", "", "Device name to advertise (default random ADB_WIFI_<id>)")
	password := flag.String("password", "", "Pairing password (default random)")
	attempts := flag.Int("attempts", defaultAttempts, "Max discovery attempts before giving up")
	timeout := flag.Duration("timeout", defaultQueryTimeout, "Per-query mDNS timeout")
	sleep := flag.Duration("sleep", defaultSleepBetween, "Sleep between discovery attempts")
	connectWait := flag.Duration("connect-wait", defaultConnectWait, "Max time to wait for connect service after pairing")
	flag.Parse()

	cfg := Config{
		DeviceName:   *name,
		Password:     *password,
		Attempts:     *attempts,
		QueryTimeout: *timeout,
		SleepBetween: *sleep,
		ConnectWait:  *connectWait,
	}
	return cfg
}

func main() {
	cfg := parseFlags()

	if cfg.DeviceName == "" {
		cfg.DeviceName = "ADB_WIFI_" + generateID()
	}
	if cfg.Password == "" {
		cfg.Password = generateID()
	}

	fmt.Println("Scan this QR in Android → Developer options → Wireless debugging → Pair device with QR code")
	showQR(cfg.DeviceName, cfg.Password)
	fmt.Println("\nScanning for ADB devices...")
	if ok := runQrPairAndConnect(cfg.DeviceName, cfg.Password, cfg); !ok {
		os.Exit(1)
	}
}
