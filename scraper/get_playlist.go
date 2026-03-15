package scraper

import (
	"context"
	"fmt"

	"github.com/embedtools/youtube-scraper/internal"
	"github.com/embedtools/youtube-scraper/types"
)

// GetPlaylist fetches metadata and videos for a YouTube playlist.
func (c *Client) GetPlaylist(ctx context.Context, in *types.GetPlaylistInput) (*types.GetPlaylistOutput, error) {
	if ctx.Err() != nil {
		return nil, ErrContextCanceled
	}

	playlistID := internal.ExtractPlaylistID(in.URL)
	if playlistID == "" {
		return nil, fmt.Errorf("%w: could not extract playlist ID from %q", ErrInvalidURL, in.URL)
	}

	browseID := "VL" + playlistID
	browseData, err := c.yt.Browse(ctx, browseID, "")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstreamChanged, err)
	}

	// Check for errors
	if alerts, ok := browseData["alerts"].([]interface{}); ok {
		for _, alert := range alerts {
			if am, ok := alert.(map[string]interface{}); ok {
				if ar, ok := am["alertRenderer"].(map[string]interface{}); ok {
					if internal.GetString(ar, "type") == "ERROR" {
						return nil, fmt.Errorf("%w: playlist %s", ErrNotFound, playlistID)
					}
				}
			}
		}
	}

	out := extractPlaylistMetadata(browseData, playlistID)

	videos, continuation := extractPlaylistVideos(browseData)

	pageCount := 1
	maxPages := in.MaxPages
	if maxPages == 0 {
		maxPages = 100
	}

	for continuation != "" && pageCount < maxPages {
		if in.Limit > 0 && len(videos) >= in.Limit {
			break
		}

		contData, err := c.yt.BrowseWithContinuation(ctx, continuation)
		if err != nil {
			break
		}

		moreVideos, nextCont := extractPlaylistContinuationVideos(contData)
		videos = append(videos, moreVideos...)
		continuation = nextCont
		pageCount++
	}

	if in.Limit > 0 && len(videos) > in.Limit {
		videos = videos[:in.Limit]
	}

	out.Videos = videos
	out.VideoCount = len(videos)
	out.Pages = pageCount

	return out, nil
}

