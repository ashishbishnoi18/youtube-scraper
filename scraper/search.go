package scraper

import (
	"context"
	"fmt"

	"github.com/embedtools/youtube-scraper/internal"
	"github.com/embedtools/youtube-scraper/types"
)

// Search filter params.
const (
	FilterVideos    = "EgIQAQ%3D%3D"
	FilterChannels  = "EgIQAg%3D%3D"
	FilterPlaylists = "EgIQAw%3D%3D"
	FilterLive      = "EgJAAQ%3D%3D"
	FilterShorts    = "EgkYChgDcgUYAcgBAQ%3D%3D"
	FilterToday     = "EgQIARAB"
	FilterThisWeek  = "EgQIAhAB"
	FilterThisMonth = "EgQIAxAB"
	FilterThisYear  = "EgQIBBAB"
)

var filterMap = map[string]string{
	"videos":    FilterVideos,
	"channels":  FilterChannels,
	"playlists": FilterPlaylists,
	"live":      FilterLive,
	"shorts":    FilterShorts,
	"today":     FilterToday,
	"week":      FilterThisWeek,
	"month":     FilterThisMonth,
	"year":      FilterThisYear,
}

// Search performs a YouTube search query.
func (c *Client) Search(ctx context.Context, in *types.SearchInput) (*types.SearchOutput, error) {
	if ctx.Err() != nil {
		return nil, ErrContextCanceled
	}
	if in.Query == "" {
		return nil, fmt.Errorf("%w: search query required", ErrInvalidURL)
	}

	params := ""
	if in.Filter != "" {
		if p, ok := filterMap[in.Filter]; ok {
			params = p
		}
	}

	searchData, err := c.yt.Search(ctx, in.Query, params)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstreamChanged, err)
	}

	results, continuation := extractSearchResults(searchData)

	pageCount := 1
	maxPages := in.MaxPages
	if maxPages == 0 {
		maxPages = 10
	}

	for continuation != "" && pageCount < maxPages {
		if in.Limit > 0 && len(results) >= in.Limit {
			break
		}

		contData, err := c.yt.SearchWithContinuation(ctx, continuation)
		if err != nil {
			break
		}

		moreResults, nextCont := extractSearchContinuationResults(contData)
		results = append(results, moreResults...)
		continuation = nextCont
		pageCount++
	}

	if in.Limit > 0 && len(results) > in.Limit {
		results = results[:in.Limit]
	}

	return &types.SearchOutput{
		Query:        in.Query,
		Results:      results,
		ResultsCount: len(results),
		Pages:        pageCount,
	}, nil
}

func extractSearchResults(data map[string]interface{}) ([]types.SearchResultItem, string) {
	var results []types.SearchResultItem
	var continuation string

	contents, ok := data["contents"].(map[string]interface{})
	if !ok {
		return results, ""
	}
	twoCol, ok := contents["twoColumnSearchResultsRenderer"].(map[string]interface{})
	if !ok {
		return results, ""
	}
	primary, ok := twoCol["primaryContents"].(map[string]interface{})
	if !ok {
		return results, ""
	}
	sectionList, ok := primary["sectionListRenderer"].(map[string]interface{})
	if !ok {
		return results, ""
	}
	sectionContents, ok := sectionList["contents"].([]interface{})
	if !ok {
		return results, ""
	}

	for _, section := range sectionContents {
		sm, ok := section.(map[string]interface{})
		if !ok {
			continue
		}
		if itemSection, ok := sm["itemSectionRenderer"].(map[string]interface{}); ok {
			if items, ok := itemSection["contents"].([]interface{}); ok {
				for _, item := range items {
					if im, ok := item.(map[string]interface{}); ok {
						if extracted := extractSearchItem(im); extracted != nil {
							results = append(results, *extracted)
						}
					}
				}
			}
		}
		if cr, ok := sm["continuationItemRenderer"].(map[string]interface{}); ok {
			continuation = internal.ExtractContinuationToken(cr)
		}
	}

	return results, continuation
}

