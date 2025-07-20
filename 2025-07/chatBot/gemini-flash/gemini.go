package main

import (
	"encoding/json" // Use encoding/json for proper JSON output
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings" // Added for strings.ReplaceAll

	"time" // Added for time.ParseDuration where applicable
)

// MPD represents the top-level element of a Media Presentation Description.
type MPD struct {
	XMLName  xml.Name `xml:"MPD"`
	BaseURL  string   `xml:"BaseURL"`
	Periods  []Period `xml:"Period"`
	Profiles string   `xml:"profiles,attr"`
}

// Period represents a period in the MPD.
type Period struct {
	XMLName        xml.Name       `xml:"Period"`
	Duration       string         `xml:"duration,attr"`
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet represents an adaptation set within a period.
type AdaptationSet struct {
	XMLName         xml.Name         `xml:"AdaptationSet"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
	Representations []Representation `xml:"Representation"`
}

// Representation represents a representation within an adaptation set.
type Representation struct {
	XMLName         xml.Name         `xml:"Representation"`
	ID              string           `xml:"id,attr"`
	BaseURL         string           `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

// SegmentTemplate represents a segment template.
type SegmentTemplate struct {
	XMLName       xml.Name       `xml:"SegmentTemplate"`
	Media         string         `xml:"media,attr"`
	Initialization string         `xml:"initialization,attr"`
	Timescale     uint64         `xml:"timescale,attr"`
	Duration      uint64         `xml:"duration,attr"`
	StartNumber   uint64         `xml:"startNumber,attr"`
	EndNumber     uint64         `xml:"endNumber,attr"`
	SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline represents a segment timeline.
type SegmentTimeline struct {
	XMLName xml.Name `xml:"SegmentTimeline"`
	Ss      []S      `xml:"S"`
}

// S represents an 'S' element within a SegmentTimeline.
type S struct {
	XMLName xml.Name `xml:"S"`
	T       uint64   `xml:"t,attr"`
	D       uint64   `xml:"d,attr"`
	R       int      `xml:"r,attr"`
}

// ResultJSON represents the structure of the desired JSON output.
type ResultJSON map[string][]string

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: go run dash_parser.go <path_to_mpd_file>")
		os.Exit(1)
	}

	mpdFilePath := os.Args[1]
	mpdData, err := ioutil.ReadFile(mpdFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading MPD file: %v\n", err)
		os.Exit(1)
	}

	var mpd MPD
	err = xml.Unmarshal(mpdData, &mpd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling MPD: %v\n", err)
		os.Exit(1)
	}

	segmentURLs := make(ResultJSON)
	// Rule 5: assume MPD URL is http://test.test/test.mpd
	mpdBaseURL, err := url.Parse("http://test.test/test.mpd")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing base MPD URL: %v\n", err)
		os.Exit(1)
	}

	for _, period := range mpd.Periods {
		periodDuration, err := parseISODuration(period.Duration)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not parse Period duration '%s': %v. Skipping period calculations for duration-based segment generation.\n", period.Duration, err)
			periodDuration = 0 // Set to 0 if parsing fails to avoid errors in calculation
		}

		for _, as := range period.AdaptationSets {
			for _, rep := range as.Representations {
				repID := rep.ID
				if repID == "" {
					fmt.Fprintln(os.Stderr, "Warning: Representation missing ID, skipping.")
					continue
				}

				var segTemplate *SegmentTemplate
				if rep.SegmentTemplate != nil {
					segTemplate = rep.SegmentTemplate
				} else if as.SegmentTemplate != nil {
					segTemplate = as.SegmentTemplate
				}

				// Rule 9: if Representation is missing SegmentBase, SegmentList, SegmentTemplate, return Representation@BaseURL
				if segTemplate == nil {
					if rep.BaseURL != "" {
						parsedURL, err := mpdBaseURL.Parse(rep.BaseURL)
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error parsing Representation BaseURL '%s': %v\n", rep.BaseURL, err)
							continue
						}
						segmentURLs[repID] = append(segmentURLs[repID], parsedURL.String())
					}
					continue
				}

				// Rule 12: SegmentTemplate@timescale is 1 if missing
				timescale := uint64(1)
				if segTemplate.Timescale != 0 {
					timescale = segTemplate.Timescale
				}

				// Rule 13: SegmentTemplate@startNumber is 1 if missing
				startNumber := uint64(1)
				if segTemplate.StartNumber != 0 {
					startNumber = segTemplate.StartNumber
				}

				var numSegments uint64
				if segTemplate.EndNumber != 0 {
					// Rule 16: SegmentTemplate@endNumber can exist. if so it defines the last segment
					if segTemplate.EndNumber < startNumber {
						fmt.Fprintf(os.Stderr, "Warning: Representation %s has EndNumber (%d) less than StartNumber (%d). Skipping.\n", repID, segTemplate.EndNumber, startNumber)
						continue
					}
					numSegments = segTemplate.EndNumber - startNumber + 1
				} else if segTemplate.SegmentTimeline != nil {
					// Rule 17: if no SegmentTemplate@endNumber use SegmentTimeline
					for _, s := range segTemplate.SegmentTimeline.Ss {
						numSegments += uint64(s.R + 1)
					}
				} else if segTemplate.Duration != 0 && periodDuration > 0 {
					// Rule 18: if no SegmentTimeline use ceil(asSeconds(Period@duration) * SegmentTemplate@timescale / SegmentTemplate@duration)
					numSegments = uint64(math.Ceil(float64(periodDuration.Seconds()) * float64(timescale) / float64(segTemplate.Duration)))
				} else {
					fmt.Fprintf(os.Stderr, "Warning: Representation %s has SegmentTemplate but no Duration, EndNumber, or SegmentTimeline, or Period duration could not be parsed. Skipping segment generation for this representation.\n", repID)
					continue
				}

				for i := uint64(0); i < numSegments; i++ {
					segmentNumber := startNumber + i
					mediaURL := segTemplate.Media
					// Rule 14: replace $RepresentationID$ with Representation@id
					mediaURL = strings.ReplaceAll(mediaURL, "$RepresentationID$", repID)
					// Rule 15: $Number$ value should increase by 1 each iteration
					mediaURL = strings.ReplaceAll(mediaURL, "$Number$", strconv.FormatUint(segmentNumber, 10))

					parsedURL, err := mpdBaseURL.Parse(mediaURL)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error parsing segment URL for %s: %v\n", repID, err)
						continue
					}
					// Rule 10: if Representation spans Periods, append URLs (handled by appending to slice)
					segmentURLs[repID] = append(segmentURLs[repID], parsedURL.String())
				}
			}
		}
	}

	// Output as JSON
	jsonOutput, err := json.MarshalIndent(segmentURLs, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonOutput))
}

