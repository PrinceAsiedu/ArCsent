package system

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ipsix/arcsent/internal/scanner"
)

type NetworkListeners struct {
	tcpPath string
	udpPath string
}

func (n *NetworkListeners) Name() string { return "system.network_listeners" }

func (n *NetworkListeners) Init(config map[string]interface{}) error {
	n.tcpPath = "/proc/net/tcp"
	n.udpPath = "/proc/net/udp"
	if v, ok := config["tcp_path"].(string); ok && v != "" {
		n.tcpPath = v
	}
	if v, ok := config["udp_path"].(string); ok && v != "" {
		n.udpPath = v
	}
	return nil
}

func (n *NetworkListeners) Run(_ context.Context) (*scanner.Result, error) {
	tcp, err := os.ReadFile(n.tcpPath)
	if err != nil {
		return nil, fmt.Errorf("read tcp: %w", err)
	}
	udp, err := os.ReadFile(n.udpPath)
	if err != nil {
		return nil, fmt.Errorf("read udp: %w", err)
	}

	result := &scanner.Result{
		ScannerName: n.Name(),
		Status:      scanner.StatusSuccess,
		Metadata: map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"tcp_count": countListeners(string(tcp)),
			"udp_count": countListeners(string(udp)),
		},
	}
	return result, nil
}

func (n *NetworkListeners) Halt(_ context.Context) error { return nil }

func countListeners(data string) int {
	lines := strings.Split(data, "\n")
	count := 0
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		// TCP state 0A = LISTEN; UDP uses 07 for listening sockets.
		state := fields[3]
		if state == "0A" || state == "07" {
			count++
		}
	}
	return count
}
