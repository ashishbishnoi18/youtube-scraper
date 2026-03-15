package scraper

import (
	"context"

	"github.com/embedtools/youtube-scraper/types"
)

const TabCommunity = "Egljb21tdW5pdHnyBgQKAkoA"

// ListChannelCommunity streams community posts from a channel's community tab.
func (c *Client) ListChannelCommunity(ctx context.Context, in *types.ListChannelCommunityInput, emit func(item *types.ChannelTabItem) error) (*types.ChannelTabSummary, error) {
	return c.listChannelTab(ctx, in.URL, in.Limit, in.MaxPages, TabCommunity, "community", emit)
}
