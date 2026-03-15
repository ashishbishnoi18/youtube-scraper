package scraper

import (
	"context"
	"fmt"

	"github.com/embedtools/youtube-scraper/internal"
	"github.com/embedtools/youtube-scraper/types"
)

// listChannelTab is a shared helper for streaming channel tab data.
func (c *Client) listChannelTab(ctx context.Context, url string, limit, maxPages int, tabParam, tabName string, emit func(item *types.ChannelTabItem) error) (*types.ChannelTabSummary, error) {
	if ctx.Err() != nil {
		return nil, ErrContextCanceled
	}

	channelID, err := internal.ResolveChannelID(ctx, c.yt, url)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotFound, err)
	}

	browseData, err := c.yt.Browse(ctx, channelID, tabParam)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstreamChanged, err)
	}

	items, continuation := extractTabItems(browseData)
	totalEmitted := 0
	pageCount := 1
	if maxPages == 0 {
		maxPages = 100
	}

	for _, item := range items {
		if limit > 0 && totalEmitted >= limit {
			break
		}
		if err := emit(item); err != nil {
			return nil, err
		}
		totalEmitted++
	}

	for continuation != "" && pageCount < maxPages {
		if limit > 0 && totalEmitted >= limit {
			break
		}

		contData, err := c.yt.BrowseWithContinuation(ctx, continuation)
		if err != nil {
			break
		}

		moreItems, nextCont := extractContinuationItems(contData)
		for _, item := range moreItems {
			if limit > 0 && totalEmitted >= limit {
				break
			}
			if err := emit(item); err != nil {
				return nil, err
			}
			totalEmitted++
		}
		continuation = nextCont
		pageCount++
	}

	return &types.ChannelTabSummary{
		ChannelID:  channelID,
		Tab:        tabName,
		TotalItems: totalEmitted,
		Pages:      pageCount,
	}, nil
}
