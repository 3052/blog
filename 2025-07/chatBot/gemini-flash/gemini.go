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

const defaultMPDBaseURL = "http://test.test/test.mpd"

// Log messages to standard error
func logError(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", a...)
}

func logInfo(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "INFO: "+format+"\n", a...)
}

// MPD represents the top-level structure of a DASH MPD file.
type MPD struct {
	XMLName xml.Name `xml:"MPD"`
	BaseURL string   `xml:"BaseURL"`
	Periods []Period `xml:"Period"`
}

// Period represents a period in the MPD.
type Period struct {
	XMLName        xml.Name       `xml:"Period"`
	BaseURL        string         `xml:"BaseURL"`
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet represents an adaptation set within a period.
type AdaptationSet struct {
	XMLName        xml.Name       `xml:"AdaptationSet"`
	BaseURL        string         `xml:"BaseURL"`
	Representations []Representation `xml:"Representation"`
}

// Representation represents a single media representation.
type Representation struct {
	XMLName        xml.Name        `xml:"Representation"`
	ID             string          `xml:"id,attr"`
	BaseURL        string          `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
	// Add other segment types if needed, e.g., SegmentList, SegmentBase
}

// SegmentTemplate represents segment information using a template.
type SegmentTemplate struct {
	XMLName         xml.Name        `xml:"SegmentTemplate"`
	Initialization string          `xml:"initialization,attr"`
	Media          string          `xml:"media,attr"`
	StartNumber    string          `xml:"startNumber,attr"`
	Timescale      string          `xml:"timescale,attr"`
	SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline represents a detailed timeline of segments.
type SegmentTimeline struct {
	XMLName xml.Name `xml:"SegmentTimeline"`
	Segments []S      `xml:"S"`
}

// S represents a single segment entry in SegmentTimeline.
type S struct {
	XMLName xml.Name `xml:"S"`
	T       string   `xml:"t,attr"` // start time
	D       string   `xml:"d,attr"` // duration
	R       string   `xml:"r,attr"` // repeat count
}

// SegmentURLs represents the output structure.
type SegmentURLs struct {
	RepresentationID string   `json:"representationId"`
	InitializationURL string  `json:"initializationUrl,omitempty"`
	SegmentURLs      []string `json:"segmentUrls"`
}

func main() {
	if len(os.Args) < 2 {
		logError("Usage: %s <path_to_mpd_file.mpd>", os.Args[0])
		os.Exit(1)
	}

	mpdFilePath := os.Args[1]
	mpdContent, err := ioutil.ReadFile(mpdFilePath)
	if err != nil {
		logError("Failed to read MPD file '%s': %v", mpdFilePath, err)
		os.Exit(1)
	}

	var mpd MPD
	err = xml.Unmarshal(mpdContent, &mpd)
	if err != nil {
		logError("Failed to unmarshal MPD XML from '%s': %v", mpdFilePath, err)
		os.Exit(1)
	}

	logInfo("Successfully parsed MPD file: %s", mpdFilePath)

	allSegmentURLs := []SegmentURLs{}

	mpdBaseURL, err := url.Parse(defaultMPDBaseURL)
	if err != nil {
		logError("Failed to parse default MPD base URL '%s': %v", defaultMPDBaseURL, err)
		os.Exit(1)
	}

	// Override with MPD's BaseURL if present
	if mpd.BaseURL != "" {
		parsedMPDBase, err := url.Parse(mpd.BaseURL)
		if err != nil {
			logError("Invalid BaseURL in MPD: %s, using default. Error: %v", mpd.BaseURL, err)
		} else {
			mpdBaseURL = mpdBaseURL.ResolveReference(parsedMPDBase)
		}
	}
	logInfo("Effective MPD Base URL for resolution: %s", mpdBaseURL.String())

	for _, period := range mpd.Periods {
		periodBaseURL := mpdBaseURL
		if period.BaseURL != "" {
			parsedPeriodBase, err := url.Parse(period.BaseURL)
			if err != nil {
				logError("Invalid BaseURL in Period: %s, using parent. Error: %v", period.BaseURL, err)
			} else {
				periodBaseURL = periodBaseURL.ResolveReference(parsedPeriodBase)
			}
		}

		for _, as := range period.AdaptationSets {
			adaptationSetBaseURL := periodBaseURL
			if as.BaseURL != "" {
				parsedASBase, err := url.Parse(as.BaseURL)
				if err != nil {
					logError("Invalid BaseURL in AdaptationSet: %s, using parent. Error: %v", as.BaseURL, err)
				} else {
					adaptationSetBaseURL = adaptationSetBaseURL.ResolveReference(parsedASBase)
				}
			}

			for _, rep := range as.Representations {
				representationBaseURL := adaptationSetBaseURL
				if rep.BaseURL != "" {
					parsedRepBase, err := url.Parse(rep.BaseURL)
					if err != nil {
						logError("Invalid BaseURL in Representation %s: %s, using parent. Error: %v", rep.ID, rep.BaseURL, err)
					} else {
						representationBaseURL = representationBaseURL.ResolveReference(parsedRepBase)
					}
				}

				if rep.SegmentTemplate != nil {
					logInfo("Processing SegmentTemplate for Representation ID: %s", rep.ID)
					var initURL string
					var segments []string

					// Resolve Initialization URL
					if rep.SegmentTemplate.Initialization != "" {
						initRelativeURL, err := url.Parse(rep.SegmentTemplate.Initialization)
						if err != nil {
							logError("Failed to parse initialization URL string '%s' for %s: %v", rep.SegmentTemplate.Initialization, rep.ID, err)
							// Continue with empty string for initURL or handle as appropriate
						} else {
							resolvedInitURL := representationBaseURL.ResolveReference(initRelativeURL).String()
							initURL = resolvedInitURL
							logInfo("Resolved Initialization URL for %s: %s", rep.ID, initURL)
						}
					}

					// Generate Segment URLs
					if rep.SegmentTemplate.SegmentTimeline != nil {
						// Handle SegmentTimeline
						logInfo("Generating Segment URLs using SegmentTimeline for %s", rep.ID)
						currentNumber := 0
						var currentTime int64 = 0

						for i, s := range rep.SegmentTemplate.SegmentTimeline.Segments {
							duration, err := strconv.ParseInt(s.D, 10, 64)
							if err != nil {
								logError("Failed to parse segment duration 'd' for %s, segment %d: %v", rep.ID, i, err)
								continue
							}

							if s.T != "" {
								t, err := strconv.ParseInt(s.T, 10, 64)
								if err != nil {
									logError("Failed to parse segment start time 't' for %s, segment %d: %v", rep.ID, i, err)
								} else {
									currentTime = t
								}
							}

							repeat := 0
							if s.R != "" {
								r, err := strconv.Atoi(s.R)
								if err != nil {
									logError("Failed to parse segment repeat 'r' for %s, segment %d: %v", rep.ID, i, err)
								} else {
									repeat = r
								}
							}

							for j := 0; j <= repeat; j++ {
								segmentURLTemplate := rep.SegmentTemplate.Media
								
								if strings.Contains(segmentURLTemplate, "$Number$") {
									startNum := 1
									if rep.SegmentTemplate.StartNumber != "" {
										parsedStartNum, err := strconv.Atoi(rep.SegmentTemplate.StartNumber)
										if err != nil {
											logError("Failed to parse StartNumber for %s: %v, using 1", rep.ID, err)
										} else {
											startNum = parsedStartNum
										}
									}
									segmentURLTemplate = strings.Replace(segmentURLTemplate, "$Number$", strconv.Itoa(startNum+currentNumber), -1)
								}
								
								if strings.Contains(segmentURLTemplate, "$Time$") {
									segmentURLTemplate = strings.Replace(segmentURLTemplate, "$Time$", strconv.FormatInt(currentTime, 10), -1)
								}

								segmentRelativeURL, err := url.Parse(segmentURLTemplate)
								if err != nil {
									logError("Failed to parse segment URL template string '%s' for %s: %v", segmentURLTemplate, rep.ID, err)
									continue
								}

								resolvedSegmentURL := representationBaseURL.ResolveReference(segmentRelativeURL).String()
								segments = append(segments, resolvedSegmentURL)
								logInfo("Generated Segment URL for %s (timeline, index %d, repeat %d): %s", rep.ID, i, j, resolvedSegmentURL)
								currentTime += duration
								currentNumber++
							}
						}

					} else if strings.Contains(rep.SegmentTemplate.Media, "$Number$") {
						// Handle SegmentTemplate with $Number$ (without SegmentTimeline)
						logInfo("Generating Segment URLs using $Number$ substitution for %s", rep.ID)
						startNumber := 1
						if rep.SegmentTemplate.StartNumber != "" {
							parsedStartNumber, err := strconv.Atoi(rep.SegmentTemplate.StartNumber)
							if err != nil {
								logError("Failed to parse StartNumber for %s: %v, using 1", rep.ID, err)
							} else {
								startNumber = parsedStartNumber
							}
						}

						// For demonstration, let's generate a few segments (e.g., 5 segments)
						numSegmentsToGenerate := 5 

						for i := 0; i < numSegmentsToGenerate; i++ {
							segmentURLTemplate := strings.Replace(rep.SegmentTemplate.Media, "$Number$", strconv.Itoa(startNumber+i), -1)
							
							segmentRelativeURL, err := url.Parse(segmentURLTemplate)
							if err != nil {
								logError("Failed to parse segment URL template string '%s' for %s: %v", segmentURLTemplate, rep.ID, err)
								continue
							}

							resolvedSegmentURL := representationBaseURL.ResolveReference(segmentRelativeURL).String()
							segments = append(segments, resolvedSegmentURL)
							logInfo("Generated Segment URL for %s ($Number$ substituted): %s", rep.ID, resolvedSegmentURL)
						}
					} else {
						// Handle SegmentTemplate with a single media file (e.g., on-demand single file)
						logInfo("Generating single Segment URL for %s", rep.ID)
						segmentRelativeURL, err := url.Parse(rep.SegmentTemplate.Media)
						if err != nil {
							logError("Failed to parse media URL string '%s' for %s: %v", rep.SegmentTemplate.Media, rep.ID, err)
						} else {
							resolvedSegmentURL := representationBaseURL.ResolveReference(segmentRelativeURL).String()
							segments = append(segments, resolvedSegmentURL)
							logInfo("Generated single Segment URL for %s: %s", rep.ID, resolvedSegmentURL)
						}
					}

					allSegmentURLs = append(allSegmentURLs, SegmentURLs{
						RepresentationID: rep.ID,
						InitializationURL: initURL,
						SegmentURLs:      segments,
					})
				} else if rep.BaseURL != "" { 
					logInfo("Processing Representation with BaseURL (no SegmentTemplate) for ID: %s. Using the pre-resolved representationBaseURL.", rep.ID)
					// The representationBaseURL already incorporates rep.BaseURL correctly.
					// Directly use it as the segment/initialization URL for this representation.
					initOrSegmentURL := representationBaseURL.String()
					allSegmentURLs = append(allSegmentURLs, SegmentURLs{
						RepresentationID: rep.ID,
						SegmentURLs:      []string{initOrSegmentURL},
					})
					logInfo("Treated pre-resolved representationBaseURL as segment/initialization URL for %s: %s", rep.ID, initOrSegmentURL)
				} else {
					logInfo("Representation %s does not have SegmentTemplate or BaseURL. Skipping segment URL extraction.", rep.ID)
				}
			}
		}
	}

	jsonData, err := json.MarshalIndent(allSegmentURLs, "", "  ")
	if err != nil {
		logError("Failed to marshal segment URLs to JSON: %v", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonData))
}