func extractSearchContinuationResults(data map[string]interface{}) ([]types.SearchResultItem, string) {
	var results []types.SearchResultItem
	var continuation string

	actions, ok := data["onResponseReceivedCommands"].([]interface{})
	if !ok {
		return results, ""
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
			if itemSection, ok := im["itemSectionRenderer"].(map[string]interface{}); ok {
				if items, ok := itemSection["contents"].([]interface{}); ok {
					for _, i := range items {
						if iMap, ok := i.(map[string]interface{}); ok {
							if extracted := extractSearchItem(iMap); extracted != nil {
								results = append(results, *extracted)
							}
						}
					}
				}
			}
			if cr, ok := im["continuationItemRenderer"].(map[string]interface{}); ok {
				continuation = internal.ExtractContinuationToken(cr)
			}
		}
	}

	return results, continuation
}

func extractSearchItem(itemMap map[string]interface{}) *types.SearchResultItem {
	if vr, ok := itemMap["videoRenderer"].(map[string]interface{}); ok {
		return &types.SearchResultItem{
			Type:        "video",
			VideoID:     internal.GetString(vr, "videoId"),
			Title:       internal.GetTextFromRuns(vr, "title"),
			Description: internal.GetTextFromRuns(vr, "descriptionSnippet"),
			Channel:     internal.GetTextFromRuns(vr, "ownerText"),
			ChannelID:   internal.ExtractChannelIDFromOwner(vr),
			Duration:    internal.GetTextFromSimpleText(vr, "lengthText"),
			Views:       internal.GetTextFromSimpleText(vr, "viewCountText"),
			Published:   internal.GetTextFromSimpleText(vr, "publishedTimeText"),
			Thumbnails:  internal.GetThumbnails(vr),
			IsLive:      internal.CheckIsLive(vr),
		}
	}

	if cr, ok := itemMap["channelRenderer"].(map[string]interface{}); ok {
		return &types.SearchResultItem{
			Type:            "channel",
			ChannelID:       internal.GetString(cr, "channelId"),
			Title:           internal.GetTextFromSimpleText(cr, "title"),
			Description:     internal.GetTextFromRuns(cr, "descriptionSnippet"),
			SubscriberCount: internal.GetTextFromSimpleText(cr, "subscriberCountText"),
			VideoCount:      internal.GetTextFromRuns(cr, "videoCountText"),
			Thumbnails:      internal.GetThumbnails(cr),
		}
	}

	if pr, ok := itemMap["playlistRenderer"].(map[string]interface{}); ok {
		return &types.SearchResultItem{
			Type:       "playlist",
			PlaylistID: internal.GetString(pr, "playlistId"),
			Title:      internal.GetTextFromSimpleText(pr, "title"),
			VideoCount: internal.GetString(pr, "videoCount"),
			Channel:    internal.GetTextFromRuns(pr, "longBylineText"),
			Thumbnails: internal.GetThumbnails(pr),
		}
	}

	if rs, ok := itemMap["reelShelfRenderer"].(map[string]interface{}); ok {
		var shorts []map[string]interface{}
		if items, ok := rs["items"].([]interface{}); ok {
			for _, item := range items {
				if im, ok := item.(map[string]interface{}); ok {
					if ri, ok := im["reelItemRenderer"].(map[string]interface{}); ok {
						shorts = append(shorts, map[string]interface{}{
							"type":       "short",
							"video_id":   internal.GetString(ri, "videoId"),
							"headline":   internal.GetTextFromSimpleText(ri, "headline"),
							"views":      internal.GetTextFromSimpleText(ri, "viewCountText"),
							"thumbnails": internal.GetThumbnails(ri),
						})
					}
				}
			}
		}
		if len(shorts) > 0 {
			return &types.SearchResultItem{
				Type:   "shorts_shelf",
				Title:  internal.GetTextFromRuns(rs, "title"),
				Shorts: shorts,
			}
		}
	}

	if rr, ok := itemMap["radioRenderer"].(map[string]interface{}); ok {
		return &types.SearchResultItem{
			Type:       "radio",
			PlaylistID: internal.GetString(rr, "playlistId"),
			Title:      internal.GetTextFromRuns(rr, "title"),
			VideoCount: internal.GetString(rr, "videoCountText"),
			Thumbnails: internal.GetThumbnails(rr),
		}
	}

	return nil
}
