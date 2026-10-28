package component

import (
	"log/slog"
	"net"
	"net/netip"
	"time"

	"github.com/Tariomka/rpi-led-communication/internal/common"
	"github.com/soypat/cyw43439"
	"github.com/soypat/lneto"
	"github.com/soypat/lneto/tcp"
	"github.com/soypat/lneto/tcp/rto"
	"github.com/soypat/lneto/x/xnet"
)

const (
	dhcpAttempts = 3
	dhcpTimeout  = 10 * time.Second

	packetIdleSleep = 5 * time.Millisecond
)

var ledStatus bool

var start = time.Now()

func nanotime() int64 { return int64(time.Since(start)) }

// backoff is the wait between polls while a TCP read, write or stack request has nothing to do;
// capped low so desktop messages are not delayed.
func backoff(consecutive uint) time.Duration {
	if consecutive == 0 {
		return lneto.BackoffFlagGosched
	}

	return time.Millisecond
}

type WirelessChip struct {
	device *cyw43439.Device // (CYW43439) WiFi + Bluetooth
	stack  xnet.StackAsync
	ready  bool // stack configured

	logger *slog.Logger
}

func NewWirelessChip(logger *slog.Logger) *WirelessChip {
	if logger == nil {
		logger = common.NewNoopLogger()
	}

	device := cyw43439.NewPicoWDevice()
	MustInitDevice(device)
	logger.Info("Wi-Fi: chip initialized")

	return &WirelessChip{
		device: device,
		logger: logger, // set logger to Wifi sender later, to send info to PC/phone
	}
}

func (this *WirelessChip) Connect(ssid, pass, staticIp, hostname string) error {
	if ssid == "" {
		return common.ErrNoWifiConfig
	}
	address, err := common.GetAddress(staticIp)
	if err != nil {
		return err
	}

	this.join(ssid, pass)
	if err := this.configureStack(hostname); err != nil {
		return err
	}

	go this.handlePackets()

	// Even if DHCP fails to provide an IP, connection is still usable
	// and static IP is used as a fallback
	return this.dhcpRequest(address)
}

// GetListener starts accepting desktop connections on listenPort.
func (this *WirelessChip) GetListener(listenPort uint16) (*Listener, error) {
	if !this.ready {
		return nil, common.ErrNoInternetConnection
	}

	pool, err := this.stack.NewTCPPool(xnet.TCPPoolConfig{
		PoolSize:           2,
		QueueSize:          3,
		TxBufSize:          1024,
		RxBufSize:          4096,
		EstablishedTimeout: 5 * time.Second,
		ClosingTimeout:     5 * time.Second,
		NewBackoff:         func() lneto.BackoffStrategy { return backoff },
		NewPolicy:          newRetransmitTimer,
	})
	if err != nil {
		return nil, err
	}

	listener := &Listener{pool: pool}
	if err := listener.tcp.Reset(listenPort, pool); err != nil {
		return nil, err
	}
	if err := this.stack.RegisterListenerTCP(&listener.tcp); err != nil {
		return nil, err
	}
	return listener, nil
}

// newRetransmitTimer gives each connection the RFC 6298 retransmission timer; without a policy
// lneto never retransmits.
func newRetransmitTimer() tcp.Policy {
	timer := new(rto.Timer)
	timer.Configure(nanotime)
	return timer
}

// Addr is the IP address in use, invalid before Connect.
func (this *WirelessChip) Addr() netip.Addr {
	if !this.ready {
		return netip.Addr{}
	}
	return netip.AddrFrom4(this.stack.Addr4())
}

func (this *WirelessChip) Blink() {
	this.device.GPIOSet(0, !ledStatus)
	time.Sleep(125 * time.Millisecond)
	this.device.GPIOSet(0, ledStatus)
	time.Sleep(125 * time.Millisecond)
}

func (this *WirelessChip) TurnLed(on bool) {
	ledStatus = on
	this.device.GPIOSet(0, ledStatus)
}

// join retries until the network accepts the Pico; Wi-Fi setup runs beside the main loop.
func (this *WirelessChip) join(ssid, password string) {
	for attempt := 1; ; attempt++ {
		this.logger.Info("Wi-Fi: joining", "ssid", ssid, "attempt", attempt)
		err := this.device.Join(ssid, cyw43439.JoinOptions{Passphrase: password})
		if err == nil {
			mac, _ := this.device.HardwareAddr6()
			this.logger.Info("Wi-Fi: joined", "mac", net.HardwareAddr(mac[:]).String())
			return
		}

		this.logger.Error("Wi-Fi: join failed", "error", err.Error(), "retry in", "5s")
		time.Sleep(5 * time.Second)
	}
}

