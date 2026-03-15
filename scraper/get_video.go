package scraper

import (
	"context"
	"fmt"
	"strings"

	"github.com/embedtools/youtube-scraper/internal"
	"github.com/embedtools/youtube-scraper/types"
)

// GetVideo fetches metadata for a single YouTube video.
func (c *Client) GetVideo(ctx context.Context, in *types.GetVideoInput) (*types.GetVideoOutput, error) {
	if ctx.Err() != nil {
		return nil, ErrContextCanceled
	}

	videoID := internal.ExtractVideoID(in.URL)
	if videoID == "" {
		return nil, fmt.Errorf("%w: could not extract video ID from %q", ErrInvalidURL, in.URL)
	}

	// Fetch player and next data in parallel
	type result struct {
		data map[string]interface{}
		err  error
	}
	playerCh := make(chan result, 1)
	nextCh := make(chan result, 1)

	go func() {
		data, err := c.yt.Player(ctx, videoID)
		playerCh <- result{data, err}
	}()
	go func() {
		data, err := c.yt.Next(ctx, videoID)
		nextCh <- result{data, err}
	}()

	playerResult := <-playerCh
	nextResult := <-nextCh

	if playerResult.err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstreamChanged, playerResult.err)
	}

	// Check playability status
	playerBlocked := false
	if ps, ok := playerResult.data["playabilityStatus"].(map[string]interface{}); ok {
		status := internal.GetString(ps, "status")
		reason := internal.GetString(ps, "reason")
		switch {
		case status == "ERROR":
			if strings.Contains(strings.ToLower(reason), "private") {
				return nil, fmt.Errorf("%w: %s", ErrPrivateResource, reason)
			}
			// If next endpoint also failed, it's truly not found
			if nextResult.err != nil {
				return nil, fmt.Errorf("%w: %s", ErrNotFound, reason)
			}
			playerBlocked = true
		case status == "UNPLAYABLE":
			if _, hasDetails := playerResult.data["videoDetails"]; !hasDetails {
				if strings.Contains(strings.ToLower(reason), "private") {
					return nil, fmt.Errorf("%w: %s", ErrPrivateResource, reason)
				}
				// Player blocked but next might still have data
				playerBlocked = true
			}
		case status == "LOGIN_REQUIRED":
			if _, hasDetails := playerResult.data["videoDetails"]; !hasDetails {
				// Player blocked but next might still have data
				playerBlocked = true
			}
		}
	}

	// If player is blocked but next endpoint works, build output from next data only
	if playerBlocked {
		if nextResult.err != nil || nextResult.data == nil {
			return nil, fmt.Errorf("%w: login required", ErrBlocked)
		}
		out := buildVideoOutput(nil, nextResult.data, videoID)
		if out.Title == "" {
			return nil, fmt.Errorf("%w: login required", ErrBlocked)
		}
		return out, nil
	}

	out := buildVideoOutput(playerResult.data, nextResult.data, videoID)
	return out, nil
}

