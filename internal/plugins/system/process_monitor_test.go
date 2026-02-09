package system

import "testing"

func TestProcessMonitorInit(t *testing.T) {
	pm := &ProcessMonitor{}
	err := pm.Init(map[string]interface{}{
		"whitelist_prefixes": []interface{}{"/usr/bin"},
	})
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if len(pm.whitelistPrefixes) != 1 {
		t.Fatalf("expected whitelist prefix to be set")
	}
}
