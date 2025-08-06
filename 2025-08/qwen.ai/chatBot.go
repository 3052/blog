package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

// MPD represents the root MPD element
type MPD struct {
	XMLName xml.Name `xml:"MPD"`
	Periods []Period `xml:"Period"`
}

// Period represents a Period element
type Period struct {
	XMLName        xml.Name         `xml:"Period"`
	BaseURL        string           `xml:"BaseURL"`
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet represents an AdaptationSet element
type AdaptationSet struct {
	XMLName         xml.Name         `xml:"AdaptationSet"`
	BaseURL         string           `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
	Representations []Representation `xml:"Representation"`
}

// Representation represents a Representation element
type Representation struct {
	XMLName   xml.Name `xml:"Representation"`
	ID        string   `xml:"id,attr"`
	BaseURL   string   `xml:"BaseURL"`
	SegmentBase *SegmentBase `xml:"SegmentBase"`
	SegmentList *SegmentList `xml:"SegmentList"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

// SegmentBase represents SegmentBase element
type SegmentBase struct {
	XMLName xml.Name `xml:"SegmentBase"`
	Initialization *URL `xml:"Initialization"`
}

// SegmentList represents SegmentList element
type SegmentList struct {
	XMLName xml.Name `xml:"SegmentList"`
	Initialization *URL `xml:"Initialization"`
	SegmentURLs    []SegmentURL `xml:"SegmentURL"`
	SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentURL represents SegmentURL element
type SegmentURL struct {
	XMLName xml.Name `xml:"SegmentURL"`
	Media   string   `xml:"media,attr"`
}

// SegmentTemplate represents SegmentTemplate element
type SegmentTemplate struct {
	XMLName      xml.Name `xml:"SegmentTemplate"`
	Initialization string   `xml:"initialization,attr"`
	Media        string   `xml:"media,attr"`
	StartNumber  string   `xml:"startNumber,attr"`
	EndNumber    string   `xml:"endNumber,attr"`
	Timescale    string   `xml:"timescale,attr"`
	Duration     string   `xml:"duration,attr"`
	SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline represents SegmentTimeline element
type SegmentTimeline struct {
	XMLName xml.Name `xml:"SegmentTimeline"`
	Segments []SegmentTimelineSegment `xml:"S"`
}

// SegmentTimelineSegment represents S elements in SegmentTimeline
type SegmentTimelineSegment struct {
	XMLName xml.Name `xml:"S"`
	T       string   `xml:"t,attr"` // presentation time
	D       string   `xml:"d,attr"` // duration
	R       string   `xml:"r,attr"` // repeat count
}

// URL represents elements with @sourceURL or @url attributes
type URL struct {
	SourceURL string `xml:"sourceURL,attr"`
	URL       string `xml:"url,attr"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <mpd_file_path>")
		os.Exit(1)
	}

	mpdFilePath := os.Args[1]
	
	// Read the MPD file
	data, err := os.ReadFile(mpdFilePath)
	if err != nil {
		fmt.Printf("Error reading MPD file: %v\n", err)
		os.Exit(1)
	}

	// Parse the MPD XML
	var mpd MPD
	err = xml.Unmarshal(data, &mpd)
	if err != nil {
		fmt.Printf("Error parsing MPD XML: %v\n", err)
		os.Exit(1)
	}

	// Base URL for resolving relative URLs
	baseMPDURL := "http://test.test/test.mpd"
	baseURL, err := url.Parse(baseMPDURL)
	if err != nil {
		fmt.Printf("Error parsing base URL: %v\n", err)
		os.Exit(1)
	}

	// Map to store representation ID to segment URLs
	result := make(map[string][]string)

	// Process each period
	for _, period := range mpd.Periods {
		periodBaseURL := resolveURL(baseURL, period.BaseURL)
		
		// Process each adaptation set
		for _, adaptationSet := range period.AdaptationSets {
			adaptationSetBaseURL := resolveURL(periodBaseURL, adaptationSet.BaseURL)
			
			// Process each representation
			for _, representation := range adaptationSet.Representations {
				representationBaseURL := resolveURL(adaptationSetBaseURL, representation.BaseURL)
				
				var segmentURLs []string
				
				// Handle SegmentList with SegmentTimeline
				if representation.SegmentList != nil {
					// Add initialization segment if it exists
					if representation.SegmentList.Initialization != nil {
						initURL := representation.SegmentList.Initialization.SourceURL
						if initURL == "" {
							initURL = representation.SegmentList.Initialization.URL
						}
						if initURL != "" {
							segmentURLs = append(segmentURLs, resolveURL(representationBaseURL, initURL).String())
						}
					}
					
					if representation.SegmentList.SegmentTimeline != nil {
						// Generate URLs based on SegmentTimeline
						timelineURLs := generateSegmentTimelineURLs(
							representation.SegmentList.SegmentTimeline,
							representationBaseURL,
							representation.SegmentList.SegmentURLs,
						)
						segmentURLs = append(segmentURLs, timelineURLs...)
					} else {
						// Handle regular SegmentList
						for _, segmentURL := range representation.SegmentList.SegmentURLs {
							if segmentURL.Media != "" {
								segmentURLs = append(segmentURLs, resolveURL(representationBaseURL, segmentURL.Media).String())
							}
						}
					}
				}
				
				// Handle SegmentTemplate inheritance
				var effectiveSegmentTemplate *SegmentTemplate
				
				// Check Representation level first
				if representation.SegmentTemplate != nil {
					effectiveSegmentTemplate = representation.SegmentTemplate
				} else if adaptationSet.SegmentTemplate != nil {
					// Fall back to AdaptationSet level
					effectiveSegmentTemplate = adaptationSet.SegmentTemplate
				}
				
				// Handle SegmentTemplate with SegmentTimeline
				if effectiveSegmentTemplate != nil {
					// Add initialization segment if it exists
					if effectiveSegmentTemplate.Initialization != "" {
						initURL := replaceTemplateVariables(effectiveSegmentTemplate.Initialization, 0, 0, representation.ID)
						segmentURLs = append(segmentURLs, resolveURL(representationBaseURL, initURL).String())
					}
					
					if effectiveSegmentTemplate.Media != "" {
						if effectiveSegmentTemplate.SegmentTimeline != nil {
							// Generate URLs based on SegmentTimeline in SegmentTemplate
							timelineURLs := generateSegmentTemplateTimelineURLs(
								effectiveSegmentTemplate.SegmentTimeline,
								effectiveSegmentTemplate.Media,
								representationBaseURL,
								effectiveSegmentTemplate.StartNumber,
								representation.ID,
							)
							segmentURLs = append(segmentURLs, timelineURLs...)
						} else if effectiveSegmentTemplate.Duration != "" {
							// Generate URLs based on duration with end number support
							durationURLs := generateSegmentTemplateDurationURLs(
								effectiveSegmentTemplate.Media,
								representationBaseURL,
								effectiveSegmentTemplate.StartNumber,
								effectiveSegmentTemplate.EndNumber,
								effectiveSegmentTemplate.Duration,
								representation.ID,
							)
							segmentURLs = append(segmentURLs, durationURLs...)
						} else {
							// Handle simple template patterns
							mediaPattern := effectiveSegmentTemplate.Media
							if !strings.Contains(mediaPattern, "$") {
								segmentURLs = append(segmentURLs, resolveURL(representationBaseURL, mediaPattern).String())
							} else {
								if strings.Contains(mediaPattern, "$Number") {
									// Generate URLs with start/end number support
									startNum := 1
									if effectiveSegmentTemplate.StartNumber != "" {
										if num, err := strconv.Atoi(effectiveSegmentTemplate.StartNumber); err == nil {
											startNum = num
										}
									}
									
									endNum := startNum + 4 // Default 5 segments
									if effectiveSegmentTemplate.EndNumber != "" {
										if num, err := strconv.Atoi(effectiveSegmentTemplate.EndNumber); err == nil {
											endNum = num
										}
									}
									
									for i := startNum; i <= endNum; i++ {
										urlStr := replaceTemplateVariables(mediaPattern, i, (i-startNum)*1000, representation.ID)
										segmentURLs = append(segmentURLs, resolveURL(representationBaseURL, urlStr).String())
									}
								} else if strings.Contains(mediaPattern, "$Time") {
									times := []int{0, 1000, 2000, 3000, 4000}
									for _, time := range times {
										urlStr := replaceTemplateVariables(mediaPattern, time/1000+1, time, representation.ID)
										segmentURLs = append(segmentURLs, resolveURL(representationBaseURL, urlStr).String())
									}
								} else {
									urlStr := replaceTemplateVariables(mediaPattern, 1, 0, representation.ID)
									segmentURLs = append(segmentURLs, resolveURL(representationBaseURL, urlStr).String())
								}
							}
						}
					}
				}
				
				// If no segments found, try to use the representation BaseURL as a segment
				if len(segmentURLs) == 0 && representation.BaseURL != "" {
					segmentURLs = append(segmentURLs, representationBaseURL.String())
				}
				
				result[representation.ID] = segmentURLs
			}
		}
	}

	// Output as JSON
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonData))
}

// replaceTemplateVariables replaces all template variables in a URL template
func replaceTemplateVariables(template string, number int, time int, representationID string) string {
	result := template
	
	// Replace $Number$ and $Number
	result = strings.ReplaceAll(result, "$Number$", fmt.Sprintf("%d", number))
	result = strings.ReplaceAll(result, "$Number", fmt.Sprintf("%d", number))
	
	// Replace $Time$ and $Time
	result = strings.ReplaceAll(result, "$Time$", fmt.Sprintf("%d", time))
	result = strings.ReplaceAll(result, "$Time", fmt.Sprintf("%d", time))
	
	// Replace $RepresentationID$ and $RepresentationID
	result = strings.ReplaceAll(result, "$RepresentationID$", representationID)
	result = strings.ReplaceAll(result, "$RepresentationID", representationID)
	
	return result
}

// generateSegmentTimelineURLs generates URLs based on SegmentTimeline in SegmentList
func generateSegmentTimelineURLs(timeline *SegmentTimeline, baseURL *url.URL, segmentURLs []SegmentURL) []string {
	var result []string
	
	// Use the actual segment URLs from SegmentList if available
	for _, segmentURL := range segmentURLs {
		if segmentURL.Media != "" {
			result = append(result, resolveURL(baseURL, segmentURL.Media).String())
		}
	}
	
	return result
}

// generateSegmentTemplateTimelineURLs generates URLs based on SegmentTimeline in SegmentTemplate
func generateSegmentTemplateTimelineURLs(timeline *SegmentTimeline, mediaTemplate string, baseURL *url.URL, startNumber string, representationID string) []string {
	var result []string
	
	startNum := 1
	if startNumber != "" {
		if num, err := strconv.Atoi(startNumber); err == nil {
			startNum = num
		}
	}
	
	segmentNumber := startNum
	timeValue := 0
	
	for _, segment := range timeline.Segments {
		// Handle presentation time if specified
		if segment.T != "" {
			if t, err := strconv.Atoi(segment.T); err == nil {
				timeValue = t
			}
		}
		
		// Get segment duration
		duration := 0
		if segment.D != "" {
			if d, err := strconv.Atoi(segment.D); err == nil {
				duration = d
			}
		}
		
		// Handle repeat count
		repeatCount := 0
		if segment.R != "" {
			if r, err := strconv.Atoi(segment.R); err == nil {
				repeatCount = r
			} else if segment.R == "-1" {
				// Special case: repeat until end of period (simplified)
				repeatCount = 0 // For now, just treat as no repeat
			}
		}
		
		// Generate URLs for this segment and its repeats
		for i := 0; i <= repeatCount; i++ {
			// Replace all template variables
			urlStr := replaceTemplateVariables(mediaTemplate, segmentNumber, timeValue, representationID)
			
			result = append(result, resolveURL(baseURL, urlStr).String())
			segmentNumber++
			
			// Increment time by segment duration for next iteration
			timeValue += duration
		}
	}
	
	return result
}

// generateSegmentTemplateDurationURLs generates URLs based on duration in SegmentTemplate with end number support
func generateSegmentTemplateDurationURLs(mediaTemplate string, baseURL *url.URL, startNumber string, endNumber string, duration string, representationID string) []string {
	var result []string
	
	startNum := 1
	if startNumber != "" {
		if num, err := strconv.Atoi(startNumber); err == nil {
			startNum = num
		}
	}
	
	endNum := startNum + 9 // Default 10 segments
	if endNumber != "" {
		if num, err := strconv.Atoi(endNumber); err == nil {
			endNum = num
		}
	}
	
	durationValue := 1000 // Default duration
	if duration != "" {
		if d, err := strconv.Atoi(duration); err == nil {
			durationValue = d
		}
	}
	
	// Generate segments from start to end number
	for i := startNum; i <= endNum; i++ {
		segmentNumber := i
		timeValue := (i - startNum) * durationValue // Time increases by duration each iteration
		
		// Replace all template variables
		urlStr := replaceTemplateVariables(mediaTemplate, segmentNumber, timeValue, representationID)
		
		result = append(result, resolveURL(baseURL, urlStr).String())
	}
	
	return result
}

// resolveURL resolves a relative URL against a base URL
func resolveURL(base *url.URL, relative string) *url.URL {
	if relative == "" {
		return base
	}
	
	// Parse the relative URL
	relURL, err := url.Parse(relative)
	if err != nil {
		// If parsing fails, treat as relative path
		result := *base
		result.Path = path.Join(path.Dir(base.Path), relative)
		return &result
	}
	
	// If relative URL has a scheme, it's absolute
	if relURL.Scheme != "" {
		return relURL
	}
	
	// Resolve against base URL
	return base.ResolveReference(relURL)
}
