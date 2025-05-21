package component

import (
	"log/slog"
	"net"
	"net/netip"
	"strings"
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
	mustInitDevice(device)

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

	address, err := getAddress(staticIp)
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

// Haven't checked what this does, it is taken from the example
func (this *WirelessChip) handlePackets() {
	var queue [queueSize][cyw43439.MTU]byte
	var lenBuf [queueSize]int
	var retries [queueSize]int
	markSent := func(i int) {
		queue[i] = [cyw43439.MTU]byte{} // Not really necessary.
		lenBuf[i] = 0
		retries[i] = 0
	}
	for {
		stallRx := true
		// Poll for incoming packets.
		for i := 0; i < 1; i++ {
			gotPacket, _ := this.device.PollOne()
			if !gotPacket {
				break
			}
			stallRx = false
		}

		// Queue packets to be sent.
		for i := range queue {
			if retries[i] != 0 {
				continue // Packet currently queued for retransmission.
			}
			var err error
			buf := queue[i][:]
			lenBuf[i], err = this.stack.HandleEth(buf[:])
			if err != nil {
				lenBuf[i] = 0
				continue
			}
			if lenBuf[i] == 0 {
				break
			}
		}
		stallTx := lenBuf == [queueSize]int{}
		if stallTx {
			if stallRx {
				// Avoid busy waiting when both Rx and Tx stall.
				time.Sleep(51 * time.Millisecond)
			}
			continue
		}

		// Send queued packets.
		for i := range queue {
			n := lenBuf[i]
			if n <= 0 {
				continue
			}
			err := this.device.SendEth(queue[i][:n])
			if err != nil {
				// Queue packet for retransmission.
				retries[i]++
				if retries[i] > maxRetries {
					markSent(i)
				}
			} else {
				markSent(i)
			}
		}
	}
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

func mustInitDevice(device *cyw43439.Device) {
	config := cyw43439.DefaultWifiBluetoothConfig()
	config.Logger = common.NewNoopLogger()
	if err := device.Init(config); err != nil {
		panic("Failed to initialize Pico W wireless interface: " + err.Error())
	}
}

func getAddress(staticIp string) (netip.Addr, error) {
	var address netip.Addr

	if strings.TrimSpace(staticIp) == "" {
		return address, nil
	}

	return netip.ParseAddr(staticIp)
}
