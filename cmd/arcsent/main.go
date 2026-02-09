package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/ipsix/arcsent/internal/cli"
	"github.com/ipsix/arcsent/internal/config"
	"github.com/ipsix/arcsent/internal/daemon"
	"github.com/ipsix/arcsent/internal/logging"
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
		"",
		"Flags:",
		"  -addr http://127.0.0.1:8788",
		"  -token <token> (or ARCSENT_TOKEN)",
		"  -plugin <plugin> (for trigger)",
		"  -format json|csv (for export)",
		"  -pretty (pretty-print JSON)",
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
