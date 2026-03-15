package scraper

import (
	"context"
	"fmt"
	"strings"

	"github.com/embedtools/youtube-scraper/internal"
	"github.com/embedtools/youtube-scraper/types"
)

// GetChannel fetches metadata for a YouTube channel.
func (c *Client) GetChannel(ctx context.Context, in *types.GetChannelInput) (*types.GetChannelOutput, error) {
	if ctx.Err() != nil {
		return nil, ErrContextCanceled
	}

	channelID, err := internal.ResolveChannelID(ctx, c.yt, in.URL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotFound, err)
	}

	browseData, err := c.yt.Browse(ctx, channelID, "")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstreamChanged, err)
	}

	// Check for errors
	if alerts, ok := browseData["alerts"].([]interface{}); ok {
		for _, alert := range alerts {
			if am, ok := alert.(map[string]interface{}); ok {
				if ar, ok := am["alertRenderer"].(map[string]interface{}); ok {
					if internal.GetString(ar, "type") == "ERROR" {
						return nil, fmt.Errorf("%w: channel %s", ErrNotFound, channelID)
					}
				}
			}
		}
	}

	out := &types.GetChannelOutput{
		ChannelID: channelID,
		URL:       "https://www.youtube.com/channel/" + channelID,
	}

	// metadata
	if metadata, ok := browseData["metadata"].(map[string]interface{}); ok {
		if cm, ok := metadata["channelMetadataRenderer"].(map[string]interface{}); ok {
			out.Title = internal.GetString(cm, "title")
			out.Description = internal.GetString(cm, "description")
			out.VanityURL = internal.GetString(cm, "vanityChannelUrl")
			out.RssURL = internal.GetString(cm, "rssUrl")
			out.ExternalID = internal.GetString(cm, "externalId")
			out.IsFamilySafe = internal.GetBool(cm, "isFamilySafe")
			if kw, ok := cm["keywords"].(string); ok {
				out.Keywords = kw
			}
			if av, ok := cm["avatar"].(map[string]interface{}); ok {
				if t, ok := av["thumbnails"].([]interface{}); ok {
					out.Avatar = t
				}
			}
			if cc, ok := cm["availableCountryCodes"].([]interface{}); ok {
				out.AvailableCountries = cc
			}
		}
	}

	// header
	if header, ok := browseData["header"].(map[string]interface{}); ok {
		extractChannelHeader(header, out)
	}

	// tabs
	if contents, ok := browseData["contents"].(map[string]interface{}); ok {
		if tcb, ok := contents["twoColumnBrowseResultsRenderer"].(map[string]interface{}); ok {
			if tabs, ok := tcb["tabs"].([]interface{}); ok {
				for _, tab := range tabs {
					if tm, ok := tab.(map[string]interface{}); ok {
						if tr, ok := tm["tabRenderer"].(map[string]interface{}); ok {
							out.AvailableTabs = append(out.AvailableTabs, internal.GetString(tr, "title"))
						}
					}
				}
			}
		}
	}

	return out, nil
}

func extractChannelHeader(header map[string]interface{}, out *types.GetChannelOutput) {
	// New: pageHeaderRenderer
	if phr, ok := header["pageHeaderRenderer"].(map[string]interface{}); ok {
		if phvm, ok := phr["pageHeaderViewModel"].(map[string]interface{}); ok {
			if title, ok := phvm["title"].(map[string]interface{}); ok {
				if dt, ok := title["dynamicTextViewModel"].(map[string]interface{}); ok {
					if t, ok := dt["text"].(map[string]interface{}); ok {
						out.Title = internal.GetString(t, "content")
					}
				}
			}
			if desc, ok := phvm["description"].(map[string]interface{}); ok {
				if dpvm, ok := desc["descriptionPreviewViewModel"].(map[string]interface{}); ok {
					if d, ok := dpvm["description"].(map[string]interface{}); ok {
						out.Description = internal.GetString(d, "content")
					}
				}
			}
			if mr, ok := phvm["metadata"].(map[string]interface{}); ok {
				if cmvm, ok := mr["contentMetadataViewModel"].(map[string]interface{}); ok {
					if rows, ok := cmvm["metadataRows"].([]interface{}); ok {
						for _, row := range rows {
							if rm, ok := row.(map[string]interface{}); ok {
								if parts, ok := rm["metadataParts"].([]interface{}); ok {
									for _, part := range parts {
										if pm, ok := part.(map[string]interface{}); ok {
											if tm, ok := pm["text"].(map[string]interface{}); ok {
												content := internal.GetString(tm, "content")
												if strings.Contains(content, "subscribers") {
													out.SubscriberCountTxt = content
												} else if strings.Contains(content, "videos") {
													out.VideoCountText = content
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
			if banner, ok := phvm["banner"].(map[string]interface{}); ok {
				if ibvm, ok := banner["imageBannerViewModel"].(map[string]interface{}); ok {
					if image, ok := ibvm["image"].(map[string]interface{}); ok {
						if s, ok := image["sources"].([]interface{}); ok {
							out.Banner = s
						}
					}
				}
			}
			if image, ok := phvm["image"].(map[string]interface{}); ok {
				if da, ok := image["decoratedAvatarViewModel"].(map[string]interface{}); ok {
					if av, ok := da["avatar"].(map[string]interface{}); ok {
						if avm, ok := av["avatarViewModel"].(map[string]interface{}); ok {
							if ai, ok := avm["image"].(map[string]interface{}); ok {
								if s, ok := ai["sources"].([]interface{}); ok {
									out.Avatar = s
								}
							}
						}
					}
				}
			}
		}
	}

	// Old: c4TabbedHeaderRenderer
	if c4, ok := header["c4TabbedHeaderRenderer"].(map[string]interface{}); ok {
		out.ChannelID = internal.GetString(c4, "channelId")
		out.Title = internal.GetString(c4, "title")
		if sc, ok := c4["subscriberCountText"].(map[string]interface{}); ok {
			out.SubscriberCountTxt = internal.GetString(sc, "simpleText")
		}
		out.VideoCountText = internal.GetTextFromRuns(c4, "videosCountText")
		if av, ok := c4["avatar"].(map[string]interface{}); ok {
			if t, ok := av["thumbnails"].([]interface{}); ok {
				out.Avatar = t
			}
		}
		if b, ok := c4["banner"].(map[string]interface{}); ok {
			if t, ok := b["thumbnails"].([]interface{}); ok {
				out.Banner = t
			}
		}
		if tb, ok := c4["tvBanner"].(map[string]interface{}); ok {
			if t, ok := tb["thumbnails"].([]interface{}); ok {
				out.TvBanner = t
			}
		}
		if mb, ok := c4["mobileBanner"].(map[string]interface{}); ok {
			if t, ok := mb["thumbnails"].([]interface{}); ok {
				out.MobileBanner = t
			}
		}
		out.ChannelHandle = internal.GetString(c4, "channelHandleText")
	}
}
