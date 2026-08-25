// services/logger/logger.go

// Package logger is a Wails-bound service bridging frontend log calls to
// s99logger.
package logger

import "github.com/smegg99/s99logger"

type Service struct {
	logger *s99logger.Logger
}

// New creates the service. logger may be nil; log calls become no-ops.
func New(logger *s99logger.Logger) *Service {
	if logger == nil {
		return &Service{}
	}
	return &Service{logger: logger.With(s99logger.String("source", "frontend"))}
}

func (s *Service) LogDebug(message string, fields map[string]any) {
	if s.logger == nil {
		return
	}
	s.logger.Debug(s99logger.NewEvent(s99logger.MessageID(message), attrs(fields)...))
}

func (s *Service) LogInfo(message string, fields map[string]any) {
	if s.logger == nil {
		return
	}
	s.logger.Info(s99logger.NewEvent(s99logger.MessageID(message), attrs(fields)...))
}

func (s *Service) LogWarn(message string, fields map[string]any) {
	if s.logger == nil {
		return
	}
	s.logger.Warn(s99logger.NewEvent(s99logger.MessageID(message), attrs(fields)...))
}

func (s *Service) LogError(message string, fields map[string]any) {
	if s.logger == nil {
		return
	}
	s.logger.Error(s99logger.NewEvent(s99logger.MessageID(message), attrs(fields)...))
}

func attrs(fields map[string]any) []s99logger.Attr {
	attrs := make([]s99logger.Attr, 0, len(fields))
	for key, value := range fields {
		attrs = append(attrs, s99logger.Any(key, value))
	}
	return attrs
}