func buildVideoOutput(playerData, nextData map[string]interface{}, videoID string) *types.GetVideoOutput {
	out := &types.GetVideoOutput{
		VideoID: videoID,
		URL:     fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID),
	}

	if playerData == nil {
		// Player was blocked — extract what we can from next data only
		if nextData != nil {
			extractEngagement(nextData, out)
		}
		return out
	}

	// videoDetails
	if vd, ok := playerData["videoDetails"].(map[string]interface{}); ok {
		out.Title = internal.GetString(vd, "title")
		out.Description = internal.GetString(vd, "shortDescription")
		out.ChannelID = internal.GetString(vd, "channelId")
		out.ChannelName = internal.GetString(vd, "author")
		out.DurationSeconds = internal.GetInt(vd, "lengthSeconds")
		out.ViewCount = internal.GetInt(vd, "viewCount")
		out.IsLive = internal.GetBool(vd, "isLiveContent")
		out.IsPrivate = internal.GetBool(vd, "isPrivate")
		out.IsUnlisted = internal.GetBool(vd, "isUnlisted")
		if kw, ok := vd["keywords"].([]interface{}); ok {
			out.Keywords = kw
		}
		out.Thumbnails = internal.GetThumbnails(vd)
	}

	// microformat
	if mf, ok := playerData["microformat"].(map[string]interface{}); ok {
		if pm, ok := mf["playerMicroformatRenderer"].(map[string]interface{}); ok {
			out.PublishDate = internal.GetString(pm, "publishDate")
			out.UploadDate = internal.GetString(pm, "uploadDate")
			out.Category = internal.GetString(pm, "category")
			out.IsFamilySafe = internal.GetBool(pm, "isFamilySafe")
			out.IsUnlisted = internal.GetBool(pm, "isUnlisted")
			if c, ok := pm["availableCountries"].([]interface{}); ok {
				out.AvailableCountries = c
			}
			if embed, ok := pm["embed"].(map[string]interface{}); ok {
				out.EmbedURL = internal.GetString(embed, "iframeUrl")
			}
			if lb, ok := pm["liveBroadcastDetails"].(map[string]interface{}); ok {
				out.LiveBroadcast = lb
			}
		}
	}

	// streamingData
	if sd, ok := playerData["streamingData"].(map[string]interface{}); ok {
		if f, ok := sd["formats"].([]interface{}); ok {
			out.Formats = f
		}
		if af, ok := sd["adaptiveFormats"].([]interface{}); ok {
			out.AdaptiveFormats = af
		}
		out.ExpiresInSeconds = internal.GetString(sd, "expiresInSeconds")
	}

	// captions
	if captions, ok := playerData["captions"].(map[string]interface{}); ok {
		if pc, ok := captions["playerCaptionsTracklistRenderer"].(map[string]interface{}); ok {
			if ct, ok := pc["captionTracks"].([]interface{}); ok {
				out.CaptionTracks = ct
			}
			if tl, ok := pc["translationLanguages"].([]interface{}); ok {
				out.TranslationLangsCnt = len(tl)
			}
		}
	}

	// storyboards
	if sb, ok := playerData["storyboards"].(map[string]interface{}); ok {
		if spec, ok := sb["playerStoryboardSpecRenderer"].(map[string]interface{}); ok {
			out.StoryboardSpec = internal.GetString(spec, "spec")
		}
	}

	// engagement data from next
	if nextData != nil {
		extractEngagement(nextData, out)
	}

	return out
}

