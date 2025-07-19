package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"math"
)

const defaultMPDBaseURL = "http://test.test/test.mpd"
const logFileName = "log.txt"

var logFile *os.File
var multiWriter io.Writer // To write to both file and stderr

// Log messages to standard error and log.txt
func logError(format string, a ...interface{}) {
	if multiWriter != nil {
		fmt.Fprintf(multiWriter, "ERROR: "+format+"\n", a...)
	} else {
		fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", a...)
	}
}

func logInfo(format string, a ...interface{}) {
	if multiWriter != nil {
		fmt.Fprintf(multiWriter, "INFO: "+format+"\n", a...)
	} else {
		fmt.Fprintf(os.Stderr, "INFO: "+format+"\n", a...)
	}
}

// MPD represents the top-level structure of a DASH MPD file.
type MPD struct {
	XMLName xml.Name `xml:"MPD"`
	BaseURL string   `xml:"BaseURL"`
	MediaPresentationDuration string `xml:"mediaPresentationDuration,attr"`
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
	XMLName        xml.Name        `xml:"AdaptationSet"`
	BaseURL        string          `xml:"BaseURL"`
	Representations []Representation `xml:"Representation"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

// Representation represents a single media representation.
type Representation struct {
	XMLName        xml.Name        `xml:"Representation"`
	ID             string          `xml:"id,attr"`
	BaseURL        string          `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

// SegmentTemplate represents segment information using a template.
type SegmentTemplate struct {
	XMLName         xml.Name        `xml:"SegmentTemplate"`
	Initialization string          `xml:"initialization,attr"`
	Media          string          `xml:"media,attr"`
	StartNumber    string          `xml:"startNumber,attr"`
	EndNumber      string          `xml:"endNumber,attr"` // Added endNumber
	Timescale      string          `xml:"timescale,attr"`
	Duration       string          `xml:"duration,attr"`
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
	T       string   `xml:"t,attr"`
	D       string   `xml:"d,attr"`
	R       string   `xml:"r,attr"`
}

// SegmentURLs represents the output structure.
type SegmentURLs struct {
	RepresentationID string   `json:"representationId"`
	InitializationURL string  `json:"initializationUrl,omitempty"`
	SegmentURLs      []string `json:"segmentUrls"`
}

// parseDuration converts an XML duration string (e.g., "PT10S", "PT1H2M3.4S") to seconds.
func parseDuration(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	s = strings.TrimPrefix(s, "PT")
	var totalSeconds float64

	if strings.Contains(s, "H") {
		parts := strings.Split(s, "H")
		hours, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid hours format in duration: %w", err)
		}
		totalSeconds += hours * 3600
		s = parts[1]
	}

	if strings.Contains(s, "M") {
		parts := strings.Split(s, "M")
		minutes, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid minutes format in duration: %w", err)
		}
		totalSeconds += minutes * 60
		s = parts[1]
	}

	if strings.Contains(s, "S") {
		parts := strings.Split(s, "S")
		seconds, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid seconds format in duration: %w", err)
		}
		totalSeconds += seconds
	}
	return totalSeconds, nil
}


