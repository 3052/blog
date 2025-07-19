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

// MPDBaseURL is used to resolve relative URLs in the MPD file.
const mpdBaseURL = "http://test.test/test.mpd"

// MPD represents the root element of a DASH Media Presentation Description.
type MPD struct {
	XMLName xml.Name `xml:"MPD"`
	Periods []Period `xml:"Period"`
}

// Period represents a period of the media presentation.
type Period struct {
	ID             string          `xml:"id,attr"`
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet groups one or more Representations that are alternative encodings of the same content.
type AdaptationSet struct {
	ID              string           `xml:"id,attr"`
	BaseURL         string           `xml:"BaseURL"` // Can be inherited by Representations
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"` // Can be inherited by Representations
	Representations []Representation `xml:"Representation"`
}

// Representation describes a deliverable encoded version of a media content component.
type Representation struct {
	ID              string           `xml:"id,attr"`
	Bandwidth       uint64           `xml:"bandwidth,attr"`
	BaseURL         string           `xml:"BaseURL"` // Overrides AdaptationSet's BaseURL
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"` // Overrides AdaptationSet's SegmentTemplate
	SegmentList     *SegmentList     `xml:"SegmentList"`
}

// SegmentTemplate specifies a template for generating segment URLs.
type SegmentTemplate struct {
	Timescale      uint64           `xml:"timescale,attr"`      // Default 1
	Initialization string           `xml:"initialization,attr"` // URL template for initialization segment
	Media          string           `xml:"media,attr"`          // URL template for media segments
	StartNumber    uint64           `xml:"startNumber,attr"`    // Default 1
	Duration       uint64           `xml:"duration,attr"`       // Duration of each segment in timescale units (if no SegmentTimeline)
	SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline provides a compact way to specify segment durations and start times.
type SegmentTimeline struct {
	Ss []S `xml:"S"`
}

// S represents a single segment or a series of repeated segments within a SegmentTimeline.
type S struct {
	T *uint64 `xml:"t,attr"` // Optional start time
	D uint64  `xml:"d,attr"` // Duration of segment in timescale units <-- CORRECTED HERE
	R *int64  `xml:"r,attr"` // Repeat count (can be -1 for implicit last segment)
}


// SegmentList provides an explicit list of segment URLs.
type SegmentList struct {
	SegmentURLs []SegmentURL `xml:"SegmentURL"`
}

// SegmentURL specifies the URL of a single media segment.
type SegmentURL struct {
	Media string `xml:"media,attr"` // Relative or absolute URL of the segment
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <path_to_dash_mpd_file.mpd>")
		os.Exit(1)
	}

	mpdFilePath := os.Args[1]

	mpdContent, err := ioutil.ReadFile(mpdFilePath)
	if err != nil {
		fmt.Printf("Error reading MPD file: %v\n", err)
		os.Exit(1)
	}

	var mpd MPD
	err = xml.Unmarshal(mpdContent, &mpd)
	if err != nil {
		fmt.Printf("Error unmarshalling MPD: %v\n", err)
		os.Exit(1)
	}

	// Parse the base URL for resolution
	resolvedMPDBaseURL, err := url.Parse(mpdBaseURL)
	if err != nil {
		fmt.Printf("Error parsing base MPD URL '%s': %v\n", mpdBaseURL, err)
		os.Exit(1)
	}

	// Map to store segment URLs, grouped by Representation ID.
	// This implicitly handles consolidation of Representations split across Periods.
	representationSegments := make(map[string][]string)

	for _, period := range mpd.Periods {
		for _, as := range period.AdaptationSets {
			// Determine effective SegmentTemplate and BaseURL for current AdaptationSet
			currentASSegmentTemplate := as.SegmentTemplate
			currentASBaseURL := as.BaseURL

			for _, rep := range as.Representations {
				// Determine the effective BaseURL for the current Representation
				effectiveRepBaseURL := rep.BaseURL
				if effectiveRepBaseURL == "" {
					effectiveRepBaseURL = currentASBaseURL // Inherit from AdaptationSet
				}

				// Determine the effective SegmentTemplate for the current Representation
				effectiveRepSegmentTemplate := rep.SegmentTemplate
				if effectiveRepSegmentTemplate == nil {
					effectiveRepSegmentTemplate = currentASSegmentTemplate // Inherit from AdaptationSet
				}

				var segments []string

				if rep.BaseURL != "" && effectiveRepSegmentTemplate == nil && rep.SegmentList == nil {
					// Case 4: Representation contains only BaseURL, treat it as the only segment
					resolvedURL, err := resolvedMPDBaseURL.Parse(rep.BaseURL)
					if err != nil {
						fmt.Printf("Warning: Could not resolve BaseURL '%s' for Representation %s: %v\n", rep.BaseURL, rep.ID, err)
					} else {
						segments = append(segments, resolvedURL.String())
					}
				} else if effectiveRepSegmentTemplate != nil {
					// Handle SegmentTemplate
					generated, err := generateSegmentTemplateURLs(
						resolvedMPDBaseURL,
						effectiveRepBaseURL,
						effectiveRepSegmentTemplate,
						rep.ID,
						rep.Bandwidth,
					)
					if err != nil {
						fmt.Printf("Warning: Error generating SegmentTemplate URLs for Representation %s: %v\n", rep.ID, err)
					}
					segments = append(segments, generated...)
				} else if rep.SegmentList != nil {
					// Handle SegmentList
					generated, err := generateSegmentListURLs(
						resolvedMPDBaseURL,
						effectiveRepBaseURL,
						rep.SegmentList,
					)
					if err != nil {
						fmt.Printf("Warning: Error generating SegmentList URLs for Representation %s: %v\n", rep.ID, err)
					}
					segments = append(segments, generated...)
				}
				// Append segments to the map, consolidating by Representation ID
				representationSegments[rep.ID] = append(representationSegments[rep.ID], segments...)
			}
		}
	}

	// Marshal the result to JSON
	jsonOutput, err := json.MarshalIndent(representationSegments, "", "  ")
	if err != nil {
		fmt.Printf("Error marshalling to JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonOutput))
}

// generateSegmentTemplateURLs generates segment URLs based on SegmentTemplate properties.
func generateSegmentTemplateURLs(baseURL *url.URL, repBaseURL string, st *SegmentTemplate, representationID string, bandwidth uint64) ([]string, error) {
	var segmentURLs []string

	timescale := st.Timescale
	if timescale == 0 {
		timescale = 1 // Default timescale is 1
	}

	startNumber := st.StartNumber
	if startNumber == 0 {
		startNumber = 1 // Default startNumber is 1
	}

	// 1. Handle Initialization segment
	if st.Initialization != "" {
		initURL := strings.ReplaceAll(st.Initialization, "$RepresentationID$", representationID)
		initURL = strings.ReplaceAll(initURL, "$Bandwidth$", strconv.FormatUint(bandwidth, 10))

		// Resolve against the MPD's base URL (http://test.test/test.mpd) combined with Representation's BaseURL
		resolvedInitURL, err := baseURL.Parse(repBaseURL + initURL)
		if err != nil {
			return nil, fmt.Errorf("error resolving initialization URL '%s': %w", repBaseURL+initURL, err)
		}
		segmentURLs = append(segmentURLs, resolvedInitURL.String())
	}

	// 2. Handle Media segments
	if st.SegmentTimeline != nil {
		// Use SegmentTimeline to generate segments
		currentTime := uint64(0)
		currentNumber := startNumber

		// If the first S element has a 't' attribute, set the initial time
		if len(st.SegmentTimeline.Ss) > 0 && st.SegmentTimeline.Ss[0].T != nil {
			currentTime = *st.SegmentTimeline.Ss[0].T
		}

		for _, s := range st.SegmentTimeline.Ss {
			// If 't' attribute is present, it explicitly sets the start time for this segment group
			if s.T != nil {
				currentTime = *s.T
			}

			// Calculate number of segments in this 'S' element
			numSegmentsInS := int64(1) // Default for r=0 or no r attribute
			if s.R != nil {
				numSegmentsInS += *s.R // r=N means N+1 segments
			}

			for i := int64(0); i < numSegmentsInS; i++ {
				// Replace placeholders in the media URL template
				mediaURL := strings.ReplaceAll(st.Media, "$Number$", strconv.FormatUint(currentNumber, 10))
				mediaURL = strings.ReplaceAll(mediaURL, "$Time$", strconv.FormatUint(currentTime, 10))
				mediaURL = strings.ReplaceAll(mediaURL, "$RepresentationID$", representationID)
				mediaURL = strings.ReplaceAll(mediaURL, "$Bandwidth$", strconv.FormatUint(bandwidth, 10))

				resolvedMediaURL, err := baseURL.Parse(repBaseURL + mediaURL)
				if err != nil {
					return nil, fmt.Errorf("error resolving media URL '%s': %w", repBaseURL+mediaURL, err)
				}
				segmentURLs = append(segmentURLs, resolvedMediaURL.String())

				// Increment time for the next segment (unless it's the last repeat of this S element and next S has a 't')
				currentTime += s.D
				currentNumber++
			}
		}
	} else if st.Duration > 0 && st.Media != "" {
		// Simple duration-based SegmentTemplate (no SegmentTimeline)
		// We need to determine how many segments to generate. Without a total duration,
		// we'll make an assumption for this script. In a real scenario, this would come
		// from Period@duration or other means.
		// For now, let's generate an arbitrary number of segments, e.g., 10,
		// ensuring $Number$ increments by 1.
		const defaultNumSegments = 10 // Arbitrary number for demonstration
		currentTime := uint64(0)
		for i := uint64(0); i < defaultNumSegments; i++ {
			segmentNumber := startNumber + i
			mediaURL := strings.ReplaceAll(st.Media, "$Number$", strconv.FormatUint(segmentNumber, 10))
			mediaURL = strings.ReplaceAll(mediaURL, "$Time$", strconv.FormatUint(currentTime, 10))
			mediaURL = strings.ReplaceAll(mediaURL, "$RepresentationID$", representationID)
			mediaURL = strings.ReplaceAll(mediaURL, "$Bandwidth$", strconv.FormatUint(bandwidth, 10))

			resolvedMediaURL, err := baseURL.Parse(repBaseURL + mediaURL)
			if err != nil {
				return nil, fmt.Errorf("error resolving media URL '%s': %w", repBaseURL+mediaURL, err)
			}
			segmentURLs = append(segmentURLs, resolvedMediaURL.String())
			currentTime += st.Duration // Increment time by the fixed segment duration
		}
	} else if st.Media == "" && st.Initialization == "" {
		return nil, fmt.Errorf("SegmentTemplate for Representation %s has no media, timeline, or initialization URL", representationID)
	}

	return segmentURLs, nil
}

// generateSegmentListURLs generates segment URLs based on a SegmentList.
func generateSegmentListURLs(baseURL *url.URL, repBaseURL string, sl *SegmentList) ([]string, error) {
	var segmentURLs []string
	for _, segURL := range sl.SegmentURLs {
		// SegmentList's SegmentURLs are usually relative to the Representation's BaseURL
		resolvedMediaURL, err := baseURL.Parse(repBaseURL + segURL.Media)
		if err != nil {
			return nil, fmt.Errorf("error resolving SegmentList media URL '%s': %w", repBaseURL+segURL.Media, err)
		}
		segmentURLs = append(segmentURLs, resolvedMediaURL.String())
	}
	return segmentURLs, nil
}
