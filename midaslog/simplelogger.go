package midaslog

import (
	"reflect"
	"time"
	"unsafe"
)

type simpleLogger struct {
	writer   IWriter
	formater IFormater

	glevel Level
}

func NewSimpleLogger(writer IWriter, formater IFormater) *simpleLogger {
	return &simpleLogger{
		writer:   writer,
		formater: formater,

		glevel: LEVEL_INFO,
	}
}

// for line programming
func (s *simpleLogger) SetLogLevel(level Level) *simpleLogger {
	_, ok := LogLevels[level]
	if ok {
		s.glevel = level
	}

	return s
}

func (s *simpleLogger) Debug(uuid, event, msg string) {
	s.Log(LEVEL_DEBUG, uuid, event, msg)
}

func (s *simpleLogger) Info(uuid, event, msg string) {
	s.Log(LEVEL_INFO, uuid, event, msg)
}

func (s *simpleLogger) Notice(uuid, event, msg string) {
	s.Log(LEVEL_NOTICE, uuid, event, msg)
}

func (s *simpleLogger) Warning(uuid, event, msg string) {
	s.Log(LEVEL_WARNING, uuid, event, msg)
}

func (s *simpleLogger) Error(uuid, event, msg string) {
	s.Log(LEVEL_ERROR, uuid, event, msg)
}

func (s *simpleLogger) Critical(uuid, event, msg string) {
	s.Log(LEVEL_CRITICAL, uuid, event, msg)
}

func (s *simpleLogger) Alert(uuid, event, msg string) {
	s.Log(LEVEL_ALERT, uuid, event, msg)
}

func (s *simpleLogger) Emergency(uuid, event, msg string) {
	s.Log(LEVEL_EMERGENCY, uuid, event, msg)
}

func (s *simpleLogger) Log(level Level, uuid, event, msg string) error {
	if level > s.glevel {
		return nil
	}
	now := time.Now().Local()
	ss := s.formater.Format(now, level, uuid, event, msg)
	// []byte can opt
	_, err := s.writer.Write(level, s2b(ss), now)
	return err
}

func s2b(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{
		Data: sh.Data,
		Len:  sh.Len,
		Cap:  sh.Len,
	}
	return *(*[]byte)(unsafe.Pointer(&bh))
}
