package scraper

import (
	"context"

	"github.com/embedtools/youtube-scraper/types"
)

const TabShorts = "EgZzaG9ydHPyBgUKA5oBAA%3D%3D"

// ListChannelShorts streams shorts from a channel's shorts tab.
func (c *Client) ListChannelShorts(ctx context.Context, in *types.ListChannelShortsInput, emit func(item *types.ChannelTabItem) error) (*types.ChannelTabSummary, error) {
	return c.listChannelTab(ctx, in.URL, in.Limit, in.MaxPages, TabShorts, "shorts", emit)
}
