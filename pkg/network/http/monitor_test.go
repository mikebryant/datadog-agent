// +build linux_bpf

package http

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	nethttp "net/http"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/DataDog/datadog-agent/pkg/network/config"
	"github.com/DataDog/datadog-agent/pkg/util/kernel"
	"github.com/stretchr/testify/require"
)

func TestHTTPMonitorIntegration(t *testing.T) {
	currKernelVersion, err := kernel.HostVersion()
	require.NoError(t, err)
	if currKernelVersion < kernel.VersionCode(4, 1, 0) {
		t.Skip("HTTP feature not available on pre 4.1.0 kernels")
	}

	srvDoneFn := serverSetup(t)
	defer srvDoneFn()

	// Create a monitor that simply buffers all HTTP requests
	var buffer []httpTX
	handlerFn := func(transactions []httpTX) {
		buffer = append(buffer, transactions...)
	}

	monitor, err := NewMonitor(config.New())
	require.NoError(t, err)
	monitor.handler = handlerFn
	err = monitor.Start()
	require.NoError(t, err)
	defer monitor.Stop()

	// Perform a number of random requests
	requestFn := requestGenerator(t)
	var requests []*nethttp.Request
	for i := 0; i < 100; i++ {
		requests = append(requests, requestFn())
	}

	// Ensure all captured transactions get sent to user-space
	time.Sleep(10 * time.Millisecond)
	monitor.GetHTTPStats()

	// Assert all requests made were correctly captured by the monitor
	for _, req := range requests {
		hasMatchingTX(t, req, buffer)
	}
}

func hasMatchingTX(t *testing.T, req *nethttp.Request, transactions []httpTX) {
	expectedStatus := statusFromPath(req.URL.Path)
	buffer := make([]byte, HTTPBufferSize)
	for _, tx := range transactions {
		if string(tx.Path(buffer)) == req.URL.Path && int(tx.response_status_code) == expectedStatus && tx.Method() == req.Method {
			return
		}
	}

	t.Errorf(
		"could not find HTTP transaction matching the following criteria:\n path=%s method=%s status=%d",
		req.URL.Path,
		req.Method,
		expectedStatus,
	)
}

// serverSetup spins up a HTTP test server that returns the status code included in the URL
// Example:
// * GET /200/foo returns a 200 status code;
// * PUT /404/bar returns a 404 status code;
func serverSetup(t *testing.T) func() {
	handler := func(w nethttp.ResponseWriter, req *nethttp.Request) {
		statusCode := statusFromPath(req.URL.Path)
		io.Copy(ioutil.Discard, req.Body)
		w.WriteHeader(statusCode)
	}

	srv := &nethttp.Server{
		Addr:         "localhost:8080",
		Handler:      nethttp.HandlerFunc(handler),
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
	}

	srv.SetKeepAlivesEnabled(false)

	go func() {
		_ = srv.ListenAndServe()
	}()

	return func() { srv.Shutdown(context.Background()) }
}

func requestGenerator(t *testing.T) func() *nethttp.Request {
	var (
		methods     = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
		statusCodes = []int{200, 300, 400, 500}
		random      = rand.New(rand.NewSource(time.Now().Unix()))
		idx         = 0
		client      = new(nethttp.Client)
	)

	return func() *nethttp.Request {
		idx++
		method := methods[random.Intn(len(methods))]
		status := statusCodes[random.Intn(len(statusCodes))]
		url := fmt.Sprintf("http://localhost:8080/%d/request-%d", status, idx)
		req, err := nethttp.NewRequest(method, url, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		return req
	}
}

var pathParser = regexp.MustCompile(`/(\d{3})/.+`)

func statusFromPath(path string) (status int) {
	matches := pathParser.FindStringSubmatch(path)
	if len(matches) == 2 {
		status, _ = strconv.Atoi(matches[1])
	}

	return
}
