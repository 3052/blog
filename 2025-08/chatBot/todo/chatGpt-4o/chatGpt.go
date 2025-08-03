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

const mpdBaseURL = "http://test.test/test.mpd"

type MPD struct {
	XMLName xml.Name `xml:"MPD"`
	BaseURL string   `xml:"BaseURL"`
	Periods []Period `xml:"Period"`
}

type Period struct {
	BaseURL        string           `xml:"BaseURL"`
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
	BaseURL         string           `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
	SegmentList     *SegmentList     `xml:"SegmentList"`
	Representations []Representation `xml:"Representation"`
}

type Representation struct {
	ID              string           `xml:"id,attr"`
	BaseURL         string           `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
	SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
	Initialization  string           `xml:"initialization,attr"`
	Media           string           `xml:"media,attr"`
	StartNumberStr  string           `xml:"startNumber,attr"`
	EndNumberStr    string           `xml:"endNumber,attr"`
	Timescale       int              `xml:"timescale,attr"`
	SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
	S []SegmentTime `xml:"S"`
}

type SegmentTime struct {
	T int64 `xml:"t,attr"`
	D int64 `xml:"d,attr"`
	R int   `xml:"r,attr"`
}

type SegmentList struct {
	Initialization *Initialization `xml:"Initialization"`
	SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
	SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
	Media string `xml:"media,attr"`
}

func resolveURL(base string, ref string) string {
	baseURL, err := url.Parse(base)
	if err != nil {
		return ref
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return baseURL.ResolveReference(refURL).String()
}

func expandTemplate(tmpl, repID string, number int, time int64) string {
	tmpl = strings.ReplaceAll(tmpl, "$RepresentationID$", repID)
	tmpl = strings.ReplaceAll(tmpl, "$Number$", fmt.Sprintf("%d", number))
	tmpl = strings.ReplaceAll(tmpl, "$Time$", fmt.Sprintf("%d", time))
	return tmpl
}

func parseAttrInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return n
}

func resolveEffectiveBaseURL(mpdURL, periodBase, asetBase, repBase string) string {
	base := mpdURL
	if periodBase != "" {
		base = resolveURL(base, periodBase)
	}
	if asetBase != "" {
		base = resolveURL(base, asetBase)
	}
	if repBase != "" {
		base = resolveURL(base, repBase)
	}
	return base
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <mpd_file_path>")
		os.Exit(1)
	}
	mpdPath := os.Args[1]
	data, err := ioutil.ReadFile(mpdPath)
	if err != nil {
		fmt.Println("Error reading MPD:", err)
		os.Exit(1)
	}

	var mpd MPD
	if err := xml.Unmarshal(data, &mpd); err != nil {
		fmt.Println("Error parsing MPD:", err)
		os.Exit(1)
	}

	output := make(map[string][]string)

	for _, period := range mpd.Periods {
		for _, aset := range period.AdaptationSets {
			for _, rep := range aset.Representations {
				id := rep.ID
				var urls []string

				base := resolveEffectiveBaseURL(mpdBaseURL, period.BaseURL, aset.BaseURL, rep.BaseURL)

				// SegmentList support
				sl := rep.SegmentList
				if sl == nil {
					sl = aset.SegmentList
				}
				if sl != nil {
					if sl.Initialization != nil && sl.Initialization.SourceURL != "" {
						urls = append(urls, resolveURL(base, sl.Initialization.SourceURL))
					}
					for _, su := range sl.SegmentURLs {
						urls = append(urls, resolveURL(base, su.Media))
					}
					output[id] = urls
					continue
				}

				// SegmentTemplate support
				st := rep.SegmentTemplate
				if st == nil {
					st = aset.SegmentTemplate
				}
				if st == nil || st.Media == "" {
					continue
				}

				if st.Initialization != "" {
					initURL := expandTemplate(st.Initialization, id, 0, 0)
					urls = append(urls, resolveURL(base, initURL))
				}

				if st.SegmentTimeline != nil {
					var currentTime int64
					for _, s := range st.SegmentTimeline.S {
						repeat := s.R
						if repeat < 0 {
							repeat = 0
						}
						startTime := s.T
						if startTime == 0 {
							startTime = currentTime
						}
						for i := 0; i <= repeat; i++ {
							segURL := expandTemplate(st.Media, id, 0, startTime)
							urls = append(urls, resolveURL(base, segURL))
							startTime += s.D
						}
						currentTime = startTime
					}
				} else {
					start := parseAttrInt(st.StartNumberStr, 1)
					end := parseAttrInt(st.EndNumberStr, -1)
					for num := start; end < 0 || num <= end; num++ {
						segURL := expandTemplate(st.Media, id, num, 0)
						urls = append(urls, resolveURL(base, segURL))
					}
				}

				output[id] = urls
			}
		}
	}

	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonData))
}
