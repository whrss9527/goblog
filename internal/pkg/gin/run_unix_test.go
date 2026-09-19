//go:build unix

package gin

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A port that is taken must end the process with an error. It used to be logged while the process kept
// idling, which looks like a healthy service to systemd.
func TestRunGinReportsListenFailure(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer taken.Close()
	port := uint32(taken.Addr().(*net.TCPAddr).Port)

	done := make(chan error, 1)
	go func() { done <- RunGin(InitGinConfig("test"), "127.0.0.1", port, time.Second) }()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "listen on 127.0.0.1:") {
			t.Errorf("RunGin() = %v, want a listen error naming the address", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RunGin keeps running although it cannot listen")
	}
}

func TestRunGinServesOnTheConfiguredHostUntilSignalled(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := uint32(probe.Addr().(*net.TCPAddr).Port)
	probe.Close()

	done := make(chan error, 1)
	go func() { done <- RunGin(InitGinConfig("test"), "127.0.0.1", port, 2*time.Second) }()

	url := fmt.Sprintf("http://127.0.0.1:%d/ping", port)
	var res *http.Response
	for i := 0; i < 50; i++ {
		if res, err = http.Get(url); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("server did not come up: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("/ping = %d", res.StatusCode)
	}

	// SIGTERM is what systemd sends; RunGin has it registered, so the test binary survives it
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("graceful shutdown returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RunGin did not stop on SIGTERM")
	}
	if _, err := http.Get(url); err == nil {
		t.Error("the port is still served after shutdown")
	}
}
