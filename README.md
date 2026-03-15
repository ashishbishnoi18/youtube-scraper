# YouTube Scraper Module

A modulemaker-compliant YouTube scraper using the InnerTube API. Provides 10 capabilities for scraping videos, channels, playlists, comments, search, and transcripts.

## Capabilities

| ID | Method | Type | Description |
|----|--------|------|-------------|
| `youtube.video.get` | `GetVideo` | sync | Video metadata, streams, engagement |
| `youtube.channel.get` | `GetChannel` | sync | Channel metadata |
| `youtube.channel-videos.list` | `ListChannelVideos` | stream | Channel videos tab |
| `youtube.channel-shorts.list` | `ListChannelShorts` | stream | Channel shorts tab |
| `youtube.channel-playlists.list` | `ListChannelPlaylists` | stream | Channel playlists tab |
| `youtube.channel-community.list` | `ListChannelCommunity` | stream | Channel community posts |
| `youtube.comments.list` | `ListComments` | stream | Video comments |
| `youtube.search.query` | `Search` | sync | YouTube search |
| `youtube.playlist.get` | `GetPlaylist` | sync | Playlist metadata + videos |
| `youtube.transcript.get` | `GetTranscript` | sync | Video transcript/captions |

## Usage

```go
import (
    "github.com/embedtools/youtube-scraper/scraper"
    "github.com/embedtools/youtube-scraper/types"
)

client, err := scraper.New(
    scraper.WithHTTPClient(myHTTPClient),
)

// Sync capability
video, err := client.GetVideo(ctx, &types.GetVideoInput{
    URL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ",
})

// Streaming capability
summary, err := client.ListChannelVideos(ctx,
    &types.ListChannelVideosInput{URL: "@MrBeast", Limit: 50},
    func(item *types.ChannelTabItem) error {
        fmt.Println(item.Title)
        return nil
    },
)
```

## Canary Tests

```bash
MODULE_CANARY_NETWORK=1 go run ./cmd/canary
```
