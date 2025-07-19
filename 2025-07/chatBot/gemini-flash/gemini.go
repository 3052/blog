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
	"time"
)

// MPD structures (simplified for demonstration)
type MPD struct {
	XMLName              xml.Name `xml:"MPD"`
	BaseURL              string   `xml:"BaseURL"`
	Periods              []Period `xml:"Period"`
}

type Period struct {
	XMLName        xml.Name       `xml:"Period"`
	ID             string         `xml:"id,attr"`
	Duration       string         `xml:"duration,attr"` // Not strictly used for segment calculation, but useful for parsing
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
	XMLName        xml.Name         `xml:"AdaptationSet"`
	ID             string           `xml:"id,attr"`
	Representations []Representation `xml:"Representation"`
}

type Representation struct {
	XMLName        xml.Name       `xml:"Representation"`
	ID             string         `xml:"id,attr"`
	BaseURL        string         `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
	SegmentList     *SegmentList     `xml:"SegmentList"`
	SegmentBase     *SegmentBase     `xml:"SegmentBase"`
	Bandwidth      uint64         `xml:"bandwidth,attr"`
	Codecs         string         `xml:"codecs,attr"`
	MimeType       string         `xml:"mimeType,attr"`
}

type SegmentTemplate struct {
	XMLName      xml.Name     `xml:"SegmentTemplate"`
	Timescale    uint64       `xml:"timescale,attr"`
	Initialization string       `xml:"initialization,attr"`
	Media        string       `xml:"media,attr"`
	StartNumber  uint64       `xml:"startNumber,attr"`
	Duration     uint64       `xml:"duration,attr"` // For fixed duration segments
	SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
	XMLName xml.Name `xml:"SegmentTimeline"`
	Segments []S      `xml:"S"`
}

type S struct {
	XMLName xml.Name `xml:"S"`
	T       uint64   `xml:"t,attr"` // Start time
	D       uint64   `xml:"d,attr"` // Duration
	R       int      `xml:"r,attr"` // Repeat count
}

// These are for completeness but not fully implemented in segment URL generation as per prompt's focus on SegmentTemplate/BaseURL
type SegmentList struct {
	XMLName xml.Name `xml:"SegmentList"`
	Segments []Segment `xml:"Segment"`
}

type SegmentBase struct {
	XMLName xml.Name `xml:"SegmentBase"`
}

type Segment struct {
	XMLName xml.Name `xml:"SegmentURL"`
	Media   string   `xml:"media,attr"`
}

// SegmentURLInfo stores segment URLs for a representation
type SegmentURLInfo struct {
	Initialization string   `json:"initialization,omitempty"`
	Segments       []string `json:"segments"`
}

// Global logger
var logger *os.File

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <path_to_mpd_file>")
		return
	}

	mpdFilePath := os.Args[1]
	mpdBaseURL := "http://test.test/test.mpd" // Fixed MPD URL for relative URL resolution

	// Setup logging
	logFile, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		return
	}
	defer logFile.Close()
	logger = logFile

	logMessage(fmt.Sprintf("Starting MPD parsing for: %s", mpdFilePath))
	logMessage(fmt.Sprintf("Assuming MPD Base URL for resolution: %s", mpdBaseURL))

	data, err := ioutil.ReadFile(mpdFilePath)
	if err != nil {
		logMessage(fmt.Sprintf("Error reading MPD file: %v", err))
		fmt.Printf("Error reading MPD file: %v\n", err)
		return
	}

	var mpd MPD
	err = xml.Unmarshal(data, &mpd)
	if err != nil {
		logMessage(fmt.Sprintf("Error unmarshalling MPD XML: %v", err))
		fmt.Printf("Error unmarshalling MPD XML: %v\n", err)
		return
	}

	// Map to store segment URLs grouped by Representation ID
	representationSegments := make(map[string]SegmentURLInfo)

	// Base URL derived from the MPD itself, if present, overrides the provided testBaseURL
	// This is for proper resolution hierarchy based on DASH spec (MPD.BaseURL > Period.BaseURL > AdaptationSet.BaseURL > Representation.BaseURL)
	currentBaseURL := mpdBaseURL
	if mpd.BaseURL != "" {
		resolved, err := resolveURL(mpdBaseURL, mpd.BaseURL)
		if err != nil {
			logMessage(fmt.Sprintf("Warning: Could not resolve MPD BaseURL %s against %s: %v", mpd.BaseURL, mpdBaseURL, err))
		} else {
			currentBaseURL = resolved
			logMessage(fmt.Sprintf("MPD-level BaseURL found: %s, new currentBaseURL: %s", mpd.BaseURL, currentBaseURL))
		}
	}


	// Iterate through Periods, AdaptationSets, and Representations
	for _, period := range mpd.Periods {
		logMessage(fmt.Sprintf("Processing Period ID: %s", period.ID))

		// Period-level BaseURL
		periodBaseURL := currentBaseURL // Inherit from higher level
		// In a full implementation, check if Period has its own BaseURL

		for _, as := range period.AdaptationSets {
			logMessage(fmt.Sprintf("  Processing AdaptationSet ID: %s", as.ID))

			// AdaptationSet-level BaseURL
			asBaseURL := periodBaseURL // Inherit from higher level
			// In a full implementation, check if AdaptationSet has its own BaseURL

			for _, rep := range as.Representations {
				logMessage(fmt.Sprintf("    Processing Representation ID: %s, MimeType: %s, Codecs: %s", rep.ID, rep.MimeType, rep.Codecs))

				repBaseURL := asBaseURL // Inherit from higher level
				if rep.BaseURL != "" {
					resolved, err := resolveURL(asBaseURL, rep.BaseURL)
					if err != nil {
						logMessage(fmt.Sprintf("Warning: Could not resolve Representation BaseURL %s against %s: %v", rep.BaseURL, asBaseURL, err))
					} else {
						repBaseURL = resolved
						logMessage(fmt.Sprintf("      Representation-level BaseURL found: %s, new repBaseURL: %s", rep.BaseURL, repBaseURL))
					}
				}

				var initURL string
				var segmentURLs []string

				if rep.SegmentTemplate != nil {
					st := rep.SegmentTemplate
					timescale := st.Timescale
					if timescale == 0 {
						timescale = 1 // Default to 1 if missing
						logMessage(fmt.Sprintf("      SegmentTemplate timescale missing for Rep ID %s, defaulting to 1", rep.ID))
					}

					// Resolve Initialization URL
					if st.Initialization != "" {
						resolvedInit, err := resolveURL(repBaseURL, st.Initialization)
						if err != nil {
							logMessage(fmt.Sprintf("      Error resolving initialization URL %s: %v", st.Initialization, err))
							initURL = st.Initialization // Keep original if resolution fails
						} else {
							initURL = resolvedInit
							logMessage(fmt.Sprintf("      Resolved Initialization URL: %s", initURL))
						}
					}

					if st.SegmentTimeline != nil {
						// Generate segments based on SegmentTimeline
						timeCounter := uint64(0)
						segmentNumber := st.StartNumber
						if segmentNumber == 0 {
							segmentNumber = 1 // Default startNumber to 1 if not specified
						}
						logMessage(fmt.Sprintf("      Generating segments using SegmentTimeline. Start Number: %d", segmentNumber))

						for _, s := range st.SegmentTimeline.Segments {
							if s.T > 0 { // If 't' attribute is present, it explicitly sets the start time
								timeCounter = s.T
							}

							numSegments := s.R + 1
							for i := 0; i < numSegments; i++ {
								segmentURL, err := replaceTemplatePlaceholders(st.Media, rep.ID, segmentNumber, timeCounter, timescale)
								if err != nil {
									logMessage(fmt.Sprintf("      Error generating segment URL for %s: %v", st.Media, err))
									continue
								}

								resolvedSegmentURL, err := resolveURL(repBaseURL, segmentURL)
								if err != nil {
									logMessage(fmt.Sprintf("      Error resolving segment URL %s: %v", segmentURL, err))
									continue
								}
								segmentURLs = append(segmentURLs, resolvedSegmentURL)
								logMessage(fmt.Sprintf("        Generated Segment URL: %s", resolvedSegmentURL))

								timeCounter += s.D
								segmentNumber++
							}
						}
					} else if st.Duration > 0 && st.Media != "" {
						// Fixed duration segments (common for static MPDs without timeline)
						logMessage(fmt.Sprintf("      Generating fixed duration segments. Duration: %d, Start Number: %d", st.Duration, st.StartNumber))
						// For this simplified example, let's assume a fixed number of segments if duration is present without timeline.
						// In a real scenario, you'd calculate based on mediaPresentationDuration.
						// For now, let's just generate a few example segments.
						segmentCount := uint64(10) // Arbitrary count for demonstration if no timeline
						if st.StartNumber == 0 {
							st.StartNumber = 1
						}
						for i := uint64(0); i < segmentCount; i++ {
							segmentNumber := st.StartNumber + i
							timeValue := i * st.Duration // Time value for $Time$ placeholder if needed
							segmentURL, err := replaceTemplatePlaceholders(st.Media, rep.ID, segmentNumber, timeValue, timescale)
							if err != nil {
								logMessage(fmt.Sprintf("      Error generating segment URL for %s: %v", st.Media, err))
								continue
							}
							resolvedSegmentURL, err := resolveURL(repBaseURL, segmentURL)
							if err != nil {
								logMessage(fmt.Sprintf("      Error resolving segment URL %s: %v", segmentURL, err))
								continue
							}
							segmentURLs = append(segmentURLs, resolvedSegmentURL)
							logMessage(fmt.Sprintf("        Generated Segment URL: %s", resolvedSegmentURL))
						}
					}
				} else if rep.BaseURL != "" && rep.SegmentTemplate == nil && rep.SegmentList == nil && rep.SegmentBase == nil {
					// Representation contains only BaseURL, treat as the only segment
					resolved, err := resolveURL(asBaseURL, rep.BaseURL) // Resolve against AdaptationSet's base, then MPD base
					if err != nil {
						logMessage(fmt.Sprintf("      Error resolving single segment BaseURL %s: %v", rep.BaseURL, err))
					} else {
						segmentURLs = append(segmentURLs, resolved)
						logMessage(fmt.Sprintf("      Treating BaseURL as single segment: %s", resolved))
					}
				}
				// else {
				// 	logMessage(fmt.Sprintf("      Representation %s has no SegmentTemplate, SegmentList, SegmentBase, or BaseURL for segment generation.", rep.ID))
				// }

				// Consolidate segments by Representation ID
				if _, ok := representationSegments[rep.ID]; !ok {
					representationSegments[rep.ID] = SegmentURLInfo{
						Initialization: initURL,
						Segments:       []string{},
					}
				}
				info := representationSegments[rep.ID]
				info.Segments = append(info.Segments, segmentURLs...)
				// If a new initURL is found for the same rep.ID (across periods), prefer the latest one or handle conflict as needed.
				// For this problem, we'll just keep the first one encountered if already set.
				if info.Initialization == "" && initURL != "" {
					info.Initialization = initURL
				}
				representationSegments[rep.ID] = info
			}
		}
	}

	// Output as JSON
	jsonOutput, err := json.MarshalIndent(representationSegments, "", "  ")
	if err != nil {
		logMessage(fmt.Sprintf("Error marshalling JSON output: %v", err))
		fmt.Printf("Error marshalling JSON output: %v\n", err)
		return
	}

	fmt.Println(string(jsonOutput))
	logMessage("MPD parsing completed successfully.")
}

// logMessage writes a message to the log file with a timestamp
func logMessage(message string) {
	if logger != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		_, err := logger.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, message))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to log file: %v\n", err)
		}
	}
}

// resolveURL resolves a relative URL against a base URL
func resolveURL(baseURLStr, relativeURLStr string) (string, error) {
	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	relativeURL, err := url.Parse(relativeURLStr)
	if err != nil {
		return "", fmt.Errorf("invalid relative URL: %w", err)
	}

	return baseURL.ResolveReference(relativeURL).String(), nil
}

// replaceTemplatePlaceholders replaces $Number$, $Time$, and $RepresentationID$ in a template string
func replaceTemplatePlaceholders(template string, representationID string, number uint64, timeValue uint64, timescale uint64) (string, error) {
	res := strings.ReplaceAll(template, "$RepresentationID$", representationID)
	res = strings.ReplaceAll(res, "$RepresentationID%d$", representationID) // Handle %d format as well

	// Handle $Number$
	if strings.Contains(res, "$Number$") {
		res = strings.ReplaceAll(res, "$Number$", strconv.FormatUint(number, 10))
	} else if strings.Contains(res, "$Number%0") { // Handle $Number%0xd$ format
		// Using a simple string replace for now; a regex library like `regexp` would be more robust.
		// For simplicity and to avoid external imports for this example:
		parts := strings.Split(res, "$Number%0")
		if len(parts) > 1 {
			postNumberPart := parts[1]
			if len(postNumberPart) >= 2 && strings.HasPrefix(postNumberPart, "d$") { // e.g., "d$.m4s"
				widthStr := ""
				for _, char := range postNumberPart {
					if char >= '0' && char <= '9' {
						widthStr += string(char)
					} else {
						break
					}
				}
				if width, err := strconv.Atoi(widthStr); err == nil {
					format := fmt.Sprintf("%%0%dd", width)
					formattedNumber := fmt.Sprintf(format, number)
					res = strings.ReplaceAll(res, fmt.Sprintf("$Number%%0%dd$", width), formattedNumber)
				}
			}
		}
	}

	// Handle $Time$
	if strings.Contains(res, "$Time$") {
		res = strings.ReplaceAll(res, "$Time$", strconv.FormatUint(timeValue, 10))
	} else if strings.Contains(res, "$Time%0") { // Handle $Time%0xd$ format similarly
		parts := strings.Split(res, "$Time%0")
		if len(parts) > 1 {
			postTimePart := parts[1]
			if len(postTimePart) >= 2 && strings.HasPrefix(postTimePart, "d$") {
				widthStr := ""
				for _, char := range postTimePart {
					if char >= '0' && char <= '9' {
						widthStr += string(char)
					} else {
						break
					}
				}
				if width, err := strconv.Atoi(widthStr); err == nil {
					format := fmt.Sprintf("%%0%dd", width)
					formattedTime := fmt.Sprintf(format, timeValue)
					res = strings.ReplaceAll(res, fmt.Sprintf("$Time%%0%dd$", width), formattedTime)
				}
			}
		}
	}

	return res, nil
}
