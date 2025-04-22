package policy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	errors "github.com/eclipse-xfsc/microservice-core-go/pkg/err"
)

type Client struct {
	addr       string
	httpClient *http.Client
}

func New(addr string, httpClient *http.Client) *Client {
	return &Client{
		addr:       addr,
		httpClient: httpClient,
	}
}

// Evaluate calls the policy service to execute the given policy.
// The policy is expected as a string path uniquely identifying the
// policy that has to be evaluated. For example, with policy = `policies/xfsc/didResolve/1.0`,
// the client will do HTTP request to http://policyhost/policy/policies/xfsc/didResolve/1.0/evaluation.
func (c *Client) Evaluate(ctx context.Context, policy string, data []byte) ([]byte, error) {
	uri := c.addr + "/policy/" + policy + "/evaluation"
	policyURL, err := url.ParseRequestURI(uri)
	if err != nil {
		return nil, errors.New(errors.BadRequest, "invalid policy evaluation URL", err)
	}

	req, err := http.NewRequest("POST", policyURL.String(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response on policy evaluation: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
