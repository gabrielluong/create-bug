package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type CreateBugParams struct {
	Product     string   `json:"product"`
	Component   string   `json:"component"`
	Summary     string   `json:"summary"`
	Version     string   `json:"version"`
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	OpSys       string   `json:"op_sys,omitempty"`
	Platform    string   `json:"platform,omitempty"`
	Priority    string   `json:"priority,omitempty"`
	Severity    string   `json:"severity,omitempty"`
	Alias       string   `json:"alias,omitempty"`
	AssignedTo  string   `json:"assigned_to,omitempty"`
	CC          []string `json:"cc,omitempty"`
	Status      string   `json:"status,omitempty"`
	Blocks      []int    `json:"blocks,omitempty"`
	DependsOn   []int    `json:"depends_on,omitempty"`
}

type CreateBugResult struct {
	ID      int    `json:"id"`
	URL     string `json:"url,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

type bugzillaError struct {
	Error   bool   `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

var errorMessages = map[int]string{
	51:  "Invalid object",
	103: "Invalid alias",
	104: "Invalid field",
	105: "Invalid component",
	106: "Invalid product",
	107: "Invalid summary — summary is required",
	116: "Dependency loop detected",
	120: "Group restriction denied",
	504: "Invalid user",
}

func NewClient(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{},
	}
}

func (c *Client) buildRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-BUGZILLA-API-KEY", c.apiKey)
	}
	return req, nil
}

func (c *Client) checkError(body []byte) error {
	var errResp bugzillaError
	if err := json.Unmarshal(body, &errResp); err != nil {
		return nil
	}
	if !errResp.Error {
		return nil
	}
	msg := errResp.Message
	if mapped, ok := errorMessages[errResp.Code]; ok {
		msg = mapped
	}
	return fmt.Errorf("Bugzilla error (%d): %s", errResp.Code, msg)
}

func (c *Client) CreateBug(params CreateBugParams) (*CreateBugResult, error) {
	u := fmt.Sprintf("%s/rest/bug", c.baseURL)

	jsonBody, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req, err := c.buildRequest("POST", u, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := c.checkError(body); err != nil {
		return nil, err
	}

	var result CreateBugResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
