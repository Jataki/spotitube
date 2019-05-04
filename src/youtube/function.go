package youtube

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	spttb_track "track"

	"github.com/PuerkitoBio/goquery"
	"github.com/agnivade/levenshtein"
	"github.com/kennygrant/sanitize"
)

func pullTracksFromDoc(track spttb_track.Track, document *goquery.Document) (Tracks, error) {
	var (
		tracks            = []Track{}
		selection         = document.Find(YouTubeHTMLVideoSelector)
		selectionDesc     = document.Find(YouTubeHTMLDescSelector)
		selectionDuration = document.Find(YouTubeHTMLDurationSelector)
		selectionPointer  int
		selectionError    error
	)
	for selectionPointer+1 < len(selection.Nodes) {
		selectionPointer++

		item := selection.Eq(selectionPointer)
		itemHref, itemHrefOk := item.Attr("href")
		itemTitle, itemTitleOk := item.Attr("title")
		itemUser, _ := "UNKNOWN", false
		itemLength, itemLengthOk := 0, false
		if selectionPointer < len(selectionDesc.Nodes) {
			itemDesc := selectionDesc.Eq(selectionPointer)
			itemUser = strings.TrimSpace(itemDesc.Find("a").Text())
			// itemUserOk = true
		}
		if selectionPointer < len(selectionDuration.Nodes) {
			var itemLengthMin, itemLengthSec int
			itemDuration := selectionDuration.Eq(selectionPointer)
			itemLengthSectr := strings.TrimSpace(itemDuration.Text())
			if strings.Contains(itemLengthSectr, ": ") {
				itemLengthSectr = strings.Split(itemLengthSectr, ": ")[1]
				itemLengthMin, selectionError = strconv.Atoi(strings.Split(itemLengthSectr, ":")[0])
				if selectionError == nil {
					itemLengthSec, selectionError = strconv.Atoi(strings.Split(itemLengthSectr, ":")[1][:2])
					if selectionError == nil {
						itemLength = itemLengthMin*60 + itemLengthSec
						itemLengthOk = true
					}
				}
			}
		}
		if itemHrefOk && itemTitleOk && itemLengthOk &&
			(strings.Contains(strings.ToLower(itemHref), "youtu.be") || !strings.Contains(strings.ToLower(itemHref), "&list=")) &&
			(strings.Contains(strings.ToLower(itemHref), "youtu.be") || strings.Contains(strings.ToLower(itemHref), "watch?v=")) {
			tracks = append(tracks, Track{
				Track:    &track,
				ID:       IDFromURL(YouTubeVideoPrefix + itemHref),
				URL:      YouTubeVideoPrefix + itemHref,
				Title:    itemTitle,
				User:     itemUser,
				Duration: itemLength,
			})
		}
	}

	return tracks, nil
}

func (tracks Tracks) evaluateScores() Tracks {
	var evaluatedTracks Tracks
	for _, track := range tracks {
		if math.Abs(float64(track.Track.Duration-track.Duration)) <= float64(YouTubeDurationTolerance/2) {
			track.AffinityScore += 20
		} else if math.Abs(float64(track.Track.Duration-track.Duration)) <= float64(YouTubeDurationTolerance) {
			track.AffinityScore += 10
		}
		if err := track.Track.SeemsByWordMatch(fmt.Sprintf("%s %s", track.User, track.Title)); err == nil {
			track.AffinityScore += 10
		}
		if strings.Contains(sanitize.Name(track.User), sanitize.Name(track.Track.Artist)) {
			track.AffinityScore += 10
		}
		if spttb_track.SeemsType(track.Title, track.Track.SongType) {
			track.AffinityScore += 10
		}
		levenshteinDistance := levenshtein.ComputeDistance(track.Track.SearchPattern, fmt.Sprintf("%s %s", track.User, track.Title))
		track.AffinityScore -= levenshteinDistance
		evaluatedTracks = append(evaluatedTracks, track)
	}
	return evaluatedTracks
}
