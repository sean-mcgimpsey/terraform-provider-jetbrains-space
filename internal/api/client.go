package jetbrains_space_api_client_go

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseAPIEndpoint = "/api/http/projects"
)

func NewClient(host, token string) (*Client, error) {
	c := Client{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		HostURL:    host,
		Token:      token,
	}

	if host == "" {
		return nil, errors.New("ERROR: Host is undefined")
	}
	if token == "" {
		return nil, errors.New("token is undefined")
	}

	return &c, nil
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	req.Header.Set("Accept", "application/json")

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status: %d, body: %s", res.StatusCode, body)
	}

	return body, err
}
