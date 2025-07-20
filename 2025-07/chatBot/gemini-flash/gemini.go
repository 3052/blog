package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const mpdBaseURL = "http://test.test/test.mpd"

// MPD represents the root element of a Media Presentation Description.
type MPD struct {
	XMLName  xml.Name `xml:"MPD"`
	Periods  []Period `xml:"Period"`
	MediaPresentationDuration string `xml:"mediaPresentationDuration,attr"`
}

// Period represents a period within the MPD.
type Period struct {
	XMLName      xml.Name       `xml:"Period"`
	Start        string         `xml:"start,attr"`
	Duration     string         `xml:"duration,attr"`
	BaseURLs     []BaseURL      `xml:"BaseURL"`
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// BaseURL represents a base URL.
type BaseURL struct {
	XMLName xml.Name `xml:"BaseURL"`
	Value   string   `xml:",chardata"`
}

// AdaptationSet represents an adaptation set.
type AdaptationSet struct {
	XMLName       xml.Name         `xml:"AdaptationSet"`
	MimeType      string           `xml:"mimeType,attr"`
	ContentType   string           `xml:"contentType,attr"`
	BaseURLs      []BaseURL        `xml:"BaseURL"`
	Representations []Representation `xml:"Representation"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"` // Can be child of AdaptationSet
}

// Representation represents a media representation.
type Representation struct {
	XMLName         xml.Name         `xml:"Representation"`
	ID              string           `xml:"id,attr"`
	Bandwidth       uint64           `xml:"bandwidth,attr"`
	Codecs          string           `xml:"codecs,attr"`
	BaseURLs        []BaseURL        `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"` // Can be child of Representation
}

// SegmentTemplate represents a segment template.
type SegmentTemplate struct {
	XMLName         xml.Name         `xml:"SegmentTemplate"`
	Media           string           `xml:"media,attr"`
	Initialization  string           `xml:"initialization,attr"`
	Duration        uint64           `xml:"duration,attr"` // In timescale units
	Timescale       uint64           `xml:"timescale,attr"`
	StartNumber     uint64           `xml:"startNumber,attr"`
	EndNumber       *uint64          `xml:"endNumber,attr"` // Pointer to distinguish missing from 0
	SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline represents a segment timeline.
type SegmentTimeline struct {
	XMLName xml.Name `xml:"SegmentTimeline"`
	Segments []S      `xml:"S"`
}

// S represents a segment in a SegmentTimeline.
type S struct {
	XMLName  xml.Name `xml:"S"`
	T        *uint64  `xml:"t,attr"` // Start time in timescale units
	D        uint64   `xml:"d,attr"` // Duration in timescale units
	R        *int64   `xml:"r,attr"` // Repeat count (-1 for indefinite)
}

// Output structure for JSON
type Output struct {
	Representations map[string][]string `json:"representations"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run mpd_parser.go <path_to_mpd_file>")
		os.Exit(1)
	}

	mpdFilePath := os.Args[1]

	// Read the MPD file
	xmlFile, err := ioutil.ReadFile(mpdFilePath)
	if err != nil {
		fmt.Printf("Error reading MPD file: %v\n", err)
		os.Exit(1)
	}

	var mpd MPD
	err = xml.Unmarshal(xmlFile, &mpd)
	if err != nil {
		fmt.Printf("Error unmarshalling MPD XML: %v\n", err)
		os.Exit(1)
	}

	output := Output{
		Representations: make(map[string][]string),
	}

	mpdBase, err := url.Parse(mpdBaseURL)
	if err != nil {
		fmt.Printf("Error parsing MPD base URL: %v\n", err)
		os.Exit(1)
	}

	for _, period := range mpd.Periods {
		// Period BaseURLs
		periodBaseURLs := getBaseURLs(period.BaseURLs)

		// Calculate Period Duration in seconds
		periodDurationSec, err := parseDuration(period.Duration)
		if err != nil {
			fmt.Printf("Warning: Could not parse Period duration '%s': %v. Skipping period duration calculation for segment count.\n", period.Duration, err)
			periodDurationSec = 0 // Indicate indefinite or unknown duration
		}

		for _, as := range period.AdaptationSets {
			// AdaptationSet BaseURLs
			asBaseURLs := getBaseURLs(as.BaseURLs)

			// Inherit SegmentTemplate from AdaptationSet if not present on Representation
			asSegmentTemplate := as.SegmentTemplate

			for _, rep := range as.Representations {
				// Representation BaseURLs
				repBaseURLs := getBaseURLs(rep.BaseURLs)

				// Determine effective BaseURL for this representation
				effectiveBaseURLs := getEffectiveBaseURLs(periodBaseURLs, asBaseURLs, repBaseURLs)
				if len(effectiveBaseURLs) == 0 {
					// Default to MPD base URL if no BaseURL is specified anywhere
					effectiveBaseURLs = []string{mpdBase.String()}
				}

				// Determine effective SegmentTemplate
				effectiveST := rep.SegmentTemplate
				if effectiveST == nil {
					effectiveST = asSegmentTemplate
				}

				var segmentURLs []string

				if effectiveST == nil {
					// Case: Representation with only BaseURL means one segment
					for _, baseURLStr := range effectiveBaseURLs {
						baseURI, err := url.Parse(baseURLStr)
						if err != nil {
							fmt.Printf("Warning: Could not parse BaseURL '%s': %v. Skipping.\n", baseURLStr, err)
							continue
						}
						resolvedURL := mpdBase.ResolveReference(baseURI)
						segmentURLs = append(segmentURLs, resolvedURL.String())
					}
				} else {
					// Handle SegmentTemplate
					timescale := uint64(1) // SegmentTemplate@timescale is 1 if missing
					if effectiveST.Timescale != 0 {
						timescale = effectiveST.Timescale
					}

					var segmentCount int
					if effectiveST.EndNumber != nil {
						// SegmentTemplate@endNumber defines the last segment
						segmentCount = int(*effectiveST.EndNumber - effectiveST.StartNumber + 1)
					} else if effectiveST.SegmentTimeline != nil {
						// Use SegmentTimeline
						currentSegmentTime := uint64(0)
						for _, s := range effectiveST.SegmentTimeline.Segments {
							if s.T != nil {
								currentSegmentTime = *s.T
							}
							segmentCount++
							numRepeats := 0
							if s.R != nil && *s.R != -1 { // -1 for indefinite, treat as single segment for now
								numRepeats = int(*s.R)
							}
							segmentCount += numRepeats
							currentSegmentTime += (s.D * uint64(1+numRepeats))
						}
					} else {
						// Calculate segment count using Period@duration, SegmentTemplate@timescale, SegmentTemplate@duration
						if periodDurationSec > 0 && effectiveST.Duration > 0 && timescale > 0 {
							calculatedSegments := float64(periodDurationSec) * float64(timescale) / float64(effectiveST.Duration)
							segmentCount = int(math.Ceil(calculatedSegments))
						} else {
							fmt.Printf("Warning: Cannot determine segment count for representation %s due to missing Period duration or SegmentTemplate duration/timescale. Defaulting to 1 segment if media template available.\n", rep.ID)
							if effectiveST.Media != "" {
								segmentCount = 1 // At least one segment if media template is provided
							}
						}
					}

					for i := 0; i < segmentCount; i++ {
						segmentNumber := effectiveST.StartNumber + uint64(i)
						mediaURLTemplate := effectiveST.Media

						// Replace $Number$
						mediaURL := strings.Replace(mediaURLTemplate, "$Number$", fmt.Sprintf("%d", segmentNumber), -1)
						// Replace $RepresentationID$
						mediaURL = strings.Replace(mediaURL, "$RepresentationID$", rep.ID, -1)
						// Replace $Bandwidth$
						mediaURL = strings.Replace(mediaURL, "$Bandwidth$", fmt.Sprintf("%d", rep.Bandwidth), -1)

						// Resolve relative URL
						for _, baseURLStr := range effectiveBaseURLs {
							baseURI, err := url.Parse(baseURLStr)
							if err != nil {
								fmt.Printf("Warning: Could not parse BaseURL '%s': %v. Skipping.\n", baseURLStr, err)
								continue
							}
							// Relative media URL
							relativeMediaURI, err := url.Parse(mediaURL)
							if err != nil {
								fmt.Printf("Warning: Could not parse media URL '%s': %v. Skipping.\n", mediaURL, err)
								continue
							}

							resolvedURL := mpdBase.ResolveReference(baseURI.ResolveReference(relativeMediaURI))
							segmentURLs = append(segmentURLs, resolvedURL.String())
						}
					}
				}

				if len(segmentURLs) > 0 {
					if _, ok := output.Representations[rep.ID]; !ok {
						output.Representations[rep.ID] = []string{}
					}
					output.Representations[rep.ID] = append(output.Representations[rep.ID], segmentURLs...)
				}
			}
		}
	}

	// Output as JSON
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Printf("Error marshalling to JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonData))
}

// Helper to get BaseURLs from a slice of BaseURL structs.
func getBaseURLs(bases []BaseURL) []string {
	var urls []string
	for _, b := range bases {
		urls = append(urls, b.Value)
	}
	return urls
}

// Helper to determine the effective BaseURL based on hierarchy.
func getEffectiveBaseURLs(periodURLs, asURLs, repURLs []string) []string {
	if len(repURLs) > 0 {
		return repURLs
	}
	if len(asURLs) > 0 {
		return asURLs
	}
	return periodURLs
}

// parseDuration parses an XSD duration string (e.g., "PT3M30S") into seconds.
func parseDuration(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	// Simplified parsing for common cases like PT...S, PT...M, PT...H
	// Does not handle P...Y...M...D or combinations of H, M, S rigorously.
	// For production, a more robust XSD duration parser is recommended.

	var totalSeconds float64
	s = strings.TrimPrefix(s, "P")
	if strings.HasPrefix(s, "T") {
		s = strings.TrimPrefix(s, "T")
	} else {
		return 0, fmt.Errorf("unsupported duration format: %s (expected 'PT...')", s)
	}

	parts := strings.Split(s, "")
	var currentNumStr string
	for _, part := range parts {
		if '0' <= part[0] && part[0] <= '9' || part == "." {
			currentNumStr += part
		} else {
			if currentNumStr == "" {
				continue
			}
			val, err := strconv.ParseFloat(currentNumStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number in duration: %s", currentNumStr)
			}
			switch part {
			case "H":
				totalSeconds += val * 3600
			case "M":
				totalSeconds += val * 60
			case "S":
				totalSeconds += val
			default:
				return 0, fmt.Errorf("unsupported duration unit: %s", part)
			}
			currentNumStr = ""
		}
	}
	if currentNumStr != "" { // Handle case where duration ends with a number without unit, e.g., "PT10"
		return 0, fmt.Errorf("duration ends with number without unit: %s", s)
	}

	return totalSeconds, nil
}
