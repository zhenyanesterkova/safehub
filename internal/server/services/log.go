package services

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

type LogService struct {
	Log *logrus.Logger
}

func NewLogService(lvl string) (*LogService, error) {
	logrusLvl, err := logrus.ParseLevel(lvl)
	if err != nil {
		return nil, fmt.Errorf("failed parse log level: %w", err)
	}

	logger := logrus.New()
	logger.SetLevel(logrusLvl)

	return &LogService{
		Log: logger,
	}, nil
}
