package log

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogging initialize the global log format and timezone
func InitLogging(isLocalRuntimeEnv bool, level zerolog.Level) {
	// make the global logger use the same format as createBasicLogger
	zerolog.TimeFieldFormat = time.RFC3339

	// Work-around to make the time always be in UTC timezone
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().UTC()
	}

	if isLocalRuntimeEnv {
		logger := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, NoColor: true}
		logger.FormatLevel = func(i interface{}) string {
			return strings.ToUpper(fmt.Sprintf("| %-5s |", i))
		}
		logger.FormatFieldValue = func(i interface{}) string {
			return fmt.Sprintf("%s,", i)
		}
		log.Logger = zerolog.New(logger).With().Timestamp().Logger()
	}

	zerolog.SetGlobalLevel(level)
}
