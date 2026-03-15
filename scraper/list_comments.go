package scraper

import (
	"context"
	"fmt"
	"strings"

	"github.com/embedtools/youtube-scraper/internal"
	"github.com/embedtools/youtube-scraper/types"
)

// ListComments streams comments for a YouTube video.
func (c *Client) ListComments(ctx context.Context, in *types.ListCommentsInput, emit func(item *types.CommentItem) error) (*types.CommentsSummary, error) {
	if ctx.Err() != nil {
		return nil, ErrContextCanceled
	}

	videoID := internal.ExtractVideoID(in.URL)
	if videoID == "" {
		return nil, fmt.Errorf("%w: could not extract video ID from %q", ErrInvalidURL, in.URL)
	}

	nextData, err := c.yt.Next(ctx, videoID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUpstreamChanged, err)
	}

	continuation := extractCommentsContinuation(nextData)
	if continuation == "" {
		return &types.CommentsSummary{
			VideoID:    videoID,
			TotalItems: 0,
			Pages:      0,
		}, nil
	}

	totalEmitted := 0
	pageCount := 0
	maxPages := in.MaxPages
	if maxPages == 0 {
		maxPages = 100
	}

	for continuation != "" && pageCount < maxPages {
		if in.Limit > 0 && totalEmitted >= in.Limit {
			break
		}

		commentsData, err := c.yt.NextWithContinuation(ctx, continuation)
		if err != nil {
			break
		}

		comments, nextCont := extractCommentsFromResponse(commentsData)
		for _, comment := range comments {
			if in.Limit > 0 && totalEmitted >= in.Limit {
				break
			}
			if err := emit(comment); err != nil {
				return nil, err
			}
			totalEmitted++
		}
		continuation = nextCont
		pageCount++
	}

	return &types.CommentsSummary{
		VideoID:    videoID,
		TotalItems: totalEmitted,
		Pages:      pageCount,
	}, nil
}

// extractCommentsContinuation extracts the initial comments continuation token.
func extractCommentsContinuation(nextData map[string]interface{}) string {
	panels, ok := nextData["engagementPanels"].([]interface{})
	if !ok {
		return ""
	}
	for _, panel := range panels {
		pm, ok := panel.(map[string]interface{})
		if !ok {
			continue
		}
		sl, ok := pm["engagementPanelSectionListRenderer"].(map[string]interface{})
		if !ok {
			continue
		}
		if internal.GetString(sl, "panelIdentifier") != "engagement-panel-comments-section" {
			continue
		}
		content, ok := sl["content"].(map[string]interface{})
		if !ok {
			continue
		}
		slr, ok := content["sectionListRenderer"].(map[string]interface{})
		if !ok {
			continue
		}
		contents, ok := slr["contents"].([]interface{})
		if !ok {
			continue
		}
		for _, c := range contents {
			cm, ok := c.(map[string]interface{})
			if !ok {
				continue
			}
			is, ok := cm["itemSectionRenderer"].(map[string]interface{})
			if !ok {
				continue
			}
			sc, ok := is["contents"].([]interface{})
			if !ok {
				continue
			}
			for _, s := range sc {
				sm, ok := s.(map[string]interface{})
				if !ok {
					continue
				}
				if cr, ok := sm["continuationItemRenderer"].(map[string]interface{}); ok {
					return internal.ExtractContinuationToken(cr)
				}
			}
		}
	}
	return ""
}

// extractCommentsFromResponse extracts comments from a continuation response.
func extractCommentsFromResponse(data map[string]interface{}) ([]*types.CommentItem, string) {
	var comments []*types.CommentItem
	var continuation string

	// Build comment data map from frameworkUpdates
	commentDataMap := make(map[string]*types.CommentItem)

	if fu, ok := data["frameworkUpdates"].(map[string]interface{}); ok {
		if ebu, ok := fu["entityBatchUpdate"].(map[string]interface{}); ok {
			if mutations, ok := ebu["mutations"].([]interface{}); ok {
				for _, mutation := range mutations {
					mm, ok := mutation.(map[string]interface{})
					if !ok {
						continue
					}
					payload, ok := mm["payload"].(map[string]interface{})
					if !ok {
						continue
					}
					cp, ok := payload["commentEntityPayload"].(map[string]interface{})
					if !ok {
						continue
					}
					item := extractCommentFromPayload(cp)
					if item != nil && item.CommentID != "" {
						commentDataMap[item.CommentID] = item
					}
				}
			}
		}
	}

	// Extract threads from onResponseReceivedEndpoints
	endpoints, ok := data["onResponseReceivedEndpoints"].([]interface{})
	if !ok {
		return comments, ""
	}

	for _, endpoint := range endpoints {
		em, ok := endpoint.(map[string]interface{})
		if !ok {
			continue
		}

		// Process both reloadContinuationItemsCommand and appendContinuationItemsAction
		for _, key := range []string{"reloadContinuationItemsCommand", "appendContinuationItemsAction"} {
			action, ok := em[key].(map[string]interface{})
			if !ok {
				continue
			}
			contItems, ok := action["continuationItems"].([]interface{})
			if !ok {
				continue
			}
			for _, item := range contItems {
				im, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				if thread, ok := im["commentThreadRenderer"].(map[string]interface{}); ok {
					comment := extractCommentThread(thread, commentDataMap)
					if comment != nil {
						comments = append(comments, comment)
					}
				}
				if cr, ok := im["continuationItemRenderer"].(map[string]interface{}); ok {
					continuation = internal.ExtractContinuationFromRenderer(cr)
				}
			}
		}
	}

	return comments, continuation
}