// parseISODuration parses a simplified ISO 8601 duration string (e.g., PT1H30M5S)
// It handles P, T, H, M (minutes), S. Years, Months, Weeks, Days are ignored for simplicity as time.Duration doesn't directly support them.
func parseISODuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}
	if s[0] != 'P' {
		return 0, fmt.Errorf("duration string must start with 'P'")
	}
	s = s[1:] // Remove 'P'

	var duration time.Duration
	var currentNumStr string
	inTime := false // Flag to differentiate between Month and Minute

	for _, r := range s {
		switch r {
		case 'T':
			inTime = true
		case 'H':
			if val, err := strconv.ParseInt(currentNumStr, 10, 64); err == nil {
				duration += time.Duration(val) * time.Hour
			}
			currentNumStr = ""
		case 'M': // This 'M' is for Month if before 'T', or Minute if after 'T'
			if inTime { // If we're past 'T', it's minutes
				if val, err := strconv.ParseInt(currentNumStr, 10, 64); err == nil {
					duration += time.Duration(val) * time.Minute
				}
			}
			// If not inTime, it's a Month, which we currently ignore for time.Duration
			currentNumStr = ""
		case 'S':
			if val, err := strconv.ParseFloat(currentNumStr, 64); err == nil {
				duration += time.Duration(val * float64(time.Second))
			}
			currentNumStr = ""
		case 'Y', 'W', 'D': // Ignore Years, Weeks, Days for time.Duration
			currentNumStr = ""
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.':
			currentNumStr += string(r)
		default:
			return 0, fmt.Errorf("unsupported character in duration: %c", r)
		}
	}
	return duration, nil
}
