package media

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Embed accepts a pasted video URL and returns a sandboxed iframe URL. It costs
// nothing to run, which is how most tutors already host their video.
type Embed struct {
	// Extra hosts an operator trusts, beyond the built-in ones.
	AllowedHosts []string
}

var (
	youtubeID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
	numericID = regexp.MustCompile(`^[0-9]{6,20}$`)
)

func (Embed) Caps() Caps {
	return Caps{Name: "embed", AcceptsURL: true, Renditions: true, Captions: true}
}

func (e Embed) Ingest(_ context.Context, _ string, src Source) (Ingested, error) {
	if strings.TrimSpace(src.URL) == "" {
		return Ingested{}, ErrUnsupportedSource
	}

	u, err := url.Parse(strings.TrimSpace(src.URL))
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return Ingested{}, fmt.Errorf("%w: provide an https video link", ErrUnsupportedSource)
	}

	embedURL, platform, err := e.resolve(u)
	if err != nil {
		return Ingested{}, err
	}
	return Ingested{
		ExternalRef: embedURL,
		State:       StateReady,
		ContentType: "text/html",
		Metadata:    map[string]any{"platform": platform, "source_url": u.String()},
	}, nil
}

func (Embed) Playback(_ context.Context, a Asset, _ Viewer) (Playback, error) {
	if a.State != StateReady {
		return Playback{}, ErrNotReady
	}
	return Playback{
		Kind: PlaybackEmbed,
		URL:  a.ExternalRef,
		Headers: map[string]any{
			"sandbox": "allow-scripts allow-same-origin allow-presentation",
			"allow":   "accelerometer; encrypted-media; picture-in-picture; fullscreen",
		},
	}, nil
}

// resolve turns a watch URL into an embed URL, rejecting anything unrecognized.
func (e Embed) resolve(u *url.URL) (string, string, error) {
	host := strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
	path := strings.Trim(u.Path, "/")

	switch host {
	case "youtube.com", "m.youtube.com", "youtube-nocookie.com":
		id := u.Query().Get("v")
		if id == "" {
			id = strings.TrimPrefix(lastSegment(path), "embed/")
		}
		if !youtubeID.MatchString(id) {
			return "", "", fmt.Errorf("%w: that YouTube link has no video id", ErrUnsupportedSource)
		}
		return "https://www.youtube-nocookie.com/embed/" + id, "youtube", nil

	case "youtu.be":
		if !youtubeID.MatchString(path) {
			return "", "", fmt.Errorf("%w: that YouTube link has no video id", ErrUnsupportedSource)
		}
		return "https://www.youtube-nocookie.com/embed/" + path, "youtube", nil

	case "vimeo.com", "player.vimeo.com":
		id := lastSegment(path)
		if !numericID.MatchString(id) {
			return "", "", fmt.Errorf("%w: that Vimeo link has no video id", ErrUnsupportedSource)
		}
		return "https://player.vimeo.com/video/" + id, "vimeo", nil

	case "dailymotion.com":
		id := strings.SplitN(lastSegment(path), "_", 2)[0]
		if id == "" {
			return "", "", fmt.Errorf("%w: that Dailymotion link has no video id", ErrUnsupportedSource)
		}
		return "https://www.dailymotion.com/embed/video/" + id, "dailymotion", nil
	}

	// Self-hosted platforms an operator has explicitly trusted, PeerTube included.
	for _, allowed := range e.AllowedHosts {
		if host == strings.ToLower(strings.TrimSpace(allowed)) {
			return u.String(), host, nil
		}
	}
	return "", "", fmt.Errorf("%w: %s is not an allowed video host", ErrUnsupportedSource, host)
}

func lastSegment(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}