func extractCommentFromPayload(payload map[string]interface{}) *types.CommentItem {
	item := &types.CommentItem{}

	if props, ok := payload["properties"].(map[string]interface{}); ok {
		item.CommentID = internal.GetString(props, "commentId")
		item.Published = internal.GetString(props, "publishedTime")
		if content, ok := props["content"].(map[string]interface{}); ok {
			item.Content = internal.GetString(content, "content")
		}
	}

	if author, ok := payload["author"].(map[string]interface{}); ok {
		item.Author = internal.GetString(author, "displayName")
		item.AuthorChannelID = internal.GetString(author, "channelId")
		item.AuthorThumbnail = internal.GetString(author, "avatarThumbnailUrl")
		item.IsVerified = internal.GetBool(author, "isVerified")
		item.IsCreator = internal.GetBool(author, "isCreator")
	}

	if toolbar, ok := payload["toolbar"].(map[string]interface{}); ok {
		item.LikeCount = internal.GetString(toolbar, "likeCountNotliked")
		item.ReplyCount = internal.GetString(toolbar, "replyCount")
		if heartTooltip := internal.GetString(toolbar, "heartActiveTooltip"); heartTooltip != "" {
			item.IsHearted = true
			item.HeartedBy = heartTooltip
		}
	}

	return item
}

func extractCommentThread(thread map[string]interface{}, dataMap map[string]*types.CommentItem) *types.CommentItem {
	var item *types.CommentItem

	// New structure: commentViewModel
	if vmOuter, ok := thread["commentViewModel"].(map[string]interface{}); ok {
		var commentID string
		if vmInner, ok := vmOuter["commentViewModel"].(map[string]interface{}); ok {
			commentID = internal.GetString(vmInner, "commentId")
		}
		if commentID == "" {
			commentID = internal.GetString(vmOuter, "commentId")
		}
		if commentID != "" {
			if data, exists := dataMap[commentID]; exists {
				item = data
			} else {
				item = &types.CommentItem{CommentID: commentID}
			}
		}
	}

	// Old structure fallback: comment.commentRenderer
	if comment, ok := thread["comment"].(map[string]interface{}); ok {
		if cr, ok := comment["commentRenderer"].(map[string]interface{}); ok {
			if item == nil {
				item = &types.CommentItem{}
			}
			fillFromCommentRenderer(cr, item)
		}
	}

	if item == nil {
		return nil
	}

	// Extract replies info
	if replies, ok := thread["replies"].(map[string]interface{}); ok {
		if crr, ok := replies["commentRepliesRenderer"].(map[string]interface{}); ok {
			if vr, ok := crr["viewReplies"].(map[string]interface{}); ok {
				if br, ok := vr["buttonRenderer"].(map[string]interface{}); ok {
					item.ReplyCountText = internal.GetTextFromRuns(br, "text")
				}
			}
			if contents, ok := crr["contents"].([]interface{}); ok {
				for _, c := range contents {
					if cm, ok := c.(map[string]interface{}); ok {
						if cr, ok := cm["continuationItemRenderer"].(map[string]interface{}); ok {
							item.RepliesContinuation = internal.ExtractContinuationFromRenderer(cr)
						}
					}
				}
			}
		}
	}

	return item
}

func fillFromCommentRenderer(r map[string]interface{}, item *types.CommentItem) {
	if item.CommentID == "" {
		item.CommentID = internal.GetString(r, "commentId")
	}
	if item.Content == "" {
		item.Content = internal.GetTextFromRuns(r, "contentText")
	}
	if item.Published == "" {
		item.Published = internal.GetTextFromRuns(r, "publishedTimeText")
	}
	if item.LikeCount == "" {
		if vc, ok := r["voteCount"].(map[string]interface{}); ok {
			item.LikeCount = internal.GetString(vc, "simpleText")
		}
	}
	if item.Author == "" {
		if at, ok := r["authorText"].(map[string]interface{}); ok {
			item.Author = internal.GetString(at, "simpleText")
		}
	}
	if item.AuthorChannelID == "" {
		if ae, ok := r["authorEndpoint"].(map[string]interface{}); ok {
			if be, ok := ae["browseEndpoint"].(map[string]interface{}); ok {
				item.AuthorChannelID = internal.GetString(be, "browseId")
			}
		}
	}
	if at, ok := r["authorThumbnail"].(map[string]interface{}); ok {
		if thumbs, ok := at["thumbnails"].([]interface{}); ok {
			item.AuthorThumbnails = thumbs
		}
	}
	if !item.IsHearted {
		if ab, ok := r["actionButtons"].(map[string]interface{}); ok {
			if cab, ok := ab["commentActionButtonsRenderer"].(map[string]interface{}); ok {
				if ch, ok := cab["creatorHeart"].(map[string]interface{}); ok {
					if _, ok := ch["creatorHeartRenderer"].(map[string]interface{}); ok {
						item.IsHearted = true
					}
				}
			}
		}
	}
	if _, ok := r["pinnedCommentBadge"]; ok {
		item.IsPinned = true
	}
	if rc, ok := r["replyCount"].(float64); ok {
		item.ReplyCount = fmt.Sprintf("%d", int(rc))
	}

	// Check for is_pinned using the string representation
	if strings.Contains(internal.GetString(r, "pinnedCommentBadge"), "PINNED") {
		item.IsPinned = true
	}
}
