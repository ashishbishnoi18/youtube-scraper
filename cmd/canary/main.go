package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/embedtools/youtube-scraper/scraper"
	"github.com/embedtools/youtube-scraper/types"
)

func main() {
	if os.Getenv("MODULE_CANARY_NETWORK") == "" {
		fmt.Println("SKIP: set MODULE_CANARY_NETWORK=1 to run canary tests against live APIs")
		os.Exit(0)
	}

	client, err := scraper.New()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	passed := 0
	failed := 0

	// 1. GetVideo
	fmt.Print("Testing GetVideo... ")
	video, err := client.GetVideo(ctx, &types.GetVideoInput{URL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ"})
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		failed++
	} else if video.VideoID == "" || video.Title == "" || video.ChannelID == "" {
		fmt.Println("FAIL: missing required fields")
		failed++
	} else {
		fmt.Printf("OK (title=%q)\n", video.Title)
		passed++
	}

	// 2. GetChannel
	fmt.Print("Testing GetChannel... ")
	channel, err := client.GetChannel(ctx, &types.GetChannelInput{URL: "https://www.youtube.com/@MrBeast"})
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		failed++
	} else if channel.ChannelID == "" || channel.Title == "" {
		fmt.Println("FAIL: missing required fields")
		failed++
	} else {
		fmt.Printf("OK (title=%q)\n", channel.Title)
		passed++
	}

	// 3. ListChannelVideos
	fmt.Print("Testing ListChannelVideos... ")
	var videoItems []*types.ChannelTabItem
	videoSummary, err := client.ListChannelVideos(ctx, &types.ListChannelVideosInput{URL: "https://www.youtube.com/@MrBeast", Limit: 3}, func(item *types.ChannelTabItem) error {
		videoItems = append(videoItems, item)
		return nil
	})
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		failed++
	} else if len(videoItems) == 0 {
		fmt.Println("FAIL: no items emitted")
		failed++
	} else {
		fmt.Printf("OK (%d items, %d pages)\n", videoSummary.TotalItems, videoSummary.Pages)
		passed++
	}

	// 4. Search
	fmt.Print("Testing Search... ")
	searchResult, err := client.Search(ctx, &types.SearchInput{Query: "golang tutorial", Limit: 5})
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		failed++
	} else if len(searchResult.Results) == 0 {
		fmt.Println("FAIL: no results")
		failed++
	} else {
		fmt.Printf("OK (%d results)\n", searchResult.ResultsCount)
		passed++
	}

	// 5. GetTranscript
	fmt.Print("Testing GetTranscript... ")
	transcript, err := client.GetTranscript(ctx, &types.GetTranscriptInput{URL: "https://www.youtube.com/watch?v=dQw4w9WgXcQ"})
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		failed++
	} else if transcript.VideoID == "" || len(transcript.Transcript) == 0 {
		fmt.Println("FAIL: missing required fields")
		failed++
	} else {
		fmt.Printf("OK (%d segments, lang=%s)\n", transcript.SegmentCount, transcript.Language)
		passed++
	}

	fmt.Printf("\n--- Results: %d passed, %d failed ---\n", passed, failed)

	if failed > 0 {
		os.Exit(1)
	}

	// Print sample output for verification
	if video != nil {
		fmt.Println("\nSample GetVideo output:")
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]interface{}{
			"video_id":     video.VideoID,
			"title":        video.Title,
			"channel_name": video.ChannelName,
			"view_count":   video.ViewCount,
		})
	}
}
