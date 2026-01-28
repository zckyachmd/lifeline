package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// New returns a configured zerolog logger.
func New(level string) zerolog.Logger {
	l := log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
	return l
}
