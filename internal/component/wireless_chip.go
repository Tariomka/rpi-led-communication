package component

import (
	"log/slog"
	"net"
	"net/netip"
	"time"

	"github.com/Tariomka/rpi-led-communication/internal/common"
	"github.com/soypat/cyw43439"
	"github.com/soypat/seqs/eth/dhcp"
	"github.com/soypat/seqs/stacks"
)

const (
	maxRetries    = 5
	queueSize     = 3
	tcpBufferSize = uint16(512) // MTU - ethhdr - iphdr - tcphdr
)

var ledStatus bool

type WirelessChip struct {
	device *cyw43439.Device // (CYW43439) WiFi + Bluetooth
	stack  *stacks.PortStack
	dhcp   *dhcp.ClientState

	logger *slog.Logger
}

func NewWirelessChip(logger *slog.Logger) *WirelessChip {
	device := cyw43439.NewPicoWDevice()
	MustInitDevice(device)

	if logger == nil {
		logger = common.NewNoopLogger()
	}
	return &WirelessChip{
		device: device,
		logger: logger, // set logger to Wifi sender later, to send info to PC/phone
	}
}

func (this *WirelessChip) Connect(ssid, pass, staticIp, hostname string) error {
	if err := this.joinWPA2(ssid, pass); err != nil {
		return err
	}

	if err := this.configureTcpStack(); err != nil {
		return err
	}

	go this.handlePackets()

	address, err := common.GetAddress(staticIp)
	if err != nil {
		return err
	}

	// Even if DHCP fails to provide an IP, connection is still usable
	// and static IP is used as a fallback
	return this.dhcpRequest(hostname, address)
}

func (this *WirelessChip) GetListener(listenPort uint16) (net.Listener, error) {
	if this.stack == nil {
		return nil, common.ErrNoInternetConnection
	}

	listener, err := stacks.NewTCPListener(this.stack, stacks.TCPListenerConfig{
		MaxConnections: 3,
		ConnTxBufSize:  tcpBufferSize,
		ConnRxBufSize:  tcpBufferSize,
	})
	if err != nil {
		return nil, err
	}

	if err = listener.StartListening(listenPort); err != nil {
		return nil, err
	}

	return listener, nil
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

func (this *WirelessChip) joinWPA2(ssid, password string) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = this.device.JoinWPA2(ssid, password)
		if err == nil {
			return nil
		}

		this.logger.Error("Wifi join failed", "err", err.Error())
		time.Sleep(5 * time.Second)
	}

	return err
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
	this.device.RecvEthHandle(this.stack.RecvEth)
	return nil
}

func (this *WirelessChip) dhcpRequest(hostname string, address netip.Addr) error {
	dhcpClient := stacks.NewDHCPClient(this.stack, dhcp.DefaultClientPort)
	dhcpConfig := stacks.DHCPRequestConfig{
		RequestedAddr: address,
		Xid:           uint32(time.Now().Nanosecond()),
		Hostname:      hostname,
	}
	if err := dhcpClient.BeginRequest(dhcpConfig); err != nil {
		return err
	}

	for i := 0; i < maxRetries && dhcpClient.State() != dhcp.StateBound; i++ {
		this.logger.Debug("DHCP ongoing...")
		time.Sleep(time.Second / 2)
		if i > 15 {
			if !address.IsValid() {
				this.logger.Error("DHCP did not complete and no static IP was requested")
				return common.ErrDhcpRequestFailed
			}

			this.logger.Warn("DHCP did not complete, falling back to static IP")
			this.stack.SetAddr(address)
			return nil
		}
	}

	var primaryDNS netip.Addr
	dnsServers := dhcpClient.DNSServers()
	if len(dnsServers) > 0 {
		primaryDNS = dnsServers[0]
	}
	ip := dhcpClient.Offer()
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

	if !ip.Is4() {
		this.logger.Warn(
			"DHCP did not provide a valid IP, falling back to static IP",
			"static IP", address.String())
		this.stack.SetAddr(address)
		return nil
	}

	this.stack.SetAddr(ip) // It's important to set the IP address after DHCP completes.
	return nil
}

// This was taken from the example and refactored
func (this *WirelessChip) handlePackets() {
	queue := newPacketQueue()
	for {
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