func extractPlaylistMetadata(browseData map[string]interface{}, playlistID string) *types.GetPlaylistOutput {
	out := &types.GetPlaylistOutput{
		PlaylistID: playlistID,
		URL:        "https://www.youtube.com/playlist?list=" + playlistID,
	}

	// Header
	if header, ok := browseData["header"].(map[string]interface{}); ok {
		if ph, ok := header["playlistHeaderRenderer"].(map[string]interface{}); ok {
			out.Title = internal.GetTextFromRuns(ph, "title")
			out.Description = internal.GetTextFromRuns(ph, "descriptionText")
			out.Owner = internal.GetTextFromRuns(ph, "ownerText")
			out.OwnerChannelID = internal.ExtractChannelIDFromOwner(ph)
			out.VideoCountText = internal.GetTextFromRuns(ph, "numVideosText")
			if vc, ok := ph["viewCountText"].(map[string]interface{}); ok {
				out.ViewCountText = internal.GetString(vc, "simpleText")
			}
			out.Privacy = internal.GetString(ph, "privacy")
			out.IsEditable = internal.GetBool(ph, "isEditable")
		}
	}

	// Metadata
	if metadata, ok := browseData["metadata"].(map[string]interface{}); ok {
		if pm, ok := metadata["playlistMetadataRenderer"].(map[string]interface{}); ok {
			if out.Title == "" {
				out.Title = internal.GetString(pm, "title")
			}
			if out.Description == "" {
				out.Description = internal.GetString(pm, "description")
			}
		}
	}

	// Sidebar
	if sidebar, ok := browseData["sidebar"].(map[string]interface{}); ok {
		if ps, ok := sidebar["playlistSidebarRenderer"].(map[string]interface{}); ok {
			if items, ok := ps["items"].([]interface{}); ok {
				for _, item := range items {
					im, ok := item.(map[string]interface{})
					if !ok {
						continue
					}
					if primary, ok := im["playlistSidebarPrimaryInfoRenderer"].(map[string]interface{}); ok {
						if stats, ok := primary["stats"].([]interface{}); ok {
							for i, stat := range stats {
								sm, ok := stat.(map[string]interface{})
								if !ok {
									continue
								}
								text := internal.GetTextFromRuns(sm, "")
								if text == "" {
									// Try runs directly on the stat map
									if runs, ok := sm["runs"].([]interface{}); ok {
										var t string
										for _, run := range runs {
											if rm, ok := run.(map[string]interface{}); ok {
												t += internal.GetString(rm, "text")
											}
										}
										text = t
									}
								}
								if text == "" {
									if st, ok := sm["simpleText"].(string); ok {
										text = st
									}
								}
								switch i {
								case 0:
									out.TotalVideosTxt = text
								case 1:
									out.TotalViewsTxt = text
								case 2:
									out.LastUpdated = text
								}
							}
						}

						// Thumbnails
						if tr, ok := primary["thumbnailRenderer"].(map[string]interface{}); ok {
							if pvt, ok := tr["playlistVideoThumbnailRenderer"].(map[string]interface{}); ok {
								out.Thumbnails = internal.GetThumbnails(pvt)
							}
							if pct, ok := tr["playlistCustomThumbnailRenderer"].(map[string]interface{}); ok {
								out.Thumbnails = internal.GetThumbnails(pct)
							}
						}
					}

					if secondary, ok := im["playlistSidebarSecondaryInfoRenderer"].(map[string]interface{}); ok {
						if vo, ok := secondary["videoOwner"].(map[string]interface{}); ok {
							if vor, ok := vo["videoOwnerRenderer"].(map[string]interface{}); ok {
								out.Owner = internal.GetTextFromRuns(vor, "title")
								if ne, ok := vor["navigationEndpoint"].(map[string]interface{}); ok {
									if be, ok := ne["browseEndpoint"].(map[string]interface{}); ok {
										out.OwnerChannelID = internal.GetString(be, "browseId")
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return out
}

func extractPlaylistVideos(browseData map[string]interface{}) ([]types.PlaylistVideo, string) {
	var videos []types.PlaylistVideo
	var continuation string

	contents, ok := browseData["contents"].(map[string]interface{})
	if !ok {
		return videos, ""
	}
	twoCol, ok := contents["twoColumnBrowseResultsRenderer"].(map[string]interface{})
	if !ok {
		return videos, ""
	}
	tabs, ok := twoCol["tabs"].([]interface{})
	if !ok {
		return videos, ""
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
		content, ok := tr["content"].(map[string]interface{})
		if !ok {
			continue
		}
		sl, ok := content["sectionListRenderer"].(map[string]interface{})
		if !ok {
			continue
		}
		sectionContents, ok := sl["contents"].([]interface{})
		if !ok {
			continue
		}
		for _, section := range sectionContents {
			sm, ok := section.(map[string]interface{})
			if !ok {
				continue
			}
			is, ok := sm["itemSectionRenderer"].(map[string]interface{})
			if !ok {
				continue
			}
			items, ok := is["contents"].([]interface{})
			if !ok {
				continue
			}
			for _, item := range items {
				im, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				pvl, ok := im["playlistVideoListRenderer"].(map[string]interface{})
				if !ok {
					continue
				}
				videoContents, ok := pvl["contents"].([]interface{})
				if !ok {
					continue
				}
				for _, vc := range videoContents {
					vcm, ok := vc.(map[string]interface{})
					if !ok {
						continue
					}
					if pvr, ok := vcm["playlistVideoRenderer"].(map[string]interface{}); ok {
						videos = append(videos, extractPlaylistVideoItem(pvr))
					}
					if cr, ok := vcm["continuationItemRenderer"].(map[string]interface{}); ok {
						continuation = internal.ExtractContinuationToken(cr)
					}
				}
			}
		}
	}

	return videos, continuation
}

func extractPlaylistContinuationVideos(contData map[string]interface{}) ([]types.PlaylistVideo, string) {
	var videos []types.PlaylistVideo
	var continuation string

	actions, ok := contData["onResponseReceivedActions"].([]interface{})
	if !ok {
		return videos, ""
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
			if pvr, ok := im["playlistVideoRenderer"].(map[string]interface{}); ok {
				videos = append(videos, extractPlaylistVideoItem(pvr))
			}
			if cr, ok := im["continuationItemRenderer"].(map[string]interface{}); ok {
				continuation = internal.ExtractContinuationToken(cr)
			}
		}
	}

	return videos, continuation
}

func extractPlaylistVideoItem(r map[string]interface{}) types.PlaylistVideo {
	return types.PlaylistVideo{
		VideoID:    internal.GetString(r, "videoId"),
		Title:      internal.GetTextFromRuns(r, "title"),
		Duration:   internal.GetTextFromSimpleText(r, "lengthText"),
		Channel:    internal.GetTextFromRuns(r, "shortBylineText"),
		ChannelID:  internal.ExtractChannelIDFromShortByline(r),
		Index:      internal.GetInt(r, "index"),
		Thumbnails: internal.GetThumbnails(r),
		IsPlayable: internal.GetBool(r, "isPlayable"),
	}
}
