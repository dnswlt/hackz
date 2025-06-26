package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"time"

	"golang.org/x/net/ipv4"
)

// A simple program that can act as a server or client for receiving or sending
// multicast packets. As a server, the program responds to multicast "discovery"
// packets to announce its presence on the network interface.
// As a client, the program sends a multicast packet to the broadcast address
// of the network interface to identify other machines on the same network.

var (
	ifaceFlag          = flag.String("iface", "", "Network interface to use.")
	serverModeFlag     = flag.Bool("server", false, "Run as server.")
	multicastAddrFlag  = flag.String("multicast-addr", "239.36.36.36", "Multicast address to use.")
	portFlag           = flag.Int("port", 12345, "Port to use for multicast.")
	timeoutFlag        = flag.Duration("timeout", 5*time.Second, "Timeout for receiving multicast responses.")
	customMessageFlag  = flag.String("custom-message", "", "Custom message to send in the status message.")
	listInterfacesFlag = flag.Bool("list-ifaces", false, "Print all interfaces and exit.")
	debugFlag          = flag.Bool("debug", false, "Enable debug logging")
)

// MessageType is the type of the multicast message (the tag in the tagged union that Message is).
type MessageType string

const (
	// MessageTypeDiscovery is the type of the multicast message sent
	// by the client to discover other machines on the network.
	MessageTypeDiscovery = MessageType("discovery")
	// MessageTypeStatus is the type of the multicast message sent
	// by the servers to respond to a multicast discovery packet.
	MessageTypeStatus = MessageType("status")
)

// Message is the type of messages sent by the multicast clients and servers.
type Message struct {
	Type    MessageType     `json:"type"`
	Message json.RawMessage `json:"message"`
}

// StatusMessage is the type of the multicast message sent by the servers to
// respond to a multicast discovery packet.
type StatusMessage struct {
	Timestamp     time.Time `json:"timestamp"`
	Hostname      string    `json:"hostname"`
	MAC           string    `json:"mac"`
	IPv4          string    `json:"ipv4"`
	OS            string    `json:"os"`
	RunID         string    `json:"runID"`
	UptimeSeconds float64   `json:"uptimeSeconds"`
	CustomMessage string    `json:"customMessage"`
}

// DiscoveryMessage is the type of the multicast message sent by the client
// to discover other machines on the network.
type DiscoveryMessage struct {
	Timestamp time.Time `json:"timestamp"`
}

func debugLog(format string, v ...any) {
	if !*debugFlag {
		return
	}
	log.Printf(format, v...)
}

func printInterface(iface net.Interface) {
	fmt.Printf("Name: %q\n", iface.Name)
	fmt.Printf("  Flags: %s\n", iface.Flags.String())
	fmt.Printf("  MAC: %s\n", iface.HardwareAddr.String())
	addrs, err := iface.Addrs()
	if err == nil {
		for _, addr := range addrs {
			fmt.Printf("  Addr: %s\n", addr.String())
		}
	}
	fmt.Println()
}

func listIPv4Interfaces() []net.Interface {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Fatalf("Failed to list interfaces: %v", err)
	}

	var result []net.Interface
	for _, iface := range ifaces {
		// Skip interfaces that are down or loopback
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		hasIPv4 := false
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.To4() != nil {
				hasIPv4 = true
				break
			}
		}

		if !hasIPv4 {
			continue
		}

		result = append(result, iface)
	}

	return result
}

func newMessage(msgType MessageType, msg any) (*Message, error) {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return &Message{
		Type:    msgType,
		Message: msgBytes,
	}, nil
}

func runServer(ifaceName string, multicastAddr string, port int, customMessage string) {
	debugLog("Starting server...\n")
	started := time.Now()

	// To receive multicast packets, we bind the UDP socket to 0.0.0.0 (INADDR_ANY) and explicitly
	// join the multicast group on a specific interface. This is required on Windows, where binding
	// to a unicast or multicast address prevents successful group membership (setsockopt fails).
	// Binding to 0.0.0.0 ensures the socket can receive datagrams sent to the multicast group.
	// This approach is portable and works reliably across Windows, Linux, and macOS.
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4zero, // 0.0.0.0
		Port: port,
	})
	if err != nil {
		log.Fatalf("Failed to listen on UDP: %v", err)
	}
	defer conn.Close()

	// Now join the multicast group only on the given interface.
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		log.Fatalf("Failed to get interface %q: %v", ifaceName, err)
	}
	pc := ipv4.NewPacketConn(conn)
	err = pc.JoinGroup(iface, &net.UDPAddr{
		IP: net.ParseIP(multicastAddr),
	})
	if err != nil {
		log.Fatalf("Failed to join multicast group: %v", err)
	}

	debugLog("Listening on interface %q on %v for multicast address %v \n",
		iface.Name, conn.LocalAddr().String(), multicastAddr)

	// Get data to send in the status message.
	hostname := getHostname()
	localAddr, err := getLocalIPv4Addr(iface)
	if err != nil {
		log.Fatalf("Failed to get local address: %v", err)
	}
	runID := getUID(8)

	// Serve forever.
	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			debugLog("Error reading from UDP: %v\n", err)
			continue
		}
		debugLog("Received %d bytes from %v: %s\n", n, remoteAddr, string(buf[:n]))
		var message Message
		if err := json.Unmarshal(buf[:n], &message); err != nil {
			debugLog("Error unmarshalling message: %v\n", err)
			continue
		}
		if message.Type == MessageTypeDiscovery {
			// Ignore content of the message.
			// Send a "status" message response back to the sender.
			response, err := newMessage(MessageTypeStatus, StatusMessage{
				Timestamp:     time.Now(),
				Hostname:      hostname,
				MAC:           iface.HardwareAddr.String(),
				IPv4:          localAddr.IP.String(),
				OS:            fmt.Sprintf("%v/%v", runtime.GOOS, runtime.GOARCH),
				RunID:         runID,
				UptimeSeconds: time.Since(started).Seconds(),
				CustomMessage: customMessage,
			})
			if err != nil {
				log.Fatalf("Error creating response: %v\n", err)
				continue
			}
			responseBytes, err := json.Marshal(response)
			if err != nil {
				log.Fatalf("Error marshalling response: %v\n", err)
				continue
			}
			_, err = conn.WriteTo(responseBytes, remoteAddr)
			if err != nil {
				debugLog("Error writing to UDP: %v\n", err)
				continue
			}
		} else {
			debugLog("Received unknown message type %s, ignoring...", message.Type)
		}
	}
}

