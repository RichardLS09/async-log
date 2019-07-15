package midaslog

import (
	"fmt"
	"time"
)

const (
	// golang's birth, just for memory
	defaultTimeFormat = "2006-01-02 15:04:05.000"
	// must five %s == time+level+uuid+event+msg
	defaultTextFormat = "%s>%s>%s>%s>%s\n"
)

type simpleFormater struct {
	timeLayout string
	textFormat string
}

func NewSimpleFormater(timeLayout, textFormat string) *simpleFormater {
	if timeLayout == "" {
		timeLayout = defaultTimeFormat
	}
	if textFormat == "" {
		textFormat = defaultTextFormat
	}
	return &simpleFormater{timeLayout, textFormat}
}

func (s *simpleFormater) Format(now time.Time, level Level, uuid, event, msg string) string {
	lm, ok := LogLevels[level]
	if !ok {
		lm = "Unknown"
	}

	return fmt.Sprintf(s.textFormat, now.Format(s.timeLayout), lm, uuid, event, msg)
}
