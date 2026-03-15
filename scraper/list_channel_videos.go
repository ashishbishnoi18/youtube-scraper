package scraper

import (
	"context"
	"fmt"

	"github.com/embedtools/youtube-scraper/internal"
	"github.com/embedtools/youtube-scraper/types"
)

const TabVideos = "EgZ2aWRlb3PyBgQKAjoA"

// ListChannelVideos streams videos from a channel's videos tab.
func (c *Client) ListChannelVideos(ctx context.Context, in *types.ListChannelVideosInput, emit func(item *types.ChannelTabItem) error) (*types.ChannelTabSummary, error) {
	if ctx.Err() != nil {
		return nil, ErrContextCanceled
	}

	channelID, err := internal.ResolveChannelID(ctx, c.yt, in.URL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotFound, err)
	}

	browseData, err := c.yt.Browse(ctx, channelID, TabVideos)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstreamChanged, err)
	}

	items, continuation := extractTabItems(browseData)
	totalEmitted := 0
	pageCount := 1
	maxPages := in.MaxPages
	if maxPages == 0 {
		maxPages = 100
	}

	for _, item := range items {
		if in.Limit > 0 && totalEmitted >= in.Limit {
			break
		}
		if err := emit(item); err != nil {
			return nil, err
		}
		totalEmitted++
	}

	for continuation != "" && pageCount < maxPages {
		if in.Limit > 0 && totalEmitted >= in.Limit {
			break
		}

		contData, err := c.yt.BrowseWithContinuation(ctx, continuation)
		if err != nil {
			break
		}

		moreItems, nextCont := extractContinuationItems(contData)
		for _, item := range moreItems {
			if in.Limit > 0 && totalEmitted >= in.Limit {
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
		Tab:        "videos",
		TotalItems: totalEmitted,
		Pages:      pageCount,
	}, nil
}

// extractTabItems extracts items from initial tab browse response.
func extractTabItems(browseData map[string]interface{}) ([]*types.ChannelTabItem, string) {
	var items []*types.ChannelTabItem
	var continuation string

	contents, ok := browseData["contents"].(map[string]interface{})
	if !ok {
		return items, ""
	}
	twoCol, ok := contents["twoColumnBrowseResultsRenderer"].(map[string]interface{})
	if !ok {
		return items, ""
	}
	tabs, ok := twoCol["tabs"].([]interface{})
	if !ok {
		return items, ""
	}

	for _, tab := range tabs {
		tm, ok := tab.(map[string]interface{})
		if !ok {
			continue
		}
		tr, ok := tm["tabRenderer"].(map[string]interface{})
		if !ok {
			continue
		}
		if !internal.GetBool(tr, "selected") {
			continue
		}
		content, ok := tr["content"].(map[string]interface{})
		if !ok {
			continue
		}
		items, continuation = extractRichGridItems(content)
	}

	return items, continuation
}

// extractRichGridItems extracts items from richGridRenderer or sectionListRenderer.
func extractRichGridItems(content map[string]interface{}) ([]*types.ChannelTabItem, string) {
	var items []*types.ChannelTabItem
	var continuation string

	// richGridRenderer (videos, shorts)
	if richGrid, ok := content["richGridRenderer"].(map[string]interface{}); ok {
		if contentsList, ok := richGrid["contents"].([]interface{}); ok {
			for _, item := range contentsList {
				im, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				if richItem, ok := im["richItemRenderer"].(map[string]interface{}); ok {
					if itemContent, ok := richItem["content"].(map[string]interface{}); ok {
						if extracted := extractItemByType(itemContent); extracted != nil {
							items = append(items, extracted)
						}
					}
				}
				if contItem, ok := im["continuationItemRenderer"].(map[string]interface{}); ok {
					continuation = internal.ExtractContinuationToken(contItem)
				}
			}
		}
	}

	// sectionListRenderer (community posts)
	if sectionList, ok := content["sectionListRenderer"].(map[string]interface{}); ok {
		if contentsList, ok := sectionList["contents"].([]interface{}); ok {
			for _, section := range contentsList {
				sm, ok := section.(map[string]interface{})
				if !ok {
					continue
				}
				if itemSection, ok := sm["itemSectionRenderer"].(map[string]interface{}); ok {
					if sectionContents, ok := itemSection["contents"].([]interface{}); ok {
						for _, item := range sectionContents {
							im, ok := item.(map[string]interface{})
							if !ok {
								continue
							}
							if extracted := extractItemByType(im); extracted != nil {
								items = append(items, extracted)
							}
							if contItem, ok := im["continuationItemRenderer"].(map[string]interface{}); ok {
								continuation = internal.ExtractContinuationToken(contItem)
							}
						}
					}
				}
			}
		}
	}

	return items, continuation
}

// extractContinuationItems extracts items from a continuation response.
func extractContinuationItems(contData map[string]interface{}) ([]*types.ChannelTabItem, string) {
	var items []*types.ChannelTabItem
	var continuation string

	actions, ok := contData["onResponseReceivedActions"].([]interface{})
	if !ok {
		return items, ""
	}

	for _, action := range actions {
		am, ok := action.(map[string]interface{})
		if !ok {
			continue
		}
		appendAction, ok := am["appendContinuationItemsAction"].(map[string]interface{})
		if !ok {
			continue
		}
		contItems, ok := appendAction["continuationItems"].([]interface{})
		if !ok {
			continue
		}
		for _, item := range contItems {
			im, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			// Rich item wrapper
			if richItem, ok := im["richItemRenderer"].(map[string]interface{}); ok {
				if content, ok := richItem["content"].(map[string]interface{}); ok {
					if extracted := extractItemByType(content); extracted != nil {
						items = append(items, extracted)
					}
				}
			}
			// Direct items (community posts)
			if extracted := extractItemByType(im); extracted != nil {
				items = append(items, extracted)
			}
			// Continuation
			if contItem, ok := im["continuationItemRenderer"].(map[string]interface{}); ok {
				continuation = internal.ExtractContinuationToken(contItem)
			}
		}
	}

	return items, continuation
}

// extractItemByType dispatches to the correct extractor based on renderer type.
func extractItemByType(itemMap map[string]interface{}) *types.ChannelTabItem {
	if vr, ok := itemMap["videoRenderer"].(map[string]interface{}); ok {
		return extractVideoTabItem(vr)
	}
	if ri, ok := itemMap["reelItemRenderer"].(map[string]interface{}); ok {
		return extractReelTabItem(ri)
	}
	if sl, ok := itemMap["shortsLockupViewModel"].(map[string]interface{}); ok {
		return extractShortsLockupTabItem(sl)
	}
	if gp, ok := itemMap["gridPlaylistRenderer"].(map[string]interface{}); ok {
		return extractPlaylistTabItem(gp)
	}
	if pr, ok := itemMap["playlistRenderer"].(map[string]interface{}); ok {
		return extractPlaylistTabItem(pr)
	}
	if bpt, ok := itemMap["backstagePostThreadRenderer"].(map[string]interface{}); ok {
		if post, ok := bpt["post"].(map[string]interface{}); ok {
			if bp, ok := post["backstagePostRenderer"].(map[string]interface{}); ok {
				return extractCommunityPostItem(bp)
			}
		}
	}
	return nil
}

func extractVideoTabItem(r map[string]interface{}) *types.ChannelTabItem {
	return &types.ChannelTabItem{
		Type:       "video",
		VideoID:    internal.GetString(r, "videoId"),
		Title:      internal.GetTextFromRuns(r, "title"),
		Description: internal.GetTextFromRuns(r, "descriptionSnippet"),
		Duration:   internal.GetTextFromSimpleText(r, "lengthText"),
		Views:      internal.GetTextFromSimpleText(r, "viewCountText"),
		Published:  internal.GetTextFromSimpleText(r, "publishedTimeText"),
		Thumbnails: internal.GetThumbnails(r),
	}
}

func extractReelTabItem(r map[string]interface{}) *types.ChannelTabItem {
	return &types.ChannelTabItem{
		Type:       "short",
		VideoID:    internal.GetString(r, "videoId"),
		Headline:   internal.GetTextFromSimpleText(r, "headline"),
		Views:      internal.GetTextFromSimpleText(r, "viewCountText"),
		Thumbnails: internal.GetThumbnails(r),
	}
}

func extractShortsLockupTabItem(lockup map[string]interface{}) *types.ChannelTabItem {
	item := &types.ChannelTabItem{
		Type:             "short",
		AccessibilityTxt: internal.GetString(lockup, "accessibilityText"),
		EntityID:         internal.GetString(lockup, "entityId"),
	}

	if onTap, ok := lockup["onTap"].(map[string]interface{}); ok {
		if cmd, ok := onTap["innertubeCommand"].(map[string]interface{}); ok {
			if rw, ok := cmd["reelWatchEndpoint"].(map[string]interface{}); ok {
				item.VideoID = internal.GetString(rw, "videoId")
				if thumb, ok := rw["thumbnail"].(map[string]interface{}); ok {
					if t, ok := thumb["thumbnails"].([]interface{}); ok {
						item.Thumbnails = t
					}
				}
			}
		}
	}

	// Parse title and views from accessibility text
	a11y := internal.GetString(lockup, "accessibilityText")
	if a11y != "" {
		if idx := len(a11y) - len(" - play Short"); idx > 0 && a11y[idx:] == " - play Short" {
			a11y = a11y[:idx]
		}
		for i := len(a11y) - 1; i >= 0; i-- {
			if a11y[i] == ',' {
				item.Headline = trimSpace(a11y[:i])
				item.Views = trimSpace(a11y[i+1:])
				break
			}
		}
	}

	return item
}

func extractPlaylistTabItem(r map[string]interface{}) *types.ChannelTabItem {
	return &types.ChannelTabItem{
		Type:       "playlist",
		PlaylistID: internal.GetString(r, "playlistId"),
		Title:      internal.GetTextFromRuns(r, "title"),
		VideoCount: internal.GetTextFromRuns(r, "videoCountText"),
		Thumbnails: internal.GetThumbnails(r),
	}
}

func extractCommunityPostItem(r map[string]interface{}) *types.ChannelTabItem {
	return &types.ChannelTabItem{
		Type:      "post",
		PostID:    internal.GetString(r, "postId"),
		Content:   internal.GetTextFromRuns(r, "contentText"),
		Published: internal.GetTextFromRuns(r, "publishedTimeText"),
		VoteCount: internal.GetTextFromSimpleText(r, "voteCount"),
		Author:    internal.GetTextFromRuns(r, "authorText"),
	}
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