func runClient(ifaceName string, multicastAddr string, port int, timeout time.Duration) {
	debugLog("Starting client...\n")
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", multicastAddr, port))
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v", err)
	}
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		log.Fatalf("Failed to get interface %q: %v", ifaceName, err)
	}
	localAddr, err := getLocalIPv4Addr(iface)
	if err != nil {
		log.Fatalf("Failed to get local address: %v", err)
	}
	conn, err := net.ListenPacket("udp", localAddr.String())
	if err != nil {
		log.Fatalf("Failed to dial multicast UDP: %v", err)
	}

	debugLog("Listening on %v\n", conn.LocalAddr().String())
	defer conn.Close()
	debugLog("Sending to %v\n", addr)

	message, err := newMessage(MessageTypeDiscovery, DiscoveryMessage{Timestamp: time.Now()})
	if err != nil {
		log.Fatalf("Failed to create discovery message: %v", err)
	}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Fatalf("Failed to marshal discovery message: %v", err)
	}
	_, err = conn.WriteTo(messageBytes, addr)
	if err != nil {
		log.Fatalf("Failed to write to UDP: %v", err)
	}

	deadline := time.Now().Add(timeout)
	conn.SetReadDeadline(deadline)
	buf := make([]byte, 1024)
	for time.Now().Before(deadline) {
		n, remoteAddr, err := conn.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				debugLog("Timeout while reading from UDP.\n")
			} else {
				debugLog("Error reading from UDP: %v\n", err)
			}
			break
		}
		var message Message
		if err := json.Unmarshal(buf[:n], &message); err != nil {
			debugLog("Error unmarshalling message: %v\n", err)
			continue
		}
		if message.Type == MessageTypeStatus {
			var statusMessage StatusMessage
			if err := json.Unmarshal(message.Message, &statusMessage); err != nil {
				debugLog("Error unmarshalling status message: %v\n", err)
				continue
			}
			formatted, err := json.MarshalIndent(statusMessage, "", " ")
			if err != nil {
				log.Fatalf("Error marshalling status message: %v\n", err)
			}
			fmt.Printf("Status from %v:\n%s\n", remoteAddr, string(formatted))
		} else {
			debugLog("Received unknown message type %s, ignoring...", message.Type)
		}
	}
}

func getLocalIPv4Addr(iface *net.Interface) (*net.UDPAddr, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get addresses for interface %s: %w", iface.Name, err)
	}
	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		default:
			continue
		}
		ip = ip.To4()
		if ip == nil {
			continue // Not an IPv4 address
		}
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue // Skip loopback or link-local
		}
		return &net.UDPAddr{IP: ip}, nil
	}
	return nil, fmt.Errorf("no suitable IPv4 address found for interface %s", iface.Name)
}

func getHostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}

func getUID(nBytes int) string {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("Failed to generate ID: %v", err)
	}
	return fmt.Sprintf("%x", b)
}

func main() {
	flag.Parse()

	ifaceName := *ifaceFlag
	serverMode := *serverModeFlag
	multicastAddr := *multicastAddrFlag
	port := *portFlag
	timeout := *timeoutFlag
	customMessage := *customMessageFlag

	if *listInterfacesFlag {
		for _, iface := range listIPv4Interfaces() {
			printInterface(iface)
		}
		os.Exit(0)
	}

	if ifaceName == "" {
		ifaces := listIPv4Interfaces()
		if len(ifaces) == 1 {
			ifaceName = ifaces[0].Name
		} else {
			log.Fatalf("No interface name specified and no unique interface found. Use --list-ifaces to list them.")
		}
	}

	debugLog("Using interface: %s\n", ifaceName)
	debugLog("Multicast address: %s\n", multicastAddr)
	debugLog("Port: %d\n", port)
	debugLog("Timeout: %v\n", timeout)

	if serverMode {
		runServer(ifaceName, multicastAddr, port, customMessage)
	} else {
		runClient(ifaceName, multicastAddr, port, timeout)
	}
}
