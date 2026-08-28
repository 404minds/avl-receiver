package store

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/404minds/avl-receiver/internal/types"
)

// Records still queued when the connection closes must be written, not dropped: CloseChan and
// ProcessChan used to race in the same select.
func TestJsonLinesStoreFlushesOnClose(t *testing.T) {
	f, err := os.Create(filepath.Join(t.TempDir(), "x.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s := &JsonLinesStore{File: f, ProcessChan: make(chan *types.DeviceStatus, 10), CloseChan: make(chan bool, 1), DeviceID: "x"}
	for i := 0; i < 5; i++ {
		s.ProcessChan <- &types.DeviceStatus{Imei: "x"}
	}
	s.CloseChan <- true // both channels ready before Process even starts
	s.Process(context.Background())

	rf, _ := os.Open(f.Name())
	defer rf.Close()
	n := 0
	for sc := bufio.NewScanner(rf); sc.Scan(); {
		n++
	}
	if n != 5 {
		t.Fatalf("got %d records written, want 5", n)
	}
}
