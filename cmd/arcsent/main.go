package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"

	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/ipsix/arcsent/internal/cli"
	"github.com/ipsix/arcsent/internal/config"
	"github.com/ipsix/arcsent/internal/daemon"
	"github.com/ipsix/arcsent/internal/logging"
	"github.com/ipsix/arcsent/internal/storage"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "ctl" {
		runCLI(os.Args[2:])
		return
	}

	configPath := flag.String("config", config.DefaultConfigPath, "Path to config file")
	reload := flag.Bool("reload", false, "Send SIGHUP to a running arcsent process and exit")
	pidValue := flag.String("pid", "", "PID to signal for -reload (or set ARCSENT_PID)")
	flag.Parse()

	if *reload {
		pid := *pidValue
		if pid == "" {
			pid = os.Getenv("ARCSENT_PID")
		}
		if pid == "" {
			_, _ = os.Stderr.WriteString("reload error: pid is required (use -pid or ARCSENT_PID)\n")
			os.Exit(1)
		}
		parsed, err := strconv.Atoi(pid)
		if err != nil || parsed <= 0 {
			_, _ = os.Stderr.WriteString("reload error: pid must be a positive integer\n")
			os.Exit(1)
		}
		proc, err := os.FindProcess(parsed)
		if err != nil {
			_, _ = os.Stderr.WriteString("reload error: " + err.Error() + "\n")
			os.Exit(1)
		}
		if err := proc.Signal(syscall.SIGHUP); err != nil {
			_, _ = os.Stderr.WriteString("reload error: " + err.Error() + "\n")
			os.Exit(1)
		}
		_, _ = os.Stdout.WriteString("reload signal sent\n")
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		_, _ = os.Stderr.WriteString("config error: " + err.Error() + "\n")
		os.Exit(1)
	}

	logger := logging.New(cfg.Daemon.LogFormat)
	logger.Info("arcsent starting", logging.Field{Key: "config", Value: cfg.Redacted()})

	runner := daemon.New(cfg, logger, *configPath)
	if err := runner.Run(context.Background()); err != nil {
		logger.Error("daemon exited with error", logging.Field{Key: "error", Value: err.Error()})
		os.Exit(1)
	}
}

func runCLI(args []string) {
	fs := flag.NewFlagSet("ctl", flag.ExitOnError)
	addr := fs.String("addr", "http://127.0.0.1:8788", "API base URL")
	token := fs.String("token", "", "API token (or set ARCSENT_TOKEN)")
	format := fs.String("format", "json", "Output format for export (json|csv)")
	plugin := fs.String("plugin", "", "Plugin name for trigger")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON responses")
	configPath := fs.String("config", config.DefaultConfigPath, "Config path for validate/storage-check")
	envFile := fs.String("env-file", "", "Env file to load before validate/storage-check")
	fs.Parse(args)

	if *token == "" {
		*token = os.Getenv("ARCSENT_TOKEN")
	}
	if *token == "" {
		_, _ = os.Stderr.WriteString("ctl error: token is required (use -token or ARCSENT_TOKEN)\n")
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		usageCLI()
		os.Exit(2)
	}

	client := cli.NewClient(*addr, *token)
	cmd := fs.Arg(0)
	sub := ""
	if fs.NArg() > 1 {
		sub = fs.Arg(1)
	}

	ctx := context.Background()
	var (
		raw []byte
		err error
	)

	switch cmd {
	case "status":
		raw, err = client.DoJSON(ctx, http.MethodGet, "/status", nil)
	case "health":
		raw, err = client.DoJSON(ctx, http.MethodGet, "/health", nil)
	case "scanners":
		raw, err = client.DoJSON(ctx, http.MethodGet, "/scanners", nil)
	case "findings":
		raw, err = client.DoJSON(ctx, http.MethodGet, "/findings", nil)
	case "baselines":
		raw, err = client.DoJSON(ctx, http.MethodGet, "/baselines", nil)
	case "results":
		if sub == "latest" {
			raw, err = client.DoJSON(ctx, http.MethodGet, "/results/latest", nil)
		} else if sub == "history" || sub == "" {
			raw, err = client.DoJSON(ctx, http.MethodGet, "/results/history", nil)
		} else {
			usageCLI()
			os.Exit(2)
		}
	case "trigger":
		name := *plugin
		if name == "" && sub != "" {
			name = sub
		}
		if name == "" {
			_, _ = os.Stderr.WriteString("ctl error: plugin name is required\n")
			os.Exit(2)
		}
		raw, err = client.DoJSON(ctx, http.MethodPost, "/scanners/trigger/"+name, nil)
	case "signatures":
		switch sub {
		case "status":
			raw, err = client.DoJSON(ctx, http.MethodGet, "/signatures/status", nil)
		case "update":
			raw, err = client.DoJSON(ctx, http.MethodPost, "/signatures/update", nil)
		default:
			usageCLI()
			os.Exit(2)
		}
	case "export":
		switch sub {
		case "results":
			raw, err = client.DoText(ctx, http.MethodGet, "/export/results?format="+strings.ToLower(*format))
		case "baselines":
			raw, err = client.DoText(ctx, http.MethodGet, "/export/baselines?format="+strings.ToLower(*format))
		default:
			usageCLI()
			os.Exit(2)
		}
	case "metrics":
		raw, err = client.DoText(ctx, http.MethodGet, "/metrics")
	case "validate":
		err = runValidate(*configPath, *envFile)
		raw = []byte(`{"status":"ok"}`)
	case "storage-check":
		err = runStorageCheck(*configPath, *envFile)
		raw = []byte(`{"status":"ok"}`)
	default:
		usageCLI()
		os.Exit(2)
	}

	if err != nil {
		_, _ = os.Stderr.WriteString("ctl error: " + err.Error() + "\n")
		os.Exit(1)
	}
	raw = maybePrettyJSON(raw, *pretty)
	_, _ = os.Stdout.Write(raw)
	if len(raw) > 0 && raw[len(raw)-1] != '\n' {
		_, _ = os.Stdout.Write([]byte("\n"))
	}
}

