package midaslog

import (
	"io"
	"time"
)

type Level int

const (
	LEVEL_EMERGENCY = Level(iota)
	LEVEL_ALERT
	LEVEL_CRITICAL
	LEVEL_ERROR
	LEVEL_WARNING
	LEVEL_NOTICE
	LEVEL_INFO
	LEVEL_DEBUG
)

var LogLevels = map[Level]string{
	LEVEL_EMERGENCY: "emergency",
	LEVEL_ALERT:     "alert",
	LEVEL_CRITICAL:  "critical",
	LEVEL_ERROR:     "error",
	LEVEL_WARNING:   "warning",
	LEVEL_NOTICE:    "notice",
	LEVEL_INFO:      "info",
	LEVEL_DEBUG:     "debug",
}

type ILogger interface {
	Debug(msg []byte)
	Info(msg []byte)
	Notice(msg []byte)
	Warning(msg []byte)
	Error(msg []byte)
	Critical(msg []byte)
	Alert(msg []byte)
	Emergency(msg []byte)

	Log(level Level, msg []byte) error
}

type IFormater interface {
	Format(now time.Time, level Level, uuid, event, msg string) string
}

type IWriter interface {
	Write(level Level, p []byte, now time.Time) (int, error)
}

type IWriterWithTime interface {
	WriteWithTime(p []byte, now time.Time) (n int, err error)
	io.WriteCloser
}
