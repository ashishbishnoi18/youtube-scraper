package types

// GetVideoInput is the input for the GetVideo capability.
type GetVideoInput struct {
	URL    string   `json:"url"`
	Fields []string `json:"fields,omitempty"`
}

// GetChannelInput is the input for the GetChannel capability.
type GetChannelInput struct {
	URL    string   `json:"url"`
	Fields []string `json:"fields,omitempty"`
}

// ListChannelVideosInput is the input for the ListChannelVideos streaming capability.
type ListChannelVideosInput struct {
	URL      string `json:"url"`
	Limit    int    `json:"limit,omitempty"`
	MaxPages int    `json:"max_pages,omitempty"`
}

// ListChannelShortsInput is the input for the ListChannelShorts streaming capability.
type ListChannelShortsInput struct {
	URL      string `json:"url"`
	Limit    int    `json:"limit,omitempty"`
	MaxPages int    `json:"max_pages,omitempty"`
}

// ListChannelPlaylistsInput is the input for the ListChannelPlaylists streaming capability.
type ListChannelPlaylistsInput struct {
	URL      string `json:"url"`
	Limit    int    `json:"limit,omitempty"`
	MaxPages int    `json:"max_pages,omitempty"`
}

// ListChannelCommunityInput is the input for the ListChannelCommunity streaming capability.
type ListChannelCommunityInput struct {
	URL      string `json:"url"`
	Limit    int    `json:"limit,omitempty"`
	MaxPages int    `json:"max_pages,omitempty"`
}

// ListCommentsInput is the input for the ListComments streaming capability.
type ListCommentsInput struct {
	URL      string `json:"url"`
	Limit    int    `json:"limit,omitempty"`
	MaxPages int    `json:"max_pages,omitempty"`
}

// SearchInput is the input for the Search capability.
type SearchInput struct {
	Query    string `json:"query"`
	Filter   string `json:"filter,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	MaxPages int    `json:"max_pages,omitempty"`
}

// GetPlaylistInput is the input for the GetPlaylist capability.
type GetPlaylistInput struct {
	URL      string   `json:"url"`
	Fields   []string `json:"fields,omitempty"`
	Limit    int      `json:"limit,omitempty"`
	MaxPages int      `json:"max_pages,omitempty"`
}

// GetTranscriptInput is the input for the GetTranscript capability.
type GetTranscriptInput struct {
	URL      string `json:"url"`
	Language string `json:"language,omitempty"`
}
