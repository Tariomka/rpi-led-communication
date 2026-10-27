package component

import (
	"log/slog"
	"net"
	"net/netip"
	"runtime"
	"time"

	"github.com/Tariomka/rpi-led-communication/internal/common"
	"github.com/soypat/cyw43439"
	"github.com/soypat/seqs/eth/dhcp"
	"github.com/soypat/seqs/stacks"
)

const (
	maxRetries = 5
	queueSize  = 3

	// seqs advertises a receive window only when it sends a segment and drops the peer's
	// zero-window probes, so a connection whose window ever reaches zero stalls for good. The
	// receive buffer therefore holds the largest desktop message (tcp.HeaderSize + 4 + tcp.MaxChunk
	// = 2056 bytes) with room to spare; the desktop sends one upload chunk at a time.
	tcpRxBufferSize = uint16(4096)
	tcpTxBufferSize = uint16(1024)
	tcpConnections  = 2 // the desktop and a replacement while the old connection closes

	dhcpAttempts = 3
	dhcpTimeout  = 10 * time.Second
)

var ledStatus bool

type WirelessChip struct {
	device *cyw43439.Device // (CYW43439) WiFi + Bluetooth
	stack  *stacks.PortStack
	dhcp   *dhcp.ClientState

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
	if err := this.configureTcpStack(); err != nil {
		return err
	}

	go this.handlePackets()

	// Even if DHCP fails to provide an IP, connection is still usable
	// and static IP is used as a fallback
	return this.dhcpRequest(hostname, address)
}

func (this *WirelessChip) GetListener(listenPort uint16) (net.Listener, error) {
	if this.stack == nil {
		return nil, common.ErrNoInternetConnection
	}

	listener, err := stacks.NewTCPListener(this.stack, stacks.TCPListenerConfig{
		MaxConnections: tcpConnections,
		ConnTxBufSize:  tcpTxBufferSize,
		ConnRxBufSize:  tcpRxBufferSize,
	})
	if err != nil {
		return nil, err
	}

	if err = listener.StartListening(listenPort); err != nil {
		return nil, err
	}

	return listener, nil
}

// Addr is the IP address in use, invalid before Connect.
func (this *WirelessChip) Addr() netip.Addr {
	if this.stack == nil {
		return netip.Addr{}
	}
	return this.stack.Addr()
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

func (this *WirelessChip) configureTcpStack() error {
	mac, err := this.device.HardwareAddr6()
	if err != nil {
		return err
	}

	this.stack = stacks.NewPortStack(stacks.PortStackConfig{
		MAC:             mac,
		MaxOpenPortsUDP: 1,
		MaxOpenPortsTCP: 1,
		MTU:             cyw43439.MTU,
		Logger:          common.NewNoopLogger(),
	})
	this.device.RecvEthHandle(func(pkt []byte) { this.stack.RecvEth(pkt) })
	return nil
}

func (this *WirelessChip) dhcpRequest(hostname string, address netip.Addr) error {
	for attempt := 1; attempt <= dhcpAttempts; attempt++ {
		this.logger.Info("DHCP: requesting an address", "attempt", attempt)
		dhcpClient := stacks.NewDHCPClient(this.stack, dhcp.DefaultClientPort)
		err := dhcpClient.BeginRequest(stacks.DHCPRequestConfig{
			RequestedAddr: address,
			Xid:           uint32(time.Now().UnixNano()) | 1, // must be non-zero
			Hostname:      hostname,
		})
		if err != nil {
			return err
		}

		deadline := time.Now().Add(dhcpTimeout)
		for dhcpClient.State() != dhcp.StateBound && time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
		}
		if dhcpClient.State() == dhcp.StateBound {
			this.dhcpBound(dhcpClient)
			return nil
		}

		this.logger.Warn("DHCP: no answer", "timeout", dhcpTimeout.String())
		dhcpClient.Abort()
		time.Sleep(200 * time.Millisecond) // lets the packet loop drop the aborted client
		this.stack.CloseUDP(dhcp.DefaultClientPort)
	}

	if !address.IsValid() {
		this.logger.Error("DHCP: failed and no static IP is configured")
		return common.ErrDhcpRequestFailed
	}

	this.logger.Warn("DHCP: failed, using the static IP", "ip", address.String())
	this.stack.SetAddr(address)
	return nil
}