func extractEngagement(nextData map[string]interface{}, out *types.GetVideoOutput) {
	contents, ok := nextData["contents"].(map[string]interface{})
	if !ok {
		return
	}
	twoColumn, ok := contents["twoColumnWatchNextResults"].(map[string]interface{})
	if !ok {
		return
	}

	// Primary + secondary info
	if results, ok := twoColumn["results"].(map[string]interface{}); ok {
		if rr, ok := results["results"].(map[string]interface{}); ok {
			if cl, ok := rr["contents"].([]interface{}); ok {
				for _, item := range cl {
					if im, ok := item.(map[string]interface{}); ok {
						if vp, ok := im["videoPrimaryInfoRenderer"].(map[string]interface{}); ok {
							out.Title = internal.GetTextFromRuns(vp, "title")
							if vc, ok := vp["viewCount"].(map[string]interface{}); ok {
								if vvc, ok := vc["videoViewCountRenderer"].(map[string]interface{}); ok {
									out.ViewCountText = internal.GetTextFromSimpleText(vvc, "viewCount")
									out.ShortViewCount = internal.GetTextFromSimpleText(vvc, "shortViewCount")
									out.IsLive = internal.GetBool(vvc, "isLive")
								}
							}
							if dt, ok := vp["dateText"].(map[string]interface{}); ok {
								out.DateText = internal.GetString(dt, "simpleText")
							}
							extractLikes(vp, out)
						}
						if vs, ok := im["videoSecondaryInfoRenderer"].(map[string]interface{}); ok {
							if owner, ok := vs["owner"].(map[string]interface{}); ok {
								if vo, ok := owner["videoOwnerRenderer"].(map[string]interface{}); ok {
									out.ChannelName = internal.GetTextFromRuns(vo, "title")
									if sc, ok := vo["subscriberCountText"].(map[string]interface{}); ok {
										out.ChannelSubscriberCnt = internal.GetString(sc, "simpleText")
									}
									out.ChannelThumbnail = internal.GetThumbnails(vo)
									if ne, ok := vo["navigationEndpoint"].(map[string]interface{}); ok {
										if be, ok := ne["browseEndpoint"].(map[string]interface{}); ok {
											out.ChannelID = internal.GetString(be, "browseId")
											out.ChannelURL = internal.GetString(be, "canonicalBaseUrl")
										}
									}
								}
							}
							if desc, ok := vs["attributedDescription"].(map[string]interface{}); ok {
								out.FullDescription = internal.GetString(desc, "content")
							}
						}
					}
				}
			}
		}
	}

	// Related videos
	if sr, ok := twoColumn["secondaryResults"].(map[string]interface{}); ok {
		if srr, ok := sr["secondaryResults"].(map[string]interface{}); ok {
			if rl, ok := srr["results"].([]interface{}); ok {
				for _, item := range rl {
					if im, ok := item.(map[string]interface{}); ok {
						if cv, ok := im["compactVideoRenderer"].(map[string]interface{}); ok {
							out.RelatedVideos = append(out.RelatedVideos, types.RelatedVideo{
								VideoID:   internal.GetString(cv, "videoId"),
								Title:     internal.GetTextFromRuns(cv, "title"),
								Channel:   internal.GetTextFromRuns(cv, "longBylineText"),
								Views:     internal.GetTextFromSimpleText(cv, "viewCountText"),
								Duration:  internal.GetTextFromSimpleText(cv, "lengthText"),
								Published: internal.GetTextFromSimpleText(cv, "publishedTimeText"),
							})
						}
					}
				}
			}
		}
	}

	// Comments continuation
	if panels, ok := nextData["engagementPanels"].([]interface{}); ok {
		for _, panel := range panels {
			if pm, ok := panel.(map[string]interface{}); ok {
				if sl, ok := pm["engagementPanelSectionListRenderer"].(map[string]interface{}); ok {
					if internal.GetString(sl, "panelIdentifier") == "comment-item-section" {
						if content, ok := sl["content"].(map[string]interface{}); ok {
							if slr, ok := content["sectionListRenderer"].(map[string]interface{}); ok {
								if cs, ok := slr["contents"].([]interface{}); ok {
									for _, c := range cs {
										if cm, ok := c.(map[string]interface{}); ok {
											if is, ok := cm["itemSectionRenderer"].(map[string]interface{}); ok {
												if sc, ok := is["contents"].([]interface{}); ok {
													for _, s := range sc {
														if sm, ok := s.(map[string]interface{}); ok {
															if cr, ok := sm["continuationItemRenderer"].(map[string]interface{}); ok {
																out.CommentsContinuation = internal.ExtractContinuationToken(cr)
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
					}
				}
			}
		}
	}
}

func extractLikes(vp map[string]interface{}, out *types.GetVideoOutput) {
	va, ok := vp["videoActions"].(map[string]interface{})
	if !ok {
		return
	}
	mr, ok := va["menuRenderer"].(map[string]interface{})
	if !ok {
		return
	}
	tlb, ok := mr["topLevelButtons"].([]interface{})
	if !ok {
		return
	}
	for _, btn := range tlb {
		bm, ok := btn.(map[string]interface{})
		if !ok {
			continue
		}
		if seg, ok := bm["segmentedLikeDislikeButtonViewModel"].(map[string]interface{}); ok {
			if lb, ok := seg["likeButtonViewModel"].(map[string]interface{}); ok {
				if lbvm, ok := lb["likeButtonViewModel"].(map[string]interface{}); ok {
					if tb, ok := lbvm["toggleButtonViewModel"].(map[string]interface{}); ok {
						if tbvm, ok := tb["toggleButtonViewModel"].(map[string]interface{}); ok {
							if db, ok := tbvm["defaultButtonViewModel"].(map[string]interface{}); ok {
								if bvm, ok := db["buttonViewModel"].(map[string]interface{}); ok {
									out.LikeCountText = internal.GetString(bvm, "title")
									out.LikeCountA11y = internal.GetString(bvm, "accessibilityText")
								}
							}
						}
					}
				}
			}
		}
		if tb, ok := bm["toggleButtonRenderer"].(map[string]interface{}); ok {
			if di, ok := tb["defaultIcon"].(map[string]interface{}); ok {
				if internal.GetString(di, "iconType") == "LIKE" {
					if dt, ok := tb["defaultText"].(map[string]interface{}); ok {
						out.LikeCountText = internal.GetString(dt, "simpleText")
						if a11y, ok := dt["accessibility"].(map[string]interface{}); ok {
							if a11d, ok := a11y["accessibilityData"].(map[string]interface{}); ok {
								out.LikeCountA11y = internal.GetString(a11d, "label")
							}
						}
					}
				}
			}
		}
	}
}
