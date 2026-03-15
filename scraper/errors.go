package scraper

import "errors"

var (
	ErrInvalidURL      = errors.New("youtube: invalid input")
	ErrNotFound        = errors.New("youtube: resource not found")
	ErrPrivateResource = errors.New("youtube: resource is private")
	ErrRateLimited     = errors.New("youtube: rate limited")
	ErrBlocked         = errors.New("youtube: blocked by upstream")
	ErrUpstreamChanged = errors.New("youtube: upstream response format changed")
	ErrContextCanceled = errors.New("youtube: context canceled")
)