func usageCLI() {
	usage := []string{
		"Usage: arcsent ctl [flags] <command>",
		"",
		"Commands:",
		"  status",
		"  health",
		"  scanners",
		"  findings",
		"  baselines",
		"  results [latest|history]",
		"  trigger <plugin>",
		"  signatures status|update",
		"  export results|baselines",
		"  metrics",
		"  validate",
		"  storage-check",
		"",
		"Flags:",
		"  -addr http://127.0.0.1:8788",
		"  -token <token> (or ARCSENT_TOKEN)",
		"  -plugin <plugin> (for trigger)",
		"  -format json|csv (for export)",
		"  -pretty (pretty-print JSON)",
		"  -config <path> (for validate/storage-check)",
		"  -env-file <path> (optional env file for validate/storage-check)",
	}
	_, _ = os.Stderr.WriteString(strings.Join(usage, "\n") + "\n")
}

func maybePrettyJSON(raw []byte, pretty bool) []byte {
	if !pretty {
		return raw
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return raw
	}
	if trimmed[0] != '{' && trimmed[0] != '[' {
		return raw
	}
	var out bytes.Buffer
	if err := json.Indent(&out, []byte(trimmed), "", "  "); err != nil {
		return raw
	}
	out.WriteByte('\n')
	return out.Bytes()
}

func runValidate(configPath, envFile string) error {
	restore, err := loadEnvFile(envFile)
	if err != nil {
		return err
	}
	defer restore()

	_, err = config.Load(configPath)
	return err
}

func runStorageCheck(configPath, envFile string) error {
	restore, err := loadEnvFile(envFile)
	if err != nil {
		return err
	}
	defer restore()

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	store, err := storage.NewBadgerStoreWithKey(cfg.Storage.DBPath, cfg.Storage.EncryptionKeyBase64)
	if err != nil {
		return err
	}
	return store.Close()
}

func loadEnvFile(path string) (func(), error) {
	if path == "" {
		return func() {}, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(raw), "\n")
	previous := map[string]*string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}
		if existing, ok := os.LookupEnv(key); ok {
			copy := existing
			previous[key] = &copy
		} else {
			previous[key] = nil
		}
		_ = os.Setenv(key, value)
	}
	return func() {
		for key, value := range previous {
			if value == nil {
				_ = os.Unsetenv(key)
				continue
			}
			_ = os.Setenv(key, *value)
		}
	}, nil
}
