package types

// Thumbnail represents a video/channel thumbnail.
type Thumbnail struct {
	URL    string `json:"url"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

// --- GetVideo ---

type GetVideoOutput struct {
	VideoID              string                 `json:"video_id"`
	URL                  string                 `json:"url"`
	Title                string                 `json:"title"`
	Description          string                 `json:"description"`
	FullDescription      string                 `json:"full_description,omitempty"`
	ChannelID            string                 `json:"channel_id"`
	ChannelName          string                 `json:"channel_name"`
	ChannelURL           string                 `json:"channel_url,omitempty"`
	ChannelSubscriberCnt string                 `json:"channel_subscriber_count,omitempty"`
	ChannelThumbnail     []interface{}          `json:"channel_thumbnail,omitempty"`
	DurationSeconds      int64                  `json:"duration_seconds"`
	ViewCount            int64                  `json:"view_count"`
	ViewCountText        string                 `json:"view_count_text,omitempty"`
	ShortViewCount       string                 `json:"short_view_count,omitempty"`
	LikeCountText        string                 `json:"like_count_text,omitempty"`
	LikeCountA11y        string                 `json:"like_count_a11y,omitempty"`
	DateText             string                 `json:"date_text,omitempty"`
	PublishDate          string                 `json:"publish_date,omitempty"`
	UploadDate           string                 `json:"upload_date,omitempty"`
	Category             string                 `json:"category,omitempty"`
	IsLive               bool                   `json:"is_live"`
	IsPrivate            bool                   `json:"is_private"`
	IsUnlisted           bool                   `json:"is_unlisted"`
	IsFamilySafe         bool                   `json:"is_family_safe"`
	Keywords             []interface{}          `json:"keywords,omitempty"`
	Thumbnails           []interface{}          `json:"thumbnails,omitempty"`
	AvailableCountries   []interface{}          `json:"available_countries,omitempty"`
	EmbedURL             string                 `json:"embed_url,omitempty"`
	Formats              []interface{}          `json:"formats,omitempty"`
	AdaptiveFormats      []interface{}          `json:"adaptive_formats,omitempty"`
	ExpiresInSeconds     string                 `json:"expires_in_seconds,omitempty"`
	CaptionTracks        []interface{}          `json:"caption_tracks,omitempty"`
	TranslationLangsCnt  int                    `json:"translation_languages_count,omitempty"`
	StoryboardSpec       string                 `json:"storyboard_spec,omitempty"`
	LiveBroadcast        map[string]interface{} `json:"live_broadcast,omitempty"`
	RelatedVideos        []RelatedVideo         `json:"related_videos,omitempty"`
	CommentsContinuation string                 `json:"comments_continuation,omitempty"`
}

type RelatedVideo struct {
	VideoID   string `json:"video_id"`
	Title     string `json:"title"`
	Channel   string `json:"channel"`
	Views     string `json:"views"`
	Duration  string `json:"duration"`
	Published string `json:"published"`
}

// --- GetChannel ---

type GetChannelOutput struct {
	ChannelID          string        `json:"channel_id"`
	URL                string        `json:"url"`
	Title              string        `json:"title"`
	Description        string        `json:"description"`
	VanityURL          string        `json:"vanity_url,omitempty"`
	RssURL             string        `json:"rss_url,omitempty"`
	ExternalID         string        `json:"external_id,omitempty"`
	IsFamilySafe       bool          `json:"is_family_safe"`
	Keywords           string        `json:"keywords,omitempty"`
	SubscriberCountTxt string        `json:"subscriber_count_text,omitempty"`
	VideoCountText     string        `json:"video_count_text,omitempty"`
	ChannelHandle      string        `json:"channel_handle,omitempty"`
	Avatar             []interface{} `json:"avatar,omitempty"`
	Banner             []interface{} `json:"banner,omitempty"`
	TvBanner           []interface{} `json:"tv_banner,omitempty"`
	MobileBanner       []interface{} `json:"mobile_banner,omitempty"`
	AvailableCountries []interface{} `json:"available_countries,omitempty"`
	AvailableTabs      []string      `json:"available_tabs,omitempty"`
}

// --- Channel Tab Items (streaming) ---

type ChannelTabItem struct {
	Type             string        `json:"type"`
	VideoID          string        `json:"video_id,omitempty"`
	PlaylistID       string        `json:"playlist_id,omitempty"`
	PostID           string        `json:"post_id,omitempty"`
	Title            string        `json:"title,omitempty"`
	Headline         string        `json:"headline,omitempty"`
	Description      string        `json:"description,omitempty"`
	Duration         string        `json:"duration,omitempty"`
	Views            string        `json:"views,omitempty"`
	Published        string        `json:"published,omitempty"`
	VoteCount        string        `json:"vote_count,omitempty"`
	Author           string        `json:"author,omitempty"`
	VideoCount       string        `json:"video_count,omitempty"`
	Content          string        `json:"content,omitempty"`
	AccessibilityTxt string        `json:"accessibility_text,omitempty"`
	EntityID         string        `json:"entity_id,omitempty"`
	Thumbnails       []interface{} `json:"thumbnails,omitempty"`
}

type ChannelTabSummary struct {
	ChannelID  string `json:"channel_id"`
	Tab        string `json:"tab"`
	TotalItems int    `json:"total_items"`
	Pages      int    `json:"pages"`
}

// --- Comments (streaming) ---

type CommentItem struct {
	CommentID          string        `json:"comment_id"`
	Content            string        `json:"content"`
	Published          string        `json:"published"`
	Author             string        `json:"author"`
	AuthorChannelID    string        `json:"author_channel_id,omitempty"`
	AuthorThumbnail    string        `json:"author_thumbnail,omitempty"`
	AuthorThumbnails   []interface{} `json:"author_thumbnails,omitempty"`
	IsVerified         bool          `json:"is_verified"`
	IsCreator          bool          `json:"is_creator"`
	IsHearted          bool          `json:"is_hearted"`
	IsPinned           bool          `json:"is_pinned"`
	HeartedBy          string        `json:"hearted_by,omitempty"`
	LikeCount          string        `json:"like_count,omitempty"`
	ReplyCount         string        `json:"reply_count,omitempty"`
	ReplyCountText     string        `json:"reply_count_text,omitempty"`
	RepliesContinuation string       `json:"replies_continuation,omitempty"`
}

type CommentsSummary struct {
	VideoID    string `json:"video_id"`
	TotalItems int    `json:"total_items"`
	Pages      int    `json:"pages"`
}

// --- Search ---

type SearchOutput struct {
	Query        string             `json:"query"`
	Results      []SearchResultItem `json:"results"`
	ResultsCount int                `json:"results_count"`
	Pages        int                `json:"pages"`
}

type SearchResultItem struct {
	Type            string                   `json:"type"`
	VideoID         string                   `json:"video_id,omitempty"`
	ChannelID       string                   `json:"channel_id,omitempty"`
	PlaylistID      string                   `json:"playlist_id,omitempty"`
	Title           string                   `json:"title"`
	Description     string                   `json:"description,omitempty"`
	Channel         string                   `json:"channel,omitempty"`
	Duration        string                   `json:"duration,omitempty"`
	Views           string                   `json:"views,omitempty"`
	Published       string                   `json:"published,omitempty"`
	SubscriberCount string                   `json:"subscriber_count,omitempty"`
	VideoCount      string                   `json:"video_count,omitempty"`
	IsLive          bool                     `json:"is_live,omitempty"`
	Thumbnails      []interface{}            `json:"thumbnails,omitempty"`
	Shorts          []map[string]interface{} `json:"shorts,omitempty"`
}

// --- GetPlaylist ---

type GetPlaylistOutput struct {
	PlaylistID     string          `json:"playlist_id"`
	URL            string          `json:"url"`
	Title          string          `json:"title"`
	Description    string          `json:"description,omitempty"`
	Owner          string          `json:"owner,omitempty"`
	OwnerChannelID string          `json:"owner_channel_id,omitempty"`
	VideoCountText string          `json:"video_count_text,omitempty"`
	ViewCountText  string          `json:"view_count_text,omitempty"`
	Privacy        string          `json:"privacy,omitempty"`
	IsEditable     bool            `json:"is_editable"`
	TotalVideosTxt string          `json:"total_videos_text,omitempty"`
	TotalViewsTxt  string          `json:"total_views_text,omitempty"`
	LastUpdated    string          `json:"last_updated,omitempty"`
	Thumbnails     []interface{}   `json:"thumbnails,omitempty"`
	Videos         []PlaylistVideo `json:"videos"`
	VideoCount     int             `json:"video_count"`
	Pages          int             `json:"pages"`
}

type PlaylistVideo struct {
	VideoID    string        `json:"video_id"`
	Title      string        `json:"title"`
	Duration   string        `json:"duration,omitempty"`
	Channel    string        `json:"channel,omitempty"`
	ChannelID  string        `json:"channel_id,omitempty"`
	Index      int64         `json:"index"`
	Thumbnails []interface{} `json:"thumbnails,omitempty"`
	IsPlayable bool          `json:"is_playable"`
}

// --- GetTranscript ---

type GetTranscriptOutput struct {
	VideoID            string              `json:"video_id"`
	Language           string              `json:"language"`
	LanguageName       string              `json:"language_name"`
	IsAutoGenerated    bool                `json:"is_auto_generated"`
	Transcript         []TranscriptSegment `json:"transcript"`
	SegmentCount       int                 `json:"segment_count"`
	AvailableLanguages []LanguageInfo      `json:"available_languages"`
	FullText           string              `json:"full_text"`
}

type TranscriptSegment struct {
	Text     string  `json:"text"`
	Start    float64 `json:"start"`
	Duration float64 `json:"duration"`
}

type LanguageInfo struct {
	LanguageCode  string `json:"language_code"`
	Name          string `json:"name"`
	VssID         string `json:"vss_id"`
	IsTranslatable bool  `json:"is_translatable"`
}
