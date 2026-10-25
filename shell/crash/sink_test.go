// shell/crash/sink_test.go

package crash

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/smegg99/s99logger"
)

func TestSinkKeepsOneLinePerRecord(t *testing.T) {
	rec := s99logger.Record{
		Time: time.Date(2026, 10, 28, 12, 3, 4, 5e6, time.UTC), Level: s99logger.LevelWarn, MessageID: "audio.lost",
		Attrs: []s99logger.Attr{s99logger.String("device", "Mic\nTwo"), s99logger.Int("rate", 48000)},
	}
	if err := Sink().Write(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	lines := local.snapshot()
	if got := lines[len(lines)-1]; got != "12:03:04.005 WARN audio.lost device=Mic Two rate=48000" {
		t.Fatalf("line = %q", got)
	}
}

func TestSendWithoutWatcherDoesNothing(t *testing.T) {
	send(lineLog, strings.Repeat("x", 10))
	Done()
}
