package system

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNetworkListeners(t *testing.T) {
	dir := t.TempDir()
	tcpPath := filepath.Join(dir, "tcp")
	udpPath := filepath.Join(dir, "udp")

	tcpData := "sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt\n" +
		"0: 00000000:0016 00000000:0000 0A 00000000:00000000 00:00000000 00000000 100 0 0 10 0\n"
	udpData := "sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt\n" +
		"0: 00000000:0035 00000000:0000 07 00000000:00000000 00:00000000 00000000 100 0 0 10 0\n"

	if err := os.WriteFile(tcpPath, []byte(tcpData), 0o600); err != nil {
		t.Fatalf("write tcp: %v", err)
	}
	if err := os.WriteFile(udpPath, []byte(udpData), 0o600); err != nil {
		t.Fatalf("write udp: %v", err)
	}

	plugin := &NetworkListeners{}
	if err := plugin.Init(map[string]interface{}{"tcp_path": tcpPath, "udp_path": udpPath}); err != nil {
		t.Fatalf("init: %v", err)
	}
	result, err := plugin.Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Metadata["tcp_count"] != 1.0 && result.Metadata["tcp_count"] != 1 {
		t.Fatalf("expected tcp_count 1, got %v", result.Metadata["tcp_count"])
	}
}
