package cmd

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"

	"github.com/containifyci/dunebot/cmd"

	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	WithTimeout(t, 5*time.Second, func(t *testing.T) {
		// Get the root command
		rootCmd := cmd.RootCmd

		// Capture the output
		output := new(bytes.Buffer)
		rootCmd.SetOut(output)

		t.Setenv("TOKEN_SYNC_PERIOD", "0m")

		// Execute the greet command
		httpPort := fmt.Sprintf("%d", getFreePort())
		rootCmd.SetArgs([]string{"token", "service", "--storage", "", "--httpport", httpPort})
		go func() {
			time.Sleep(1 * time.Second)
			t.Logf("Send SIGTERM")

			err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
			assert.NoError(t, err)
		}()
		err := rootCmd.Execute()
		assert.NoError(t, err)
	})
}

// Utility

func WithTimeout(t *testing.T, timeout time.Duration, f func(t *testing.T)) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		f(t)
		close(done)
	}()
	select {
	case <-time.After(timeout):
		t.Fatalf("test didn't finish in time %s", timeout)
	case <-done:
	}
}

// Create a new instance of the test HTTP server
func RepositoryServer(response string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(response))
		if err != nil {
			panic(err)
		}
	}))
	return server
}

func getFreePort() int {
	var a *net.TCPAddr
	a, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	var l *net.TCPListener
	l, err = net.ListenTCP("tcp", a)
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
