package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MPD represents the top-level structure of a DASH MPD file.
type MPD struct {
	XMLName  xml.Name `xml:"MPD"`
	BaseURL  string   `xml:"BaseURL"`
	Periods  []Period `xml:"Period"`
	MediaPresentationDuration string `xml:"mediaPresentationDuration,attr"`
}

// Period represents a Period element within the MPD.
type Period struct {
	XMLName      xml.Name       `xml:"Period"`
	ID           string         `xml:"id,attr"`
	Start        string         `xml:"start,attr"`
	Duration     string         `xml:"duration,attr"`
	BaseURL      string         `xml:"BaseURL"`
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet represents an AdaptationSet element.
type AdaptationSet struct {
	XMLName         xml.Name         `xml:"AdaptationSet"`
	ID              string           `xml:"id,attr"`
	BaseURL         string           `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
	Representations []Representation `xml:"Representation"`
}

// Representation represents a Representation element.
type Representation struct {
	XMLName         xml.Name         `xml:"Representation"`
	ID              string           `xml:"id,attr"`
	BaseURL         string           `xml:"BaseURL"`
	SegmentBase     *SegmentBase     `xml:"SegmentBase"`
	SegmentList     *SegmentList     `xml:"SegmentList"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

// SegmentBase represents a SegmentBase element.
type SegmentBase struct {
	XMLName xml.Name `xml:"SegmentBase"`
	BaseURL string   `xml:"BaseURL"` // Not explicitly in DASH, but for consistency if we consider inherited BaseURL
}

// SegmentList represents a SegmentList element.
type SegmentList struct {
	XMLName xml.Name `xml:"SegmentList"`
	BaseURL string   `xml:"BaseURL"` // Not explicitly in DASH, but for consistency if we consider inherited BaseURL
	SegmentURLs []SegmentURL `xml:"SegmentURL"`
}

// SegmentURL represents a SegmentURL element within SegmentList.
type SegmentURL struct {
	XMLName xml.Name `xml:"SegmentURL"`
	Media   string   `xml:"media,attr"`
}

// SegmentTemplate represents a SegmentTemplate element.
type SegmentTemplate struct {
	XMLName       xml.Name       `xml:"SegmentTemplate"`
	Media         string         `xml:"media,attr"`
	Initialization string        `xml:"initialization,attr"`
	Timescale     uint64         `xml:"timescale,attr"`
	Duration      uint64         `xml:"duration,attr"`
	StartNumber   uint64         `xml:"startNumber,attr"`
	EndNumber     uint64         `xml:"endNumber,attr"`
	SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline represents a SegmentTimeline element.
type SegmentTimeline struct {
	XMLName xml.Name `xml:"SegmentTimeline"`
	Ss      []S      `xml:"S"`
}

// S represents an S element within SegmentTimeline.
type S struct {
	XMLName xml.Name `xml:"S"`
	T       uint64   `xml:"t,attr"` // Start time in units of timescale
	D       uint64   `xml:"d,attr"` // Duration in units of timescale
	R       int64    `xml:"r,attr"` // Repeat count
}

// asSeconds converts an XML duration string (e.g., "PT1H2M3S") to seconds.
func asSeconds(durationStr string) (float64, error) {
	if durationStr == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(strings.ToLower(strings.ReplaceAll(durationStr, "PT", "")))
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration '%s': %w", durationStr, err)
	}
	return d.Seconds(), nil
}

// calculateSegmentURLs generates segment URLs for a given Representation based on its SegmentTemplate.
func calculateSegmentURLs(mpdURL *url.URL, periodBaseURL *url.URL, asBaseURL *url.URL, repBaseURL *url.URL, repID string, st *SegmentTemplate, duration float64) ([]string, error) {
	var urls []string

	timescale := uint64(1)
	if st.Timescale != 0 {
		timescale = st.Timescale
	}

	startNumber := uint64(1)
	if st.StartNumber != 0 {
		startNumber = st.StartNumber
	}

	if st.SegmentTimeline != nil && len(st.SegmentTimeline.Ss) > 0 {
		currentNumber := startNumber
		for _, s := range st.SegmentTimeline.Ss {
			numRepeats := s.R
			if numRepeats == -1 { // Indefinite repeat, usually for live, but here we cap it
				numRepeats = 0 // In a static MPD, -1 usually means one segment
			}
			for i := int64(0); i <= numRepeats; i++ {
				segmentURL := strings.ReplaceAll(st.Media, "$RepresentationID$", repID)
				segmentURL = strings.ReplaceAll(segmentURL, "$Number$", strconv.FormatUint(currentNumber, 10))

				baseURL := mpdURL
				if periodBaseURL != nil {
					baseURL = periodBaseURL
				}
				if asBaseURL != nil {
					baseURL = asBaseURL
				}
				if repBaseURL != nil {
					baseURL = repBaseURL
				}

				resolvedURL, err := baseURL.Parse(segmentURL)
				if err != nil {
					return nil, fmt.Errorf("failed to parse segment URL '%s': %w", segmentURL, err)
				}
				urls = append(urls, resolvedURL.String())
				currentNumber++
			}
		}
	} else if st.Duration != 0 {
		var endNumber uint64
		if st.EndNumber != 0 {
			endNumber = st.EndNumber
		} else {
			if duration == 0 {
				return nil, fmt.Errorf("period duration is zero, cannot calculate endNumber without SegmentTimeline")
			}
			totalSegments := math.Ceil(duration * float64(timescale) / float64(st.Duration))
			endNumber = startNumber + uint64(totalSegments) - 1
		}

		for i := startNumber; i <= endNumber; i++ {
			segmentURL := strings.ReplaceAll(st.Media, "$RepresentationID$", repID)
			segmentURL = strings.ReplaceAll(segmentURL, "$Number$", strconv.FormatUint(i, 10))

			baseURL := mpdURL
			if periodBaseURL != nil {
				baseURL = periodBaseURL
			}
			if asBaseURL != nil {
				baseURL = asBaseURL
			}
			if repBaseURL != nil {
				baseURL = repBaseURL
			}

			resolvedURL, err := baseURL.Parse(segmentURL)
			if err != nil {
				return nil, fmt.Errorf("failed to parse segment URL '%s': %w", segmentURL, err)
			}
			urls = append(urls, resolvedURL.String())
		}
	} else {
		return nil, fmt.Errorf("SegmentTemplate requires either @duration or SegmentTimeline")
	}

	return urls, nil
}

// parseMPD parses a local DASH MPD file and extracts segment URLs.
func parseMPD(mpdFilePath string) (map[string][]string, error) {
	data, err := ioutil.ReadFile(mpdFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read MPD file: %w", err)
	}

	var mpd MPD
	err = xml.Unmarshal(data, &mpd)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal MPD XML: %w", err)
	}

	output := make(map[string][]string)

	mpdURL, err := url.Parse("http://test.test/test.mpd")
	if err != nil {
		return nil, fmt.Errorf("failed to parse base MPD URL: %w", err)
	}

	for _, period := range mpd.Periods {
		var periodBaseURL *url.URL
		if period.BaseURL != "" {
			periodBaseURL, err = mpdURL.Parse(period.BaseURL)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error parsing Period BaseURL:", err)
				periodBaseURL = nil // Continue without this BaseURL
			}
		}

		periodDuration := 0.0
		if period.Duration != "" {
			periodDuration, err = asSeconds(period.Duration)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Warning: Could not parse Period duration:", err)
				periodDuration = 0 // Treat as unknown duration
			}
		} else if mpd.MediaPresentationDuration != "" {
			// If Period@duration is missing, fall back to MPD@mediaPresentationDuration
			// This is a simplification and might not be fully accurate for multi-period MPDs
			periodDuration, err = asSeconds(mpd.MediaPresentationDuration)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Warning: Could not parse MPD mediaPresentationDuration:", err)
				periodDuration = 0 // Treat as unknown duration
			}
		}


		for _, as := range period.AdaptationSets {
			var asBaseURL *url.URL
			if as.BaseURL != "" {
				base := mpdURL
				if periodBaseURL != nil {
					base = periodBaseURL
				}
				asBaseURL, err = base.Parse(as.BaseURL)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error parsing AdaptationSet BaseURL:", err)
					asBaseURL = nil
				}
			}

			for _, rep := range as.Representations {
				var repBaseURL *url.URL
				if rep.BaseURL != "" {
					base := mpdURL
					if periodBaseURL != nil {
						base = periodBaseURL
					}
					if asBaseURL != nil {
						base = asBaseURL
					}
					repBaseURL, err = base.Parse(rep.BaseURL)
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error parsing Representation BaseURL:", err)
						repBaseURL = nil
					}
				}

				if rep.SegmentBase != nil || rep.SegmentList != nil || rep.SegmentTemplate != nil || as.SegmentTemplate != nil {
					// Handle SegmentList
					if rep.SegmentList != nil {
						for _, segURL := range rep.SegmentList.SegmentURLs {
							baseURL := mpdURL
							if periodBaseURL != nil {
								baseURL = periodBaseURL
							}
							if asBaseURL != nil {
								baseURL = asBaseURL
							}
							if repBaseURL != nil {
								baseURL = repBaseURL
							}

							resolvedURL, err := baseURL.Parse(segURL.Media)
							if err != nil {
								fmt.Fprintln(os.Stderr, "Error parsing SegmentList media URL:", err)
								continue
							}
							output[rep.ID] = append(output[rep.ID], resolvedURL.String())
						}
						continue
					}

					// Prioritize Representation's SegmentTemplate
					if rep.SegmentTemplate != nil {
						urls, err := calculateSegmentURLs(mpdURL, periodBaseURL, asBaseURL, repBaseURL, rep.ID, rep.SegmentTemplate, periodDuration)
						if err != nil {
							fmt.Fprintln(os.Stderr, "Error calculating segment URLs for Representation", rep.ID, ":", err)
							continue
						}
						output[rep.ID] = append(output[rep.ID], urls...)
						continue
					}

					// Fallback to AdaptationSet's SegmentTemplate
					if as.SegmentTemplate != nil {
						urls, err := calculateSegmentURLs(mpdURL, periodBaseURL, asBaseURL, repBaseURL, rep.ID, as.SegmentTemplate, periodDuration)
						if err != nil {
							fmt.Fprintln(os.Stderr, "Error calculating segment URLs for AdaptationSet's template (Representation", rep.ID, "):", err)
							continue
						}
						output[rep.ID] = append(output[rep.ID], urls...)
						continue
					}
				} else {
					// If missing SegmentBase, SegmentList, SegmentTemplate, return Representation@BaseURL
					baseURL := mpdURL
					if periodBaseURL != nil {
						baseURL = periodBaseURL
					}
					if asBaseURL != nil {
						baseURL = asBaseURL
					}
					if repBaseURL != nil {
						baseURL = repBaseURL
					}

					if repBaseURL != nil {
						output[rep.ID] = append(output[rep.ID], repBaseURL.String())
					} else if asBaseURL != nil {
						output[rep.ID] = append(output[rep.ID], asBaseURL.String())
					} else if periodBaseURL != nil {
						output[rep.ID] = append(output[rep.ID], periodBaseURL.String())
					} else {
						output[rep.ID] = append(output[rep.ID], baseURL.String()) // Fallback to MPD base URL if no other is found.
					}
				}
			}
		}
	}

	return output, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: go run main.go <path_to_mpd_file>")
		os.Exit(1)
	}

	mpdFilePath := os.Args[1]
	absPath, err := filepath.Abs(mpdFilePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting absolute path:", err)
		os.Exit(1)
	}

	segmentURLsMap, err := parseMPD(absPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing MPD:", err)
		os.Exit(1)
	}

	jsonData, err := json.MarshalIndent(segmentURLsMap, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error marshaling JSON:", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonData))
}
