package scraper

import (
	"context"

	"github.com/embedtools/youtube-scraper/types"
)

const TabPlaylists = "EglwbGF5bGlzdHPyBgQKAkIA"

// ListChannelPlaylists streams playlists from a channel's playlists tab.
func (c *Client) ListChannelPlaylists(ctx context.Context, in *types.ListChannelPlaylistsInput, emit func(item *types.ChannelTabItem) error) (*types.ChannelTabSummary, error) {
	return c.listChannelTab(ctx, in.URL, in.Limit, in.MaxPages, TabPlaylists, "playlists", emit)
}
