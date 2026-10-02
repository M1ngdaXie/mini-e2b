package fc

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"time"
)

func NewUDSClient(sockPath string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", sockPath)
			},
		},
	}
}

func UDSRequest(client *http.Client, method, url string, headers map[string]string, body []byte) (int, http.Header, []byte, error) {

	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, nil, err
	}
	// if resp.StatusCode != 204 {
	// 	return 0, nil, nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, respBody)
	// }
	return resp.StatusCode, resp.Header, respBody, nil
}
