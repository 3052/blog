package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ---------- XML structures ----------

type MPD struct {
	XMLName  xml.Name  `xml:"MPD"`
	BaseURL  string    `xml:"BaseURL"`
	Periods  []Period  `xml:"Period"`
	Xmlns    string    `xml:"xmlns,attr"`
	Type     string    `xml:"type,attr"`
	MediaPresentationDuration string `xml:"mediaPresentationDuration,attr"`
}

type Period struct {
	BaseURL string        `xml:"BaseURL"`
	AS      []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
	BaseURL       string          `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
	Representations []Representation `xml:"Representation"`
}

type Representation struct {
	ID            string          `xml:"id,attr"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentTemplate struct {
	BaseURL     string `xml:"BaseURL"`
	Media       string `xml:"media,attr"`
	Initialization string `xml:"initialization,attr"`
	StartNumber int    `xml:"startNumber,attr"`
	Timescale   int    `xml:"timescale,attr"`
	Duration    int    `xml:"duration,attr"` // $Number$ mode
	SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
	EndNumber   int    `xml:"endNumber,attr"`
}

type SegmentTimeline struct {
	S []S `xml:"S"`
}

type S struct {
	T int `xml:"t,attr"` // start time
	D int `xml:"d,attr"` // duration
	R int `xml:"r,attr"` // repeat count (0 = once)
}

// ---------- helpers ----------

func mustParse(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

// resolve base URLs according to DASH spec
func resolveBase(parent, child string) string {
	if child == "" {
		return parent
	}
	pu := mustParse(parent)
	cu := mustParse(child)
	return pu.ResolveReference(cu).String()
}

func buildSegmentURLs(
	mpdURL string,
	periodBase string,
	asBase string,
	repBase string,
	st *SegmentTemplate,
	repID string,
) []string {

	base := mpdURL
	base = resolveBase(base, periodBase)
	base = resolveBase(base, asBase)
	base = resolveBase(base, repBase)
	base = resolveBase(base, st.BaseURL)

	mediaTpl := strings.ReplaceAll(st.Media, "$RepresentationID$", repID)

	var segs []string

	if st.SegmentTimeline != nil {
		startNumber := st.StartNumber
		startTime := 0
		for _, s := range st.SegmentTimeline.S {
			t := s.T
			if t != 0 {
				startTime = t
			}
			repeat := s.R
			if repeat == 0 {
				repeat = 1
			} else {
				repeat++
			}
			for i := 0; i < repeat; i++ {
				name := strings.ReplaceAll(mediaTpl, "$Time$", strconv.Itoa(startTime))
				name = strings.ReplaceAll(name, "$Number$", strconv.Itoa(startNumber))
				segs = append(segs, resolveBase(base, name))
				startTime += s.D
				startNumber++
				if st.EndNumber > 0 && startNumber > st.EndNumber {
					break
				}
			}
		}
	} else {
		// $Number$ mode
		num := st.StartNumber
		for {
			name := strings.ReplaceAll(mediaTpl, "$Number$", strconv.Itoa(num))
			segs = append(segs, resolveBase(base, name))
			num++
			if st.EndNumber > 0 && num > st.EndNumber {
				break
			}
			// crude stop when we exceed duration – real apps parse duration attr
			if st.EndNumber == 0 && len(segs) > 10000 {
				break
			}
		}
	}
	return segs
}

// ---------- main ----------

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s local.mpd\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	mpdPath := os.Args[1]

	data, err := os.ReadFile(mpdPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}

	var mpd MPD
	if err := xml.Unmarshal(data, &mpd); err != nil {
		fmt.Fprintln(os.Stderr, "parse:", err)
		os.Exit(1)
	}

	// declare baseUrl as requested
	baseUrl := "http://test.test/test.mpd"

	out := make(map[string][]string)

	for _, p := range mpd.Periods {
		periodBase := p.BaseURL
		for _, as := range p.AS {
			asBase := as.BaseURL
			asTemplate := as.SegmentTemplate
			for _, rep := range as.Representations {
				repBase := ""          // Representation/BaseURL not modelled
				var st *SegmentTemplate
				if rep.SegmentTemplate != nil {
					st = rep.SegmentTemplate
				} else {
					st = asTemplate
				}
				if st == nil || st.Media == "" {
					continue
				}
				urls := buildSegmentURLs(baseUrl, periodBase, asBase, repBase, st, rep.ID)
				out[rep.ID] = urls
			}
		}
	}

	j, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(j))
}
