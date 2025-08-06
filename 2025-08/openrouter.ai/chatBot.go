package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"
)

const baseURL = "http://test.test/test.mpd"

type MPD struct {
	XMLName xml.Name `xml:"MPD"`
	Periods []Period `xml:"Period"`
}

type Period struct {
	XMLName        xml.Name        `xml:"Period"`
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
	XMLName         xml.Name         `xml:"AdaptationSet"`
	BaseURLs        []BaseURL        `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
	Representations []Representation `xml:"Representation"`
}

type Representation struct {
	XMLName         xml.Name         `xml:"Representation"`
	ID              string           `xml:"id,attr"`
	Bandwidth       string           `xml:"bandwidth,attr"`
	BaseURLs        []BaseURL        `xml:"BaseURL"`
	SegmentList     *SegmentList     `xml:"SegmentList"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type BaseURL struct {
	XMLName xml.Name `xml:"BaseURL"`
	Value   string   `xml:",chardata"`
}

type SegmentList struct {
	XMLName     xml.Name      `xml:"SegmentList"`
	SegmentURLs []SegmentURL  `xml:"SegmentURL"`
}

type SegmentURL struct {
	XMLName xml.Name `xml:"SegmentURL"`
	Media   string   `xml:"media,attr"`
}

type SegmentTemplate struct {
	XMLName        xml.Name `xml:"SegmentTemplate"`
	Media          string   `xml:"media,attr"`
	Initialization string   `xml:"initialization,attr"`
	Timescale      string   `xml:"timescale,attr"`
	Duration       string   `xml:"duration,attr"`
	StartNumber    string   `xml:"startNumber,attr"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <mpd_file_path>")
		os.Exit(1)
	}

	mpdPath := os.Args[1]
	data, err := os.ReadFile(mpdPath)
	if err != nil {
		fmt.Printf("Error reading MPD file: %v\n", err)
		os.Exit(1)
	}

	var mpd MPD
	if err := xml.Unmarshal(data, &mpd); err != nil {
		fmt.Printf("Error parsing MPD XML: %v\n", err)
		os.Exit(1)
	}

	result := make(map[string][]string)

	for _, period := range mpd.Periods {
		for _, adaptationSet := range period.AdaptationSets {
			adaptationSetBaseURLs := resolveBaseURLs(adaptationSet.BaseURLs, baseURL)
			if len(adaptationSetBaseURLs) == 0 {
				adaptationSetBaseURLs = []string{baseURL}
			}

			// Get SegmentTemplate from AdaptationSet if exists
			var defaultTemplate *SegmentTemplate
			if adaptationSet.SegmentTemplate != nil {
				defaultTemplate = adaptationSet.SegmentTemplate
			}

			for _, representation := range adaptationSet.Representations {
				if representation.ID == "" {
					continue
				}

				repBaseURLs := resolveBaseURLs(representation.BaseURLs, adaptationSetBaseURLs[0])
				if len(repBaseURLs) == 0 {
					repBaseURLs = adaptationSetBaseURLs
				}
				if len(repBaseURLs) == 0 {
					repBaseURLs = []string{baseURL}
				}

				var urls []string
				baseURLStr := repBaseURLs[0]

				// Check for SegmentList first (highest priority)
				if representation.SegmentList != nil {
					for _, segment := range representation.SegmentList.SegmentURLs {
						if segment.Media != "" {
							urls = append(urls, resolveURL(baseURLStr, segment.Media))
						}
					}
				} else {
					// Check for SegmentTemplate (Representation level first, then AdaptationSet level)
					var template *SegmentTemplate
					if representation.SegmentTemplate != nil {
						template = representation.SegmentTemplate
					} else if defaultTemplate != nil {
						template = defaultTemplate
					}

					if template != nil {
						if template.Initialization != "" {
							initURL := strings.Replace(template.Initialization, "$RepresentationID$", representation.ID, -1)
							initURL = strings.Replace(initURL, "$Bandwidth$", representation.Bandwidth, -1)
							urls = append(urls, resolveURL(baseURLStr, initURL))
						}

						if template.Media != "" {
							startNumber := 1
							if template.StartNumber != "" {
								fmt.Sscanf(template.StartNumber, "%d", &startNumber)
							}

							// Generate 5 sample segments
							for i := 0; i < 5; i++ {
								segNum := startNumber + i
								segURL := strings.Replace(template.Media, "$Number$", fmt.Sprintf("%d", segNum), -1)
								segURL = strings.Replace(segURL, "$RepresentationID$", representation.ID, -1)
								segURL = strings.Replace(segURL, "$Bandwidth$", representation.Bandwidth, -1)
								urls = append(urls, resolveURL(baseURLStr, segURL))
							}
						}
					}
				}

				result[representation.ID] = urls
			}
		}
	}

	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(output))
}

func resolveBaseURLs(baseURLs []BaseURL, parentURL string) []string {
	var result []string
	for _, base := range baseURLs {
		value := strings.TrimSpace(base.Value)
		if value != "" {
			result = append(result, resolveURL(parentURL, value))
		}
	}
	return result
}

func resolveURL(baseStr, ref string) string {
	if isAbsoluteURL(ref) {
		return ref
	}

	base, err := url.Parse(baseStr)
	if err != nil {
		return ref
	}

	refURL, err := url.Parse(ref)
	if err != nil {
		// Handle relative paths manually
		if strings.HasPrefix(ref, "/") {
			// Absolute path from domain root
			return base.Scheme + "://" + base.Host + ref
		}
		// Relative path
		basePath := path.Dir(base.Path)
		if !strings.HasSuffix(basePath, "/") {
			basePath += "/"
		}
		return base.Scheme + "://" + base.Host + basePath + ref
	}

	resolved := base.ResolveReference(refURL)
	return resolved.String()
}

func isAbsoluteURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