func main() {
	var err error
	logFile, err = os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to open log file '%s': %v\n", logFileName, err)
	} else {
		defer logFile.Close()
		multiWriter = io.MultiWriter(os.Stderr, logFile)
	}


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

	if mpd.BaseURL != "" {
		parsedMPDBase, err := url.Parse(mpd.BaseURL)
		if err != nil {
			logError("Invalid BaseURL in MPD: %s, using default. Error: %v", mpd.BaseURL, err)
		} else {
			mpdBaseURL = mpdBaseURL.ResolveReference(parsedMPDBase)
		}
	}
	logInfo("Effective MPD Base URL for resolution: %s", mpdBaseURL.String())

	mediaPresentationDurationSeconds := 0.0
	if mpd.MediaPresentationDuration != "" {
		parsedDuration, err := parseDuration(mpd.MediaPresentationDuration)
		if err != nil {
			logError("Failed to parse mediaPresentationDuration '%s': %v", mpd.MediaPresentationDuration, err)
		} else {
			mediaPresentationDurationSeconds = parsedDuration
			logInfo("Parsed MediaPresentationDuration: %f seconds", mediaPresentationDurationSeconds)
		}
	}

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

			effectiveASTemplate := as.SegmentTemplate

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

				segmentTemplate := rep.SegmentTemplate
				if segmentTemplate == nil {
					segmentTemplate = effectiveASTemplate
				}

				if segmentTemplate != nil {
					logInfo("Processing SegmentTemplate for Representation ID: %s", rep.ID)
					var initURL string
					var segments []string

					if segmentTemplate.Initialization != "" {
						initTemplate := segmentTemplate.Initialization
						initTemplate = strings.Replace(initTemplate, "$RepresentationID$", rep.ID, -1)

						initRelativeURL, err := url.Parse(initTemplate)
						if err != nil {
							logError("Failed to parse initialization URL string '%s' for %s: %v", initTemplate, rep.ID, err)
						} else {
							resolvedInitURL := representationBaseURL.ResolveReference(initRelativeURL).String()
							initURL = resolvedInitURL
							logInfo("Resolved Initialization URL for %s: %s", rep.ID, initURL)
						}
					}

					if segmentTemplate.SegmentTimeline != nil {
						logInfo("Generating Segment URLs using SegmentTimeline for %s", rep.ID)
						currentNumber := 0
						var currentTime int64 = 0

						for i, s := range segmentTemplate.SegmentTimeline.Segments {
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
								segmentURLTemplate := segmentTemplate.Media
								
								if strings.Contains(segmentURLTemplate, "$Number$") {
									startNum := 1
									if segmentTemplate.StartNumber != "" {
										parsedStartNum, err := strconv.Atoi(segmentTemplate.StartNumber)
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

								segmentURLTemplate = strings.Replace(segmentURLTemplate, "$RepresentationID$", rep.ID, -1)

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

					} else if strings.Contains(segmentTemplate.Media, "$Number$") {
						logInfo("Generating Segment URLs using $Number$ substitution (without SegmentTimeline) for %s", rep.ID)
						startNumber := 1
						if segmentTemplate.StartNumber != "" {
							parsedStartNumber, err := strconv.Atoi(segmentTemplate.StartNumber)
							if err != nil {
								logError("Failed to parse StartNumber for %s: %v, using 1", rep.ID, err)
							} else {
								startNumber = parsedStartNumber
							}
						}

						numSegmentsToGenerate := 0
						if segmentTemplate.EndNumber != "" {
							endNumber, err := strconv.Atoi(segmentTemplate.EndNumber)
							if err != nil {
								logError("Failed to parse EndNumber '%s' for %s: %v. Falling back to duration calculation.", segmentTemplate.EndNumber, rep.ID, err)
							} else {
								// Calculate based on start and end number
								numSegmentsToGenerate = endNumber - startNumber + 1
								logInfo("Calculated %d segments for %s using StartNumber (%d) and EndNumber (%d)", numSegmentsToGenerate, rep.ID, startNumber, endNumber)
							}
						}

						// Fallback to duration calculation if endNumber is not present or invalid
						if numSegmentsToGenerate <= 0 && mediaPresentationDurationSeconds > 0 && segmentTemplate.Timescale != "" && segmentTemplate.Duration != "" {
							timescale, err := strconv.ParseFloat(segmentTemplate.Timescale, 64)
							if err != nil {
								logError("Failed to parse Timescale '%s' for %s: %v. Cannot calculate total segments.", segmentTemplate.Timescale, rep.ID, err)
							} else {
								segmentDurationInTimescale, err := strconv.ParseFloat(segmentTemplate.Duration, 64)
								if err != nil {
									logError("Failed to parse SegmentTemplate duration '%s' for %s: %v. Cannot calculate total segments.", segmentTemplate.Duration, rep.ID, err)
								} else if timescale > 0 && segmentDurationInTimescale > 0 {
									segmentDurationSeconds := segmentDurationInTimescale / timescale
									numSegmentsToGenerate = int(math.Ceil(mediaPresentationDurationSeconds / segmentDurationSeconds))
									logInfo("Calculated %d segments for %s using mediaDuration (%f) and segmentDuration (%f)", numSegmentsToGenerate, rep.ID, mediaPresentationDurationSeconds, segmentDurationSeconds)
								}
							}
						}
						
						// Final fallback to a default reasonable number if all calculations fail
						if numSegmentsToGenerate <= 0 {
							logInfo("Dynamic segment calculation failed or resulted in 0. Defaulting to 100 segments for %s.", rep.ID)
							numSegmentsToGenerate = 100
						}


						for i := 0; i < numSegmentsToGenerate; i++ {
							segmentURLTemplate := strings.Replace(segmentTemplate.Media, "$Number$", strconv.Itoa(startNumber+i), -1)
							
							segmentURLTemplate = strings.Replace(segmentURLTemplate, "$RepresentationID$", rep.ID, -1)

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
						logInfo("Generating single Segment URL for %s (no $Number$ or SegmentTimeline)", rep.ID)
						segmentURLTemplate := segmentTemplate.Media
						segmentURLTemplate = strings.Replace(segmentURLTemplate, "$RepresentationID$", rep.ID, -1)

						segmentRelativeURL, err := url.Parse(segmentURLTemplate)
						if err != nil {
							logError("Failed to parse media URL string '%s' for %s: %v", segmentURLTemplate, rep.ID, err)
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
					initOrSegmentURL := representationBaseURL.String() 
					allSegmentURLs = append(allSegmentURLs, SegmentURLs{
						RepresentationID: rep.ID,
						SegmentURLs:      []string{initOrSegmentURL},
					})
					logInfo("Treated pre-resolved representationBaseURL as segment/initialization URL for %s: %s", rep.ID, initOrSegmentURL)
				} else {
					logInfo("Representation %s does not have SegmentTemplate (neither its own nor inherited) or BaseURL. Skipping segment URL extraction.", rep.ID)
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
