package controller

import (
	"errors"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/soypat/cyw43439"
	"github.com/soypat/seqs/eth/dhcp"
	"github.com/soypat/seqs/stacks"
)

const maxRetries = 5

type Board interface {
	Connect(ssid, pass, staticIp string) error // Connects to Wi-Fi. Returns an error if connection process fails.
	GetListener(listenPort uint16) (net.Listener, error)
}

type PicoW struct {
	WirelessChip *cyw43439.Device // (CYW43439) WiFi + Bluetooth
	stack        *stacks.PortStack
	dhcp         *dhcp.ClientState

	logger   *slog.Logger
	hostname string
}

func NewPicoW(hostname string, logger *slog.Logger) Board {
	if logger == nil {
		// Default empty logger
		logger = slog.New(slog.NewTextHandler(
			io.Discard,
			&slog.HandlerOptions{Level: slog.Level(64)}))
	}
	if strings.TrimSpace(hostname) == "" {
		hostname = "PicoW"
	}

	board := PicoW{
		WirelessChip: cyw43439.NewPicoWDevice(),
		logger:       logger, // set logger to Wifi sender later, to send info to PC/phone
		hostname:     hostname,
	}

	config := cyw43439.DefaultWifiBluetoothConfig()
	config.Logger = logger
	if err := board.WirelessChip.Init(config); err != nil {
		panic("Failed to initialize Pico W wireless interface: " + err.Error())
	}

	return &board
}

func (pw *PicoW) Connect(ssid, pass, staticIp string) error {
	var err error
	var requestedAddress netip.Addr

	for i := 0; i < maxRetries; i++ {
		err = pw.WirelessChip.JoinWPA2(ssid, pass)
		if err == nil {
			break
		}
		pw.logger.Error("wifi join failed", "err", err.Error())
		time.Sleep(5 * time.Second)
	}

	mac, err := pw.WirelessChip.HardwareAddr6()
	if err != nil {
		return err
	}

	pw.stack = stacks.NewPortStack(stacks.PortStackConfig{
		MAC:             mac,
		MaxOpenPortsUDP: 1,
		MaxOpenPortsTCP: 1,
		MTU:             cyw43439.MTU,
		Logger:          pw.logger,
	})

	pw.WirelessChip.RecvEthHandle(pw.stack.RecvEth)

	go pw.handlePackets()

	if strings.TrimSpace(staticIp) == "" {
		return nil
	}

	requestedAddress, err = netip.ParseAddr(staticIp)
	if err != nil {
		return err
	}

	pw.stack.SetAddr(requestedAddress)
	return nil
}

func (pw *PicoW) DHCPRequest(staticIp string) error {
	var err error
	var requestedAddress netip.Addr

	if strings.TrimSpace(staticIp) != "" {
		requestedAddress, err = netip.ParseAddr(staticIp)
		if err != nil {
			return err
		}
	}

	// Perform DHCP request.
	dhcpClient := stacks.NewDHCPClient(pw.stack, dhcp.DefaultClientPort)
	dhcpConfig := stacks.DHCPRequestConfig{
		RequestedAddr: requestedAddress,
		Xid:           uint32(time.Now().Nanosecond()),
		Hostname:      pw.hostname,
	}
	if err = dhcpClient.BeginRequest(dhcpConfig); err != nil {
		return err
	}
	for i := 0; i < maxRetries && dhcpClient.State() != dhcp.StateBound; i++ {
		time.Sleep(time.Second / 2)
		if i > 15 {
			if !requestedAddress.IsValid() {
				return errors.New("DHCP did not complete and no static IP was requested")
			}
			pw.stack.SetAddr(requestedAddress)
			return nil
		}
	}
	var primaryDNS netip.Addr
	dnsServers := dhcpClient.DNSServers()
	if len(dnsServers) > 0 {
		primaryDNS = dnsServers[0]
	}
	ip := dhcpClient.Offer()
	pw.logger.Info("DHCP complete",
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

	pw.stack.SetAddr(ip) // It's important to set the IP address after DHCP completes.
	return nil
}

func (pw *PicoW) GetListener(listenPort uint16) (net.Listener, error) {
	tcpbufsize := uint16(512) // MTU - ethhdr - iphdr - tcphdr

	if pw.stack == nil {
		return nil, errors.New("did not connect to internet")
	}
	listener, err := stacks.NewTCPListener(pw.stack, stacks.TCPListenerConfig{
		MaxConnections: 3,
		ConnTxBufSize:  tcpbufsize,
		ConnRxBufSize:  tcpbufsize,
	})
	if err != nil {
		return nil, err
	}
	err = listener.StartListening(listenPort)
	if err != nil {
		return nil, err
	}

	return listener, nil
}

func (pw *PicoW) handlePackets() {
	// Maximum number of packets to queue before sending them.
	const (
		queueSize                = 3
		maxRetriesBeforeDropping = 3
	)
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
			gotPacket, err := pw.WirelessChip.PollOne()
			if err != nil {
				println("poll error:", err.Error())
			}
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
			lenBuf[i], err = pw.stack.HandleEth(buf[:])
			if err != nil {
				println("stack error n(should be 0)=", lenBuf[i], "err=", err.Error())
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
			err := pw.WirelessChip.SendEth(queue[i][:n])
			if err != nil {
				// Queue packet for retransmission.
				retries[i]++
				if retries[i] > maxRetriesBeforeDropping {
					markSent(i)
					println("dropped outgoing packet:", err.Error())
				}
			} else {
				markSent(i)
			}
		}
	}
}