func (this *WirelessChip) dhcpBound(dhcpClient *stacks.DHCPClient) {
	var primaryDNS netip.Addr
	dnsServers := dhcpClient.DNSServers()
	if len(dnsServers) > 0 {
		primaryDNS = dnsServers[0]
	}
	ip := dhcpClient.Offer()
	this.logger.Info("DHCP: bound", "ip", ip.String(), "router", dhcpClient.Router().String())
	this.logger.Debug(
		"DHCP complete",
		slog.Uint64("cidrbits", uint64(dhcpClient.CIDRBits())),
		slog.String("ourIP", ip.String()),
		slog.String("dns", primaryDNS.String()),
		slog.String("broadcast", dhcpClient.BroadcastAddr().String()),
		slog.String("gateway", dhcpClient.Gateway().String()),
		slog.String("router", dhcpClient.Router().String()),
		slog.String("dhcp", dhcpClient.DHCPServer().String()),
		slog.String("hostname", string(dhcpClient.Hostname())),
		slog.Duration("lease", dhcpClient.IPLeaseTime()),
		slog.Duration("renewal", dhcpClient.RenewalTime()),
		slog.Duration("rebinding", dhcpClient.RebindingTime()),
	)

	this.stack.SetAddr(ip) // It's important to set the IP address after DHCP completes.
}

// This was taken from the example and refactored
func (this *WirelessChip) handlePackets() {
	defer Recover("Wi-Fi packet loop")
	queue := newPacketQueue()
	for {
		runtime.Gosched() // The scheduler is cooperative: let the main loop run between packets.
		stallRx := !this.isPacketAvailable()
		queue.enqueue(this.stack.HandleEth)

		stallTx := queue.isQueueEmpty()
		if stallTx {
			if stallRx {
				time.Sleep(50 * time.Millisecond) // Avoid busy waiting when both Rx and Tx stall.
			}
			continue
		}

		queue.sendQueued(this.device.SendEth)
	}
}

func (this *WirelessChip) isPacketAvailable() bool {
	gotPacket, _ := this.device.PollOne()
	return gotPacket
}

type packetInstance struct {
	bufferLen int
	buffer    [cyw43439.MTU]byte
	retries   int
}

func newPacketInstance() packetInstance {
	return packetInstance{
		bufferLen: 0,
		buffer:    [cyw43439.MTU]byte{},
		retries:   0,
	}
}

type packetQueue [queueSize]packetInstance

func newPacketQueue() *packetQueue {
	return &packetQueue{}
}

func (this *packetQueue) clean(index int) {
	this[index] = packetInstance{}
}

func (this *packetQueue) enqueue(handleEth func([]byte) (int, error)) {
	for index := range this {
		if this[index].retries != 0 {
			continue // Packet currently queued for retransmission.
		}
		var err error
		buf := this[index].buffer[:]
		this[index].bufferLen, err = handleEth(buf)
		if err != nil {
			this[index].bufferLen = 0
			continue
		}
		if this[index].bufferLen == 0 {
			break
		}
	}
}

func (this *packetQueue) sendQueued(sendEth func([]byte) error) {
	for index := range this {
		n := this[index].bufferLen
		if n <= 0 {
			continue
		}
		err := sendEth(this[index].buffer[:n])
		if err != nil {
			// Queue packet for retransmission.
			this[index].retries++
			if this[index].retries > maxRetries {
				this.clean(index)
			}
		} else {
			this.clean(index)
		}
	}
}

func (this *packetQueue) isQueueEmpty() bool {
	for index := range this {
		if this[index].bufferLen > 0 {
			return false
		}
	}
	return true
}
