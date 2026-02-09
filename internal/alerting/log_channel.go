package alerting

import "github.com/ipsix/arcsent/internal/logging"

type LogChannel struct {
	logger *logging.Logger
}

func NewLogChannel(logger *logging.Logger) *LogChannel {
	return &LogChannel{logger: logger}
}

func (l *LogChannel) Name() string { return "log" }

func (l *LogChannel) Send(alert Alert) error {
	l.logger.Warn("alert",
		logging.Field{Key: "id", Value: alert.ID},
		logging.Field{Key: "severity", Value: alert.Severity},
		logging.Field{Key: "scanner", Value: alert.ScannerName},
		logging.Field{Key: "finding", Value: alert.Finding.Description},
		logging.Field{Key: "reason", Value: alert.Reason},
	)
	return nil
}
