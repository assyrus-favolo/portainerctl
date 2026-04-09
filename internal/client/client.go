package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/portainer/portainerctl/internal/config"
)

// Client wraps all HTTP calls to the Portainer BE 2.39.1 API.
type Client struct {
	baseURL    string
	token      string
	http       *http.Client
}

type APIError struct {
	Message string `json:"message"`
	Details string `json:"details"`
	Status  int
}

func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("API error %d: %s (%s)", e.Status, e.Message, e.Details)
	}
	return fmt.Sprintf("API error %d: %s", e.Status, e.Message)
}

// New constructs a Client from the active config context.
func New(ctx *config.Context) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: ctx.Insecure},
	}
	return &Client{
		baseURL: strings.TrimRight(ctx.URL, "/") + "/api",
		token:   ctx.Token,
		http: &http.Client{
			Timeout:   60 * time.Second,
			Transport: transport,
		},
	}
}

func (c *Client) newRequest(method, path string, body interface{}) (*http.Request, error) {
	url := c.baseURL + path
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c *Client) do(req *http.Request, out interface{}) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode >= 400 {
		var apiErr APIError
		apiErr.Status = resp.StatusCode
		_ = json.Unmarshal(data, &apiErr)
		if apiErr.Message == "" {
			apiErr.Message = string(data)
		}
		return &apiErr
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

func (c *Client) Get(path string, out interface{}) error {
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *Client) Post(path string, body, out interface{}) error {
	req, err := c.newRequest("POST", path, body)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *Client) Put(path string, body, out interface{}) error {
	req, err := c.newRequest("PUT", path, body)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *Client) Delete(path string) error {
	req, err := c.newRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) DeleteWithBody(path string, body interface{}) error {
	req, err := c.newRequest("DELETE", path, body)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

// RawGet returns the raw response body — used for passthrough commands (kubectl proxy, docker proxy).
func (c *Client) RawGet(path string) ([]byte, error) {
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// ProxyRequest forwards an arbitrary method+path+body through the Portainer API proxy.
// Used by the `kubectl` and `docker` passthrough subcommands.
func (c *Client) ProxyRequest(method, path string, body io.Reader) (int, []byte, error) {
	url := c.baseURL + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("X-API-Key", c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return resp.StatusCode, data, err
}
