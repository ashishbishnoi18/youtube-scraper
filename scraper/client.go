package scraper

import (
	"net/http"

	"github.com/embedtools/youtube-scraper/internal"
)

// Client is the YouTube scraper client. All outbound HTTP uses the injected
// *http.Client (or http.DefaultClient if none is provided).
type Client struct {
	http *http.Client
	yt   *internal.InnerTubeClient
}

// New creates a new YouTube scraper Client.
func New(opts ...Option) (*Client, error) {
	c := &Client{
		http: http.DefaultClient,
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	c.yt = internal.NewInnerTubeClient(c.http)
	return c, nil
}
