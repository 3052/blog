package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// MPD represents the root element of a DASH MPD file.
type MPD struct {
	XMLName  xml.Name `xml:"MPD"`
	BaseURL  string   `xml:"BaseURL"`
	Periods  []Period `xml:"Period"`
	XMLNs    string   `xml:"xmlns,attr"`
	Type     string   `xml:"type,attr"`
	MediaPresentationDuration string `xml:"mediaPresentationDuration,attr"`
}

// Period represents a period in the MPD.
type Period struct {
	XMLName      xml.Name       `xml:"Period"`
	ID           string         `xml:"id,attr"`
	Duration     string         `xml:"duration,attr"`
	BaseURL      string         `xml:"BaseURL"`
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet represents an adaptation set within a period.
type AdaptationSet struct {
	XMLName        xml.Name         `xml:"AdaptationSet"`
	ID             string           `xml:"id,attr"`
	BaseURL        string           `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"` // Can be child of AdaptationSet
	Representations []Representation `xml:"Representation"`
}

// Representation represents a single media representation.
type Representation struct {
	XMLName        xml.Name         `xml:"Representation"`
	ID             string           `xml:"id,attr"`
	BaseURL        string           `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"` // Can be child of Representation
	Bandwidth      int              `xml:"bandwidth,attr"`
	MimeType       string           `xml:"mimeType,attr"`
	Codecs         string           `xml:"codecs,attr"`
}

// SegmentTemplate defines the segment URL pattern.
type SegmentTemplate struct {
	XMLName      xml.Name      `xml:"SegmentTemplate"`
	Media        string        `xml:"media,attr"`
	Initialization string      `xml:"initialization,attr"`
	StartNumber  string        `xml:"startNumber,attr"`
	Timescale    string        `xml:"timescale,attr"`
	Duration     string        `xml:"duration,attr"` // Segment duration
	EndNumber    string        `xml:"endNumber,attr"`
	SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline defines a list of segments.
type SegmentTimeline struct {
	XMLName xml.Name `xml:"SegmentTimeline"`
	S       []S      `xml:"S"`
}

// S represents a segment in the timeline.
type S struct {
	XMLName xml.Name `xml:"S"`
	T       string   `xml:"t,attr"` // Start time
	D       string   `xml:"d,attr"` // Duration
	R       string   `xml:"r,attr"` // Repeat count
}

// parseDuration parses an ISO 8601 duration string (e.g., "PT10S") into seconds.
// This is a simplified parser and may not handle all edge cases of ISO 8601.
func parseDuration(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	// Remove "P" prefix for period
	s = strings.TrimPrefix(s, "P")

	var totalSeconds float64

	// Handle Time components (T)
	if strings.Contains(s, "T") {
		parts := strings.Split(s, "T")
		datePart := parts[0]
		timePart := parts[1]

		// Parse date part (for days)
		if strings.Contains(datePart, "D") {
			dParts := strings.Split(datePart, "D")
			days, err := strconv.ParseFloat(dParts[0], 64)
			if err != nil {
				return 0, fmt.Errorf("invalid days in duration: %w", err)
			}
			totalSeconds += days * 24 * 3600
		}

		// Parse time part
		if strings.Contains(timePart, "H") {
			hParts := strings.Split(timePart, "H")
			hours, err := strconv.ParseFloat(hParts[0], 64)
			if err != nil {
				return 0, fmt.Errorf("invalid hours in duration: %w", err)
			}
			totalSeconds += hours * 3600
			timePart = hParts[1]
		}
		if strings.Contains(timePart, "M") {
			mParts := strings.Split(timePart, "M")
			minutes, err := strconv.ParseFloat(mParts[0], 64)
			if err != nil {
				return 0, fmt.Errorf("invalid minutes in duration: %w", err)
			}
			totalSeconds += minutes * 60
			timePart = mParts[1]
		}
		if strings.Contains(timePart, "S") {
			sParts := strings.Split(timePart, "S")
			seconds, err := strconv.ParseFloat(sParts[0], 64)
			if err != nil {
				return 0, fmt.Errorf("invalid seconds in duration: %w", err)
			}
			totalSeconds += seconds
		}
	} else {
		// Only date part, or simple seconds (e.g., "10S")
		if strings.Contains(s, "D") {
			dParts := strings.Split(s, "D")
			days, err := strconv.ParseFloat(dParts[0], 64)
			if err != nil {
				return 0, fmt.Errorf("invalid days in duration: %w", err)
			}
			totalSeconds += days * 24 * 3600
		} else if strings.HasSuffix(s, "S") {
			seconds, err := strconv.ParseFloat(strings.TrimSuffix(s, "S"), 64)
			if err != nil {
				return 0, fmt.Errorf("invalid seconds in duration: %w", err)
			}
			totalSeconds += seconds
		}
	}

	return totalSeconds, nil
}

// resolveURL resolves a relative URL against a base URL.
func resolveURL(baseURL *url.URL, relativePath string) string {
	if relativePath == "" {
		return baseURL.String()
	}
	rel, err := url.Parse(relativePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing relative path %s: %v\n", relativePath, err)
		return ""
	}
	return baseURL.ResolveReference(rel).String()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <path_to_mpd_file>\n", os.Args[0])
		os.Exit(1)
	}

	mpdFilePath := os.Args[1]

	// Read the MPD file
	xmlFile, err := os.Open(mpdFilePath)
	if err != nil {
		log.Fatalf("Error opening MPD file %s: %v\n", mpdFilePath, err)
	}
	defer xmlFile.Close()

	byteValue, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		log.Fatalf("Error reading MPD file %s: %v\n", mpdFilePath, err)
	}

	var mpd MPD
	err = xml.Unmarshal(byteValue, &mpd)
	if err != nil {
		log.Fatalf("Error unmarshaling MPD XML: %v\n", err)
	}

	// Assume MPD URL is http://test.test/test.mpd
	mpdBaseURL, err := url.Parse("http://test.test/test.mpd")
	if err != nil {
		log.Fatalf("Error parsing base MPD URL: %v\n", err)
	}

	// Initialize results map
	segmentURLs := make(map[string][]string)

	// Traverse the MPD structure
	for _, period := range mpd.Periods {
		// Resolve Period BaseURL
		currentPeriodBaseURL := mpdBaseURL
		if mpd.BaseURL != "" {
			currentPeriodBaseURL, _ = url.Parse(resolveURL(currentPeriodBaseURL, mpd.BaseURL))
		}
		if period.BaseURL != "" {
			currentPeriodBaseURL, _ = url.Parse(resolveURL(currentPeriodBaseURL, period.BaseURL))
		}

		periodDurationSeconds := 0.0
		if period.Duration != "" {
			d, err := parseDuration(period.Duration)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Could not parse period duration '%s': %v\n", period.Duration, err)
			} else {
				periodDurationSeconds = d
			}
		}

		for _, as := range period.AdaptationSets {
			// Resolve AdaptationSet BaseURL
			currentASBaseURL := currentPeriodBaseURL
			if as.BaseURL != "" {
				currentASBaseURL, _ = url.Parse(resolveURL(currentASBaseURL, as.BaseURL))
			}

			// Get SegmentTemplate from AdaptationSet if present
			asSegmentTemplate := as.SegmentTemplate

			for _, rep := range as.Representations {
				// Resolve Representation BaseURL
				currentRepBaseURL := currentASBaseURL
				if rep.BaseURL != "" {
					currentRepBaseURL, _ = url.Parse(resolveURL(currentRepBaseURL, rep.BaseURL))
				}

				// Determine the effective SegmentTemplate (Representation's takes precedence)
				effectiveSegmentTemplate := rep.SegmentTemplate
				if effectiveSegmentTemplate == nil {
					effectiveSegmentTemplate = asSegmentTemplate
				}

				if effectiveSegmentTemplate == nil {
					// Case 8: Representation with only BaseURL means one segment
					if rep.BaseURL != "" {
						segmentURLs[rep.ID] = []string{resolveURL(currentRepBaseURL, "")}
					} else {
						// If no BaseURL and no SegmentTemplate, it's an error or malformed MPD for segment retrieval
						fmt.Fprintf(os.Stderr, "Warning: Representation %s has no BaseURL and no SegmentTemplate. Skipping.\n", rep.ID)
					}
					continue // Move to the next representation
				}

				// Parse SegmentTemplate attributes with defaults
				timescale := 1.0
				if effectiveSegmentTemplate.Timescale != "" {
					t, err := strconv.ParseFloat(effectiveSegmentTemplate.Timescale, 64)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: Invalid timescale '%s' for Representation %s, using default 1: %v\n", effectiveSegmentTemplate.Timescale, rep.ID, err)
					} else {
						timescale = t
					}
				}

				startNumber := 1
				if effectiveSegmentTemplate.StartNumber != "" {
					sN, err := strconv.Atoi(effectiveSegmentTemplate.StartNumber)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: Invalid startNumber '%s' for Representation %s, using default 1: %v\n", effectiveSegmentTemplate.StartNumber, rep.ID, err)
					} else {
						startNumber = sN
					}
				}

				var repSegmentURLs []string

				if effectiveSegmentTemplate.EndNumber != "" {
					// Case 14: SegmentTemplate@endNumber defines the last segment
					endNumber, err := strconv.Atoi(effectiveSegmentTemplate.EndNumber)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: Invalid endNumber '%s' for Representation %s. Falling back to SegmentTimeline/duration calculation: %v\n", effectiveSegmentTemplate.EndNumber, rep.ID, err)
						// Fallback logic will be handled below if SegmentTimeline or Duration exists
					} else {
						for i := startNumber; i <= endNumber; i++ {
							segmentName := strings.Replace(effectiveSegmentTemplate.Media, "$Number$", strconv.Itoa(i), -1)
							segmentName = strings.Replace(segmentName, "$RepresentationID$", rep.ID, -1) // Handle $RepresentationID$ if present
							fullURL := resolveURL(currentRepBaseURL, segmentName)
							if fullURL != "" {
								repSegmentURLs = append(repSegmentURLs, fullURL)
							}
						}
					}
				}

				if len(repSegmentURLs) == 0 { // If endNumber didn't resolve or was invalid
					if effectiveSegmentTemplate.SegmentTimeline != nil && len(effectiveSegmentTemplate.SegmentTimeline.S) > 0 {
						// Case 15: Use SegmentTimeline
						currentSegmentNumber := startNumber
						for _, s := range effectiveSegmentTemplate.SegmentTimeline.S {
							// Parse 'd' (duration) but the value itself is not used for segment URL generation
							// as per the requirement for $Number$ increment based on repeat counts.
							_, err := strconv.ParseFloat(s.D, 64)
							if err != nil {
								fmt.Fprintf(os.Stderr, "Warning: Invalid segment duration 'd' in SegmentTimeline for Representation %s: %v\n", rep.ID, err)
								continue
							}
							r := 0 // Default repeat count
							if s.R != "" {
								rInt, err := strconv.Atoi(s.R)
								if err != nil {
									fmt.Fprintf(os.Stderr, "Warning: Invalid repeat count 'r' in SegmentTimeline for Representation %s: %v\n", rep.ID, err)
								} else {
									r = rInt
								}
							}

							// Add the current segment
							segmentName := strings.Replace(effectiveSegmentTemplate.Media, "$Number$", strconv.Itoa(currentSegmentNumber), -1)
							segmentName = strings.Replace(segmentName, "$RepresentationID$", rep.ID, -1)
							fullURL := resolveURL(currentRepBaseURL, segmentName)
							if fullURL != "" {
								repSegmentURLs = append(repSegmentURLs, fullURL)
							}
							currentSegmentNumber++

							// Add repeated segments
							for j := 0; j < r; j++ {
								segmentName = strings.Replace(effectiveSegmentTemplate.Media, "$Number$", strconv.Itoa(currentSegmentNumber), -1)
								segmentName = strings.Replace(segmentName, "$RepresentationID$", rep.ID, -1)
								fullURL = resolveURL(currentRepBaseURL, segmentName)
								if fullURL != "" {
									repSegmentURLs = append(repSegmentURLs, fullURL)
								}
								currentSegmentNumber++
							}
						}
					} else if effectiveSegmentTemplate.Duration != "" && periodDurationSeconds > 0 {
						// Case 16: No SegmentTimeline, use Period@duration and SegmentTemplate@duration
						segmentDuration, err := strconv.ParseFloat(effectiveSegmentTemplate.Duration, 64)
						if err != nil {
							fmt.Fprintf(os.Stderr, "Warning: Invalid segment duration '%s' for Representation %s. Cannot calculate segments: %v\n", effectiveSegmentTemplate.Duration, rep.ID, err)
						} else if segmentDuration > 0 {
							numSegments := int(math.Ceil(periodDurationSeconds * timescale / segmentDuration))
							for i := 0; i < numSegments; i++ {
								segmentNumber := startNumber + i
								segmentName := strings.Replace(effectiveSegmentTemplate.Media, "$Number$", strconv.Itoa(segmentNumber), -1)
								segmentName = strings.Replace(segmentName, "$RepresentationID$", rep.ID, -1)
								fullURL := resolveURL(currentRepBaseURL, segmentName)
								if fullURL != "" {
									repSegmentURLs = append(repSegmentURLs, fullURL)
								}
							}
						}
					} else {
						fmt.Fprintf(os.Stderr, "Warning: Representation %s has SegmentTemplate but no endNumber, SegmentTimeline, or calculable duration. Skipping segment generation.\n", rep.ID)
					}
				}
				segmentURLs[rep.ID] = repSegmentURLs
			}
		}
	}

	// Marshal the results to JSON and print
	jsonData, err := json.MarshalIndent(segmentURLs, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v\n", err)
	}

	fmt.Println(string(jsonData))
}
