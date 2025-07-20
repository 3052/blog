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

// MPD represents the top-level element of a Media Presentation Description.
type MPD struct {
	XMLName  xml.Name `xml:"MPD"`
	BaseURLs []BaseURL  `xml:"BaseURL"`
	Periods  []Period   `xml:"Period"`
}

// BaseURL represents a base URL.
type BaseURL struct {
	XMLName xml.Name `xml:"BaseURL"`
	URL     string   `xml:",chardata"`
}

// Period represents a period in the media presentation.
type Period struct {
	XMLName        xml.Name       `xml:"Period"`
	ID             string         `xml:"id,attr"`
	Duration       string         `xml:"duration,attr"` // xs:duration format
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet represents a set of interchangeable Representations.
type AdaptationSet struct {
	XMLName        xml.Name        `xml:"AdaptationSet"`
	ID             string          `xml:"id,attr"`
	BaseURLs       []BaseURL       `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"` // Can be here or on Representation
	Representations []Representation `xml:"Representation"`
}

// Representation represents a single media stream.
type Representation struct {
	XMLName        xml.Name        `xml:"Representation"`
	ID             string          `xml:"id,attr"`
	BaseURLs       []BaseURL       `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"` // Can be here or on AdaptationSet
	SegmentList     *SegmentList    `xml:"SegmentList"`
}

// SegmentTemplate defines the segment naming convention and timing.
type SegmentTemplate struct {
	XMLName       xml.Name         `xml:"SegmentTemplate"`
	Media         string           `xml:"media,attr"`
	Initialization string          `xml:"initialization,attr"`
	StartNumber   string           `xml:"startNumber,attr"` // string to allow checking for existence
	Timescale     string           `xml:"timescale,attr"`   // string to allow checking for existence
	Duration      string           `xml:"duration,attr"`    // string to allow checking for existence
	EndNumber     string           `xml:"endNumber,attr"`   // string to allow checking for existence
	SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline defines a list of segments with their durations.
type SegmentTimeline struct {
	XMLName xml.Name `xml:"SegmentTimeline"`
	Segments []S       `xml:"S"`
}

// S represents a segment within a SegmentTimeline.
type S struct {
	XMLName xml.Name `xml:"S"`
	T       string   `xml:"t,attr"` // start time
	D       string   `xml:"d,attr"` // duration
	R       string   `xml:"r,attr"` // repeat count
}

// SegmentList defines a list of segments, often containing SegmentURL elements.
type SegmentList struct {
	XMLName    xml.Name     `xml:"SegmentList"`
	BaseURLs   []BaseURL    `xml:"BaseURL"` // SegmentList can have its own BaseURLs
	SegmentURLs []SegmentURL `xml:"SegmentURL"` // Common child element
	// Other possible children like Initialization, SegmentBase might be here as well
}

// SegmentURL represents a URL for a specific media segment.
type SegmentURL struct {
	XMLName xml.Name `xml:"SegmentURL"`
	Media   string   `xml:"media,attr"` // The URL path for the segment
	Index   string   `xml:"index,attr"` // Optional, points to index segment
}

// parseDuration converts an xs:duration string (e.g., "PT1H2M3.4S") to seconds.
// This is a simplified parser and might not handle all valid xs:duration formats.
func parseDuration(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	// Remove "P" prefix
	s = strings.TrimPrefix(s, "P")

	var totalSeconds float64
	var value float64
	var unit string
	var inValue bool

	for i, r := range s {
		if (r >= '0' && r <= '9') || r == '.' {
			if !inValue {
				inValue = true
			}
			unit += string(r)
		} else {
			if inValue {
				val, err := strconv.ParseFloat(unit, 64)
				if err != nil {
					return 0, fmt.Errorf("invalid duration value '%s': %w", unit, err)
				}
				value = val
				unit = ""
				inValue = false
			}

			switch r {
			case 'Y': // Years
				totalSeconds += value * 365 * 24 * 60 * 60
			case 'M': // Months or Minutes
				if i > 0 && s[i-1] == 'T' { // Is it minutes after 'T'?
					totalSeconds += value * 60
				} else { // It's months
					totalSeconds += value * 30 * 24 * 60 * 60 // Approximation for months
				}
			case 'W': // Weeks
				totalSeconds += value * 7 * 24 * 60 * 60
			case 'D': // Days
				totalSeconds += value * 24 * 60 * 60
			case 'T': // Time designator
				// Reset value/unit for time components
				value = 0
				unit = ""
			case 'H': // Hours
				totalSeconds += value * 60 * 60
			case 'S': // Seconds
				totalSeconds += value
			default:
				return 0, fmt.Errorf("unsupported duration unit '%c' in '%s'", r, s)
			}
		}
	}

	// Handle the last value if it exists (e.g., "PT10S")
	if inValue {
		val, err := strconv.ParseFloat(unit, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration value '%s': %w", unit, err)
		}
		totalSeconds += val
	}

	return totalSeconds, nil
}

// getAbsoluteURL resolves a relative URL against a base URL.
func getAbsoluteURL(base, relative string) (string, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid base URL '%s': %w", base, err)
	}
	relativeURL, err := url.Parse(relative)
	if err != nil {
		return "", fmt.Errorf("invalid relative URL '%s': %w", relative, err)
	}
	return baseURL.ResolveReference(relativeURL).String(), nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run dash_parser.go <path_to_mpd_file>")
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
		fmt.Printf("Error unmarshalling MPD: %v\n", err)
		os.Exit(1)
	}

	// Assume MPD URL is http://test.test/test.mpd
	mpdBaseURLString := "http://test.test/test.mpd"
	// Resolve the base URL for the MPD itself if it has BaseURLs
	if len(mpd.BaseURLs) > 0 {
		resolved, err := getAbsoluteURL(mpdBaseURLString, mpd.BaseURLs[0].URL)
		if err != nil {
			fmt.Printf("Warning: Could not resolve MPD BaseURL: %v. Using default.\n", err)
		} else {
			mpdBaseURLString = resolved
		}
	}

	segmentURLs := make(map[string][]string)

	for _, period := range mpd.Periods {
		periodDurationSeconds := 0.0
		if period.Duration != "" {
			parsedDuration, err := parseDuration(period.Duration)
			if err != nil {
				fmt.Printf("Warning: Could not parse Period duration '%s': %v. Skipping duration-based segment calculation for this period.\n", period.Duration, err)
			} else {
				periodDurationSeconds = parsedDuration
			}
		}

		for _, as := range period.AdaptationSets {
			// Determine the base URL for this AdaptationSet
			currentBaseURL := mpdBaseURLString
			if len(as.BaseURLs) > 0 {
				resolved, err := getAbsoluteURL(currentBaseURL, as.BaseURLs[0].URL)
				if err != nil {
					fmt.Printf("Warning: Could not resolve AdaptationSet BaseURL: %v. Using previous base URL.\n", err)
				} else {
					currentBaseURL = resolved
				}
			}

			// Determine the SegmentTemplate for this AdaptationSet
			asSegmentTemplate := as.SegmentTemplate

			for _, rep := range as.Representations {
				repBaseURL := currentBaseURL
				if len(rep.BaseURLs) > 0 {
					resolved, err := getAbsoluteURL(repBaseURL, rep.BaseURLs[0].URL)
					if err != nil {
						fmt.Printf("Warning: Could not resolve Representation BaseURL: %v. Using previous base URL.\n", err)
					} else {
						repBaseURL = resolved
					}
				}

				// Prioritize Representation's SegmentList, then SegmentTemplate, then BaseURL only
				var segments []string

				if rep.SegmentList != nil {
					// Handle SegmentList
					segmentListBaseURL := repBaseURL
					if len(rep.SegmentList.BaseURLs) > 0 {
						resolved, err := getAbsoluteURL(segmentListBaseURL, rep.SegmentList.BaseURLs[0].URL)
						if err != nil {
							fmt.Printf("Warning: Could not resolve SegmentList BaseURL for Representation %s: %v. Using previous base URL.\n", rep.ID, err)
						} else {
							segmentListBaseURL = resolved
						}
					}

					for _, segURL := range rep.SegmentList.SegmentURLs {
						if segURL.Media != "" {
							absURL, err := getAbsoluteURL(segmentListBaseURL, segURL.Media)
							if err != nil {
								fmt.Printf("Warning: Could not resolve SegmentURL media '%s' for Representation %s: %v\n", segURL.Media, rep.ID, err)
								continue
							}
							segments = append(segments, absURL)
						}
					}
				} else { // No SegmentList, now check SegmentTemplate
					st := rep.SegmentTemplate
					if st == nil {
						st = asSegmentTemplate // Fallback to AdaptationSet's SegmentTemplate
					}

					if st == nil {
						// No SegmentList AND no SegmentTemplate, implies Representation with only BaseURL
						if len(rep.BaseURLs) > 0 {
							segments = []string{repBaseURL}
						} else {
							// Fallback to parent's base URL if no BaseURL in representation
							segments = []string{repBaseURL}
						}
					} else { // SegmentTemplate exists, process it
						timescale := 1
						if st.Timescale != "" {
							ts, err := strconv.Atoi(st.Timescale)
							if err != nil {
								fmt.Printf("Warning: Invalid timescale '%s' for Representation %s. Using default 1.\n", st.Timescale, rep.ID)
							} else {
								timescale = ts
							}
						}

						startNumber := 1
						if st.StartNumber != "" {
							sn, err := strconv.Atoi(st.StartNumber)
							if err != nil {
								fmt.Printf("Warning: Invalid startNumber '%s' for Representation %s. Using default 1.\n", st.StartNumber, rep.ID)
							} else {
								startNumber = sn
							}
						}

						mediaTemplate := st.Media
						if mediaTemplate == "" {
							fmt.Printf("Warning: Missing media template for Representation %s. Skipping.\n", rep.ID)
							continue
						}

						endNumberSpecified := false
						endNumber := 0
						if st.EndNumber != "" {
							parsedEndNum, err := strconv.Atoi(st.EndNumber)
							if err != nil {
								fmt.Printf("Warning: Invalid endNumber '%s' for Representation %s. Ignoring endNumber and trying SegmentTimeline/duration.\n", st.EndNumber, rep.ID)
								// Do not set endNumberSpecified to true, proceed to next condition
							} else {
								endNumber = parsedEndNum
								endNumberSpecified = true
							}
						}

						if endNumberSpecified {
							// Case 1: EndNumber is valid and present
							for i := startNumber; i <= endNumber; i++ {
								segmentURL := strings.Replace(mediaTemplate, "$Number$", strconv.Itoa(i), -1)
								absURL, err := getAbsoluteURL(repBaseURL, segmentURL)
								if err != nil {
									fmt.Printf("Warning: Could not resolve segment URL for Representation %s: %v\n", rep.ID, err)
									continue
								}
								segments = append(segments, absURL)
							}
						} else if st.SegmentTimeline != nil {
							// Case 2: No valid EndNumber, but SegmentTimeline is present
							segmentNum := startNumber
							for _, s := range st.SegmentTimeline.Segments {
								// Use blank identifier as the integer value 'd' is not directly used for URL generation count
								_, err := strconv.Atoi(s.D)
								if err != nil {
									fmt.Printf("Warning: Invalid duration 'd' in SegmentTimeline for Representation %s: %v. Skipping segment.\n", rep.ID, err)
									continue
								}
								r := 0
								if s.R != "" {
									rVal, err := strconv.Atoi(s.R)
									if err != nil {
										fmt.Printf("Warning: Invalid repeat count 'r' in SegmentTimeline for Representation %s: %v. Using default 0.\n", rep.ID, err)
									} else {
										r = rVal
									}
								}

								// Add the current segment
								segmentURL := strings.Replace(mediaTemplate, "$Number$", strconv.Itoa(segmentNum), -1)
								absURL, err := getAbsoluteURL(repBaseURL, segmentURL)
								if err != nil {
									fmt.Printf("Warning: Could not resolve segment URL for Representation %s: %v\n", rep.ID, err)
								} else {
									segments = append(segments, absURL)
								}
								segmentNum++

								// Add repeated segments
								for i := 0; i < r; i++ {
									segmentURL := strings.Replace(mediaTemplate, "$Number$", strconv.Itoa(segmentNum), -1)
									absURL, err := getAbsoluteURL(repBaseURL, segmentURL)
									if err != nil {
										fmt.Printf("Warning: Could not resolve segment URL for Representation %s: %v\n", rep.ID, err)
										continue
									}
									segments = append(segments, absURL)
									segmentNum++
								}
							}
						} else {
							// Case 3: No valid EndNumber, no SegmentTimeline, use duration calculation
							durationPerSegment := 0.0
							if st.Duration != "" {
								dur, err := strconv.ParseFloat(st.Duration, 64)
								if err != nil {
									fmt.Printf("Warning: Invalid segment duration '%s' for Representation %s. Cannot calculate segments based on duration.\n", st.Duration, rep.ID)
									continue
								}
								durationPerSegment = dur
							} else {
								fmt.Printf("Warning: Missing segment duration for Representation %s. Cannot calculate segments based on duration.\n", rep.ID)
								continue
							}

							if periodDurationSeconds == 0 {
								fmt.Printf("Warning: Period duration not available for Representation %s. Cannot calculate segments based on duration.\n", rep.ID)
								continue
							}

							numSegments := int(math.Ceil(periodDurationSeconds * float64(timescale) / durationPerSegment))
							for i := 0; i < numSegments; i++ {
								segmentNum := startNumber + i
								segmentURL := strings.Replace(mediaTemplate, "$Number$", strconv.Itoa(segmentNum), -1)
								absURL, err := getAbsoluteURL(repBaseURL, segmentURL)
								if err != nil {
									fmt.Printf("Warning: Could not resolve segment URL for Representation %s: %v\n", rep.ID, err)
									continue
								}
								segments = append(segments, absURL)
							}
						}
					}
				}
				segmentURLs[rep.ID] = segments
			}
		}
	}

	jsonOutput, err := json.MarshalIndent(segmentURLs, "", "  ")
	if err != nil {
		fmt.Printf("Error marshalling JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonOutput))
}
