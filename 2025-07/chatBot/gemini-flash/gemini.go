package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// MPDBaseURL is used to resolve relative URLs in the MPD.
const MPDBaseURL = "http://test.test/test.mpd"

// MPD represents the root element of the DASH Media Presentation Description.
type MPD struct {
	XMLName xml.Name `xml:"MPD"`
	BaseURLs []BaseURL `xml:"BaseURL"`
	Periods []Period `xml:"Period"`
}

// Period represents a Period in the MPD.
type Period struct {
	XMLName xml.Name `xml:"Period"`
	ID *string `xml:"id,attr"`
	Duration *string `xml:"duration,attr"` // Not directly used for segment calculation, but useful for overall period understanding
	BaseURLs []BaseURL `xml:"BaseURL"`
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet represents an AdaptationSet in the MPD.
type AdaptationSet struct {
	XMLName xml.Name `xml:"AdaptationSet"`
	ID *string `xml:"id,attr"`
	BaseURLs []BaseURL `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
	Representations []Representation `xml:"Representation"`
}

// Representation represents a Representation in the MPD.
type Representation struct {
	XMLName xml.Name `xml:"Representation"`
	ID string `xml:"id,attr"`
	Bandwidth *uint64 `xml:"bandwidth,attr"` // Example attribute, not used in segment generation but common
	BaseURLs []BaseURL `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
	SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// BaseURL represents a BaseURL element.
type BaseURL struct {
	XMLName xml.Name `xml:"BaseURL"`
	Value string `xml:",chardata"`
}

// SegmentTemplate represents a SegmentTemplate element.
type SegmentTemplate struct {
	XMLName xml.Name `xml:"SegmentTemplate"`
	Media string `xml:"media,attr"`
	Initialization string `xml:"initialization,attr"`
	Duration *uint64 `xml:"duration,attr"`
	StartNumber *uint64 `xml:"startNumber,attr"`
	Timescale *uint64 `xml:"timescale,attr"`
	EndNumber *uint64 `xml:"endNumber,attr"` // endNumber defines the last segment when it exists
	SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline represents a SegmentTimeline element.
type SegmentTimeline struct {
	XMLName xml.Name `xml:"SegmentTimeline"`
	Segments []S `xml:"S"`
}

// S represents an S element within SegmentTimeline.
type S struct {
	XMLName xml.Name `xml:"S"`
	Duration uint64 `xml:"d,attr"`
	Repeat *uint64 `xml:"r,attr"`
	Time *uint64 `xml:"t,attr"` // Not used for segment number in this logic, but for presentation time
}

// Output structures for JSON
type RepresentationSegments struct {
	ID string `json:"representationId"`
	SegmentURLs []string `json:"segmentUrls"`
}

type MPDSegmentOutput struct {
	Representations []RepresentationSegments `json:"representations"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <path_to_mpd_file>")
		os.Exit(1)
	}

	mpdFilePath := os.Args[1]

	mpdData, err := ioutil.ReadFile(mpdFilePath)
	if err != nil {
		fmt.Printf("Error reading MPD file: %v\n", err)
		os.Exit(1)
	}

	var mpd MPD
	err = xml.Unmarshal(mpdData, &mpd)
	if err != nil {
		fmt.Printf("Error unmarshalling MPD XML: %v\n", err)
		os.Exit(1)
	}

	output := MPDSegmentOutput{}

	// Parse the MPD base URL
	parsedMPDBase, err := url.Parse(MPDBaseURL)
	if err != nil {
		fmt.Printf("Error parsing MPD base URL: %v\n", err)
		os.Exit(1)
	}

	// Iterate through Periods, AdaptationSets, and Representations
	for _, period := range mpd.Periods {
		// Inherit BaseURLs from MPD
		currentBaseURLs := mpd.BaseURLs
		if len(period.BaseURLs) > 0 {
			currentBaseURLs = period.BaseURLs
		}

		for _, as := range period.AdaptationSets {
			// Inherit BaseURLs from Period or MPD
			asBaseURLs := currentBaseURLs
			if len(as.BaseURLs) > 0 {
				asBaseURLs = as.BaseURLs
			}

			// Inherit SegmentTemplate from AdaptationSet if available
			asSegmentTemplate := as.SegmentTemplate

			for _, rep := range as.Representations {
				repSegments := RepresentationSegments{
					ID: rep.ID,
					SegmentURLs: []string{},
				}

				// Determine effective BaseURL for this Representation
				effectiveBaseURL := parsedMPDBase
				if len(rep.BaseURLs) > 0 {
					if u, err := url.Parse(rep.BaseURLs[0].Value); err == nil {
						effectiveBaseURL = parsedMPDBase.ResolveReference(u)
					}
				} else if len(asBaseURLs) > 0 {
					if u, err := url.Parse(asBaseURLs[0].Value); err == nil {
						effectiveBaseURL = parsedMPDBase.ResolveReference(u)
					}
				}

				// Prioritize SegmentTemplate/SegmentTimeline at Representation level
				var segmentTemplate *SegmentTemplate
				if rep.SegmentTemplate != nil {
					segmentTemplate = rep.SegmentTemplate
				} else if asSegmentTemplate != nil {
					segmentTemplate = asSegmentTemplate
				}

				var segmentTimeline *SegmentTimeline
				if rep.SegmentTimeline != nil {
					segmentTimeline = rep.SegmentTimeline
				} else if segmentTemplate != nil && segmentTemplate.SegmentTimeline != nil {
					segmentTimeline = segmentTemplate.SegmentTimeline
				}


				if segmentTimeline != nil && len(segmentTimeline.Segments) > 0 {
					// Handle SegmentTimeline
					currentSegmentNumber := uint64(0)
					if segmentTemplate != nil && segmentTemplate.StartNumber != nil {
						currentSegmentNumber = *segmentTemplate.StartNumber
					} else {
						currentSegmentNumber = 1 // Default startNumber
					}

					for _, s := range segmentTimeline.Segments {
						numRepeats := uint64(0)
						if s.Repeat != nil {
							numRepeats = *s.Repeat
						}

						for i := uint64(0); i <= numRepeats; i++ {
							segmentURL := replaceSegmentTemplatePlaceholders(segmentTemplate.Media, rep.ID, currentSegmentNumber) // Removed segmentTime parameter as it's not directly incremented in this loop
							resolvedURL := effectiveBaseURL.ResolveReference(mustParseURL(segmentURL)).String()
							repSegments.SegmentURLs = append(repSegments.SegmentURLs, resolvedURL)
							currentSegmentNumber++
						}
					}

				} else if segmentTemplate != nil {
					// Handle SegmentTemplate
					var startNumber uint64 = 1
					if segmentTemplate.StartNumber != nil {
						startNumber = *segmentTemplate.StartNumber
					}

					// timescale is used for duration calculation if needed, but not directly for URL placeholders here
					// var timescale uint64 = 1
					// if segmentTemplate.Timescale != nil {
					// 	timescale = *segmentTemplate.Timescale
					// }

					if segmentTemplate.Duration != nil {
						// Calculate number of segments based on duration and timescale
						// Assuming a fixed duration for each segment
						// MediaPresentationDuration from MPD or Period duration could be used for total segments
						// but without it, we need endNumber or an arbitrary limit.
						// For this exercise, if endNumber exists, use it. Otherwise, assume a reasonable number
						// or require endNumber.
						var totalSegments int
						if segmentTemplate.EndNumber != nil {
							totalSegments = int(*segmentTemplate.EndNumber - startNumber + 1)
						} else {
							// If endNumber is missing, we need a way to determine the number of segments.
							// For static content, MediaPresentationDuration might be available at MPD or Period.
							// For simplicity, if duration is present and endNumber is not, we'll generate
							// a fixed number of segments (e.g., 10) for demonstration purposes,
							// but in a real-world scenario, you'd need more information.
							fmt.Printf("Warning: SegmentTemplate for Representation %s has duration but no endNumber. Generating 10 segments for demonstration.\n", rep.ID)
							totalSegments = 10 // Arbitrary limit for demonstration
						}

						for i := 0; i < totalSegments; i++ {
							segmentNumber := startNumber + uint64(i)
							segmentURL := replaceSegmentTemplatePlaceholders(segmentTemplate.Media, rep.ID, segmentNumber) // Removed segmentTime parameter
							resolvedURL := effectiveBaseURL.ResolveReference(mustParseURL(segmentURL)).String()
							repSegments.SegmentURLs = append(repSegments.SegmentURLs, resolvedURL)
						}
					} else if len(rep.BaseURLs) > 0 && rep.SegmentTemplate == nil && rep.SegmentTimeline == nil {
						// Representation with only BaseURL means one segment
						resolvedURL := effectiveBaseURL.String()
						repSegments.SegmentURLs = append(repSegments.SegmentURLs, resolvedURL)
					}
				} else if len(rep.BaseURLs) > 0 && rep.SegmentTemplate == nil && rep.SegmentTimeline == nil {
					// Representation with only BaseURL means one segment
					resolvedURL := effectiveBaseURL.String()
					repSegments.SegmentURLs = append(repSegments.SegmentURLs, resolvedURL)
				}


				if len(repSegments.SegmentURLs) > 0 {
					output.Representations = append(output.Representations, repSegments)
				}
			}
		}
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Printf("Error marshalling JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonData))
}

// replaceSegmentTemplatePlaceholders replaces $RepresentationID$ and $Number$ in a string.
// $Time$ is typically used with SegmentTimeline and its value isn't directly incremented here.
func replaceSegmentTemplatePlaceholders(template string, representationID string, segmentNumber uint64) string {
	s := strings.ReplaceAll(template, "$RepresentationID$", representationID)
	s = strings.ReplaceAll(s, "$Number$", strconv.FormatUint(segmentNumber, 10))
	// $Time$ is not typically incremented in this manner for $Number$-based templates,
	// so it's not directly included as an input parameter for simple $Number$ iteration.
	// If $Time$ were needed, it would derive from `t` attribute in SegmentTimeline.
	s = strings.ReplaceAll(s, "$Time$", "0") // Placeholder, as it's not directly derived from segmentNumber here
	return s
}

// mustParseURL is a helper to parse URL and panic on error for simplicity in this example.
func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse URL: %s, error: %v", rawURL, err))
	}
	return u
}
