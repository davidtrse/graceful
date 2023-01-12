package log

import (
	"go.uber.org/zap"
	"testing"
)

func TestPrintln(t *testing.T) {
	// Correct call of Printlnf:
	Printlnf("Hello %v", 1)
	// Incorrect call of Printlnf:
	Printlnf("Hello", 1)

	Info("Hey!", zap.String("foo", "bar"), zap.Int("hello", 2))
}
