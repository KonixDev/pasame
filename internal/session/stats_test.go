package session

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestStatsCountsAndIgnoresLoopback(t *testing.T) {
	var calls atomic.Int32
	s := NewStats(func() { calls.Add(1) })
	s.ClientSeen("127.0.0.1")
	s.ClientSeen("::1")
	if snap := s.Snapshot(); snap.Clients != 0 || !snap.FirstSeen.IsZero() {
		t.Fatalf("loopback contó: %+v", snap)
	}
	s.ClientSeen("192.168.1.50")
	s.ClientSeen("192.168.1.50")
	s.ClientSeen("192.168.1.51")
	s.DownloadStarted(0)
	s.DownloadStarted(0)
	s.DownloadDone(0, true)
	s.UploadStarted("IMG_1.jpg")
	if s.InFlight() != 2 {
		t.Fatalf("InFlight = %d", s.InFlight())
	}
	s.UploadDone("IMG_1.jpg", true)
	snap := s.Snapshot()
	if snap.Clients != 2 || snap.Active[0] != 1 || snap.Completed[0] != 1 ||
		len(snap.Received) != 1 || len(snap.Uploading) != 0 || snap.FirstSeen.IsZero() {
		t.Fatalf("snap = %+v", snap)
	}
	if calls.Load() == 0 {
		t.Fatal("onChange nunca se llamó")
	}
}

func TestStatsFailedDownloadNotCompleted(t *testing.T) {
	s := NewStats(nil)
	s.DownloadStarted(1)
	s.DownloadDone(1, false)
	if snap := s.Snapshot(); snap.Completed[1] != 0 || snap.Active[1] != 0 {
		t.Fatalf("snap = %+v", snap)
	}
}

func TestStatsConcurrent(t *testing.T) {
	s := NewStats(func() {})
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.ClientSeen("10.0.0.2")
			s.DownloadStarted(0)
			s.DownloadDone(0, true)
			_ = s.Snapshot()
		}()
	}
	wg.Wait()
	if s.Snapshot().Completed[0] != 100 || s.InFlight() != 0 {
		t.Fatal("conteo concurrente incorrecto")
	}
}

func TestSnapshotIsACopy(t *testing.T) {
	s := NewStats(nil)
	s.DownloadStarted(0)
	snap := s.Snapshot()
	snap.Active[0] = 99
	if s.Snapshot().Active[0] != 1 {
		t.Fatal("Snapshot comparte el mapa interno")
	}
}
