package mockserver

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	APP_SERVER_ADDR  = "http://127.0.0.1"
	APP_SERVER_PORT  = ":" + "8080"
	APP_SERVER       = APP_SERVER_ADDR + APP_SERVER_PORT
	DATA_SERVER_ADDR = "http://127.0.0.1"
	DATA_SERVER_PORT = ":" + "443"
	DATA_SERVER      = DATA_SERVER_ADDR + DATA_SERVER_PORT
)

func HttpGetRequest(url string) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	responseText, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(responseText), nil

}
func HttpServerSanityCheck(t *testing.T, url string) {
	t.Helper()
	resString, err := HttpGetRequest(url)
	require.NoErrorf(t, err, "Healthcheck failed: %v", err)
	require.Contains(t, resString, HealthCheckMessage)
}
func HttpServerCheckData(t *testing.T) {
	t.Helper()
	resString, err := HttpGetRequest(APP_SERVER + "/check-data")
	require.NoErrorf(t, err, "Healthcheck failed: %v", err)
	// require.Contains(t, resString, HealthCheckMessage)
	t.Logf("resString: %s", resString)
}
func HttpServerSendTrace(t *require.TestingT) {}
func TestHttpServer(t *testing.T) {
	serverControlChan := startHttpServer()
	time.Sleep(3 * time.Second)
	HttpServerSanityCheck(t, APP_SERVER)
	HttpServerSanityCheck(t, DATA_SERVER)
	HttpServerCheckData(t)
	time.Sleep(5 * time.Minute)
	serverControlChan <- 0

}
