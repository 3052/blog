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

const baseURL = "http://test.test/test.mpd"

type MPD struct {
	XMLName   xml.Name `xml:"MPD"`
	Periods   []Period `xml:"Period"`
	Duration  string   `xml:"mediaPresentationDuration,attr"`
}

type Period struct {
	Duration      string          `xml:"duration,attr"`
	BaseURL       *BaseURL        `xml:"BaseURL"`
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
	BaseURL         *BaseURL          `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate  `xml:"SegmentTemplate"`
	SegmentList     *SegmentList      `xml:"SegmentList"`
	Representations []Representation  `xml:"Representation"`
}

type Representation struct {
	ID              string           `xml:"id,attr"`
	BaseURL         *BaseURL         `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
	SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
	Initialization   string           `xml:"initialization,attr"`
	Media            string           `xml:"media,attr"`
	StartNumber      int              `xml:"startNumber,attr"`
	EndNumber        int              `xml:"endNumber,attr"`
	Timescale        int              `xml:"timescale,attr"`
	Duration         int              `xml:"duration,attr"`
	SegmentTimeline  *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
	S []S `xml:"S"`
}

type S struct {
	T int `xml:"t,attr"`
	D int `xml:"d,attr"`
	R int `xml:"r,attr"`
}

type SegmentList struct {
	Initialization *SegmentURL   `xml:"Initialization"`
	SegmentURLs    []SegmentURL `xml:"SegmentURL"`
}

type SegmentURL struct {
	Media     string `xml:"media,attr"`
	SourceURL string `xml:"sourceURL,attr"`
}

type BaseURL struct {
	Value string `xml:",chardata"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <mpd_file_path>")
		return
	}
	mpdPath := os.Args[1]
	data, err := ioutil.ReadFile(mpdPath)
	if err != nil {
		panic(err)
	}

	var mpd MPD
	if err := xml.Unmarshal(data, &mpd); err != nil {
		panic(err)
	}

	durationSec := parseDuration(mpd.Duration)
	result := make(map[string][]string)

	for _, period := range mpd.Periods {
		periodDuration := durationSec
		if period.Duration != "" {
			periodDuration = parseDuration(period.Duration)
		}

		periodBase := resolveURL(baseURL, getBase(period.BaseURL))

		for _, aset := range period.AdaptationSets {
			asetBase := resolveURL(periodBase, getBase(aset.BaseURL))

			for _, rep := range aset.Representations {
				repBase := resolveURL(asetBase, getBase(rep.BaseURL))

				var urls []string
				tmpl := firstTemplate(rep.SegmentTemplate, aset.SegmentTemplate)
				slist := firstList(rep.SegmentList, aset.SegmentList)

				if tmpl != nil {
					if tmpl.Initialization != "" {
						initURL := replacePlaceholders(tmpl.Initialization, rep.ID, 0)
						urls = append(urls, resolveURL(repBase, initURL))
					}
					if tmpl.SegmentTimeline != nil {
						var t0 int
						for _, s := range tmpl.SegmentTimeline.S {
							repeat := s.R
							if repeat < 0 {
								repeat = 0
							}
							count := repeat + 1
							startTime := s.T
							if startTime == 0 {
								startTime = t0
							}
							for i := 0; i < count; i++ {
								t := startTime + i*s.D
								media := replacePlaceholders(tmpl.Media, rep.ID, t)
								urls = append(urls, resolveURL(repBase, media))
							}
							t0 = startTime + count*s.D
						}
					} else if tmpl.EndNumber > 0 {
						for i := tmpl.StartNumber; i <= tmpl.EndNumber; i++ {
							media := replacePlaceholders(tmpl.Media, rep.ID, i)
							urls = append(urls, resolveURL(repBase, media))
						}
					} else if tmpl.Duration > 0 && tmpl.Timescale > 0 {
						count := int(math.Ceil(periodDuration * float64(tmpl.Timescale) / float64(tmpl.Duration)))
						for i := 0; i < count; i++ {
							num := tmpl.StartNumber + i
							media := replacePlaceholders(tmpl.Media, rep.ID, num)
							urls = append(urls, resolveURL(repBase, media))
						}
					}
				} else if slist != nil {
					if slist.Initialization != nil {
						src := slist.Initialization.SourceURL
						urls = append(urls, resolveURL(repBase, src))
					}
					for _, su := range slist.SegmentURLs {
						u := su.Media
						if u == "" {
							u = su.SourceURL
						}
						urls = append(urls, resolveURL(repBase, u))
					}
				} else if rep.BaseURL != nil {
					urls = append(urls, repBase)
				}

				result[rep.ID] = append(result[rep.ID], urls...)
			}
		}
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(out))
}

// --- Helpers ---

func getBase(b *BaseURL) string {
	if b != nil {
		return strings.TrimSpace(b.Value)
	}
	return ""
}

func resolveURL(baseStr, refStr string) string {
	base, err := url.Parse(baseStr)
	if err != nil {
		return baseStr
	}
	ref, err := url.Parse(refStr)
	if err != nil {
		return baseStr
	}
	return base.ResolveReference(ref).String()
}

func firstTemplate(a, b *SegmentTemplate) *SegmentTemplate {
	if a != nil {
		return a
	}
	return b
}

func firstList(a, b *SegmentList) *SegmentList {
	if a != nil {
		return a
	}
	return b
}

func parseDuration(d string) float64 {
	d = strings.TrimPrefix(d, "PT")
	var total float64

	parts := strings.Split(d, "H")
	if len(parts) > 1 {
		h, _ := strconv.ParseFloat(parts[0], 64)
		total += h * 3600
		d = parts[1]
	}
	parts = strings.Split(d, "M")
	if len(parts) > 1 {
		m, _ := strconv.ParseFloat(parts[0], 64)
		total += m * 60
		d = parts[1]
	}
	parts = strings.Split(d, "S")
	if len(parts) > 0 {
		s, _ := strconv.ParseFloat(parts[0], 64)
		total += s
	}
	return total
}

func replacePlaceholders(template, repID string, val int) string {
	out := template
	out = strings.ReplaceAll(out, "$RepresentationID$", repID)

	for {
		start := strings.Index(out, "$")
		if start == -1 {
			break
		}
		end := strings.Index(out[start+1:], "$")
		if end == -1 {
			break
		}
		end += start + 1
		full := out[start : end+1]

		if strings.HasPrefix(full, "$Number") || strings.HasPrefix(full, "$Time") {
			format := "%d"
			if i := strings.Index(full, "%"); i != -1 {
				format = full[i : len(full)-1]
			}
			out = strings.Replace(out, full, fmt.Sprintf(format, val), 1)
		} else {
			break
		}
	}
	return out
}
