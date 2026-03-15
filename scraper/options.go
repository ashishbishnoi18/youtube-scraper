package scraper

import "net/http"

// Option configures a Client.
type Option func(*Client) error

// WithHTTPClient injects a custom *http.Client for all outbound HTTP.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) error {
		if c == nil {
			return ErrInvalidURL
		}
		cl.http = c
		return nil
	}
}