func (this *WirelessChip) configureStack(hostname string) error {
	mac, err := this.device.HardwareAddr6()
	if err != nil {
		return err
	}
	if hostname == "" {
		hostname = "pico" // the stack rejects an empty hostname
	}

	err = this.stack.Reset(xnet.StackConfig{
		HardwareAddress:   mac,
		Hostname:          hostname,
		RandSeed:          nanotime() | 1, // must be non-zero
		MaxActiveTCPPorts: 1,
		MaxActiveUDPPorts: 1,
		MTU:               cyw43439.MTU,
		PassivePeers:      4, // MACs learned from incoming packets, used to answer desktops
		ICMPQueueLimit:    2,
		Nanotime:          nanotime,
	})
	if err != nil {
		return err
	}
	this.ready = true
	this.device.RecvEthHandle(func(pkt []byte) { this.stack.IngressEthernet(pkt) })
	return this.stack.EnableICMP(true) // answers ping, handy to check the link
}

func (this *WirelessChip) dhcpRequest(address netip.Addr) error {
	var requested [4]byte
	if address.IsValid() && address.Is4() {
		requested = address.As4()
	}

	this.logger.Info("DHCP: requesting an address", "max attempts", dhcpAttempts)
	retrying := this.stack.StackRetrying(backoff)
	results, err := retrying.DoDHCPv4(requested, dhcpTimeout, dhcpAttempts)
	if err == nil {
		this.dhcpBound(retrying, results)
		return nil
	}

	this.logger.Warn("DHCP: no answer", "error", err.Error())
	if !address.IsValid() || !address.Is4() {
		this.logger.Error("DHCP: failed and no static IPv4 is configured")
		return common.ErrDhcpRequestFailed
	}

	this.logger.Warn("DHCP: failed, using the static IP", "ip", address.String())
	this.stack.SetSubnet4(address.As4(), 24)
	return this.stack.SetAddr4(address.As4())
}

func (this *WirelessChip) dhcpBound(retrying xnet.StackRetrying, results *xnet.DHCPResults) {
	this.logger.Info("DHCP: bound",
		"ip", netip.AddrFrom4(results.AssignedAddr4).String(),
		"router", results.Router.String(),
		"subnet", results.Subnet.String(),
		"lease", time.Duration(results.TLease)*time.Second)
	if err := this.stack.AssimilateDHCPResults(results); err != nil {
		this.logger.Error("DHCP: results rejected", "error", err.Error())
		return
	}

	// Only traffic leaving the subnet needs the router's MAC; desktops are on the LAN.
	gateway, err := retrying.DoResolveHardwareAddress6(results.Router, 500*time.Millisecond, 4)
	if err != nil {
		this.logger.Warn("ARP: router not resolved", "error", err.Error())
		return
	}
	this.stack.SetGatewayHardwareAddr(gateway)
}

func (this *WirelessChip) handlePackets() {
	defer Recover("Wi-Fi packet loop")
	buffer := make([]byte, cyw43439.MaxFrameSize)
	for {
		received, _ := this.device.PollOne()
		n, err := this.stack.EgressEthernet(buffer)
		if err != nil {
			this.logger.Debug("Wi-Fi: egress failed", "error", err.Error())
		}
		if n > 0 {
			if err := this.device.SendEth(buffer[:n]); err != nil {
				this.logger.Debug("Wi-Fi: send failed", "error", err.Error())
			}
		}
		if !received && n == 0 {
			time.Sleep(packetIdleSleep)
		}
	}
}

// Listener accepts desktop connections; connections are recycled by the pool once closed.
type Listener struct {
	tcp  tcp.Listener
	pool *xnet.TCPPool
}

// Accept blocks until a connection is established or the listener is closed.
func (this *Listener) Accept() (tcp.ConnHandle, error) {
	for this.tcp.NumberOfReadyToAccept() == 0 {
		if this.tcp.LocalPort() == 0 {
			return tcp.ConnHandle{}, net.ErrClosed
		}
		this.pool.CheckTimeouts()
		time.Sleep(packetIdleSleep)
	}
	return this.tcp.TryAccept()
}

func (this *Listener) Close() error {
	return this.tcp.Close()
}
