package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

/* ---------- MPD structs ---------- */

type MPD struct {
	XMLName xml.Name `xml:"MPD"`
	BaseURL string   `xml:"BaseURL"`
	Period  Period   `xml:"Period"`
}

type Period struct {
	BaseURL        string          `xml:"BaseURL"`
	Duration       string          `xml:"duration,attr"`
	AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
	BaseURL         string             `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate   `xml:"SegmentTemplate"`
	SegmentList     *SegmentList       `xml:"SegmentList"`
	Representations []Representation   `xml:"Representation"`
}

type Representation struct {
	ID              string            `xml:"id,attr"`
	BaseURL         string            `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate  `xml:"SegmentTemplate"`
	SegmentList     *SegmentList      `xml:"SegmentList"`
}

type SegmentTemplate struct {
	Media       string `xml:"media,attr"`
	Timescale   int    `xml:"timescale,attr"`
	Duration    int    `xml:"duration,attr"`
	StartNumber int    `xml:"startNumber,attr"`
	EndNumber   int    `xml:"endNumber,attr"`
	Timeline    []S    `xml:"SegmentTimeline>S"`
}

type S struct {
	T int `xml:"t,attr"`
	D int `xml:"d,attr"`
	R int `xml:"r,attr"`
}

type SegmentList struct {
	SegmentURLs []SegmentURL `xml:"SegmentURL"`
}

type SegmentURL struct {
	Media string `xml:"media,attr"`
}

/* ---------- Helpers ---------- */

// resolve is a tiny wrapper around url.ResolveReference.
func resolve(base, rel string) string {
	if rel == "" {
		return base
	}
	b, _ := url.Parse(base)
	r, _ := url.Parse(rel)
	return b.ResolveReference(r).String()
}

func parseDuration(dur string) float64 {
	if !strings.HasPrefix(dur, "PT") {
		return 0
	}
	dur = strings.TrimPrefix(dur, "PT")
	var sec float64
	if strings.Contains(dur, "H") {
		parts := strings.Split(dur, "H")
		h, _ := strconv.ParseFloat(parts[0], 64)
		sec += h * 3600
		dur = parts[1]
	}
	if strings.Contains(dur, "M") {
		parts := strings.Split(dur, "M")
		m, _ := strconv.ParseFloat(parts[0], 64)
		sec += m * 60
		dur = parts[1]
	}
	if strings.Contains(dur, "S") {
		parts := strings.Split(dur, "S")
		s, _ := strconv.ParseFloat(parts[0], 64)
		sec += s
	}
	return sec
}

func expandTemplate(tpl, id string, n, t int) string {
	tpl = strings.ReplaceAll(tpl, "$RepresentationID$", id)
	tpl = strings.ReplaceAll(tpl, "$Number$", strconv.Itoa(n))
	tpl = strings.ReplaceAll(tpl, "$Time$", strconv.Itoa(t))

	// %0xd padding
	if strings.Contains(tpl, "$Number%") {
		start := strings.Index(tpl, "$Number%")
		end := strings.Index(tpl[start:], "$")
		if end != -1 {
			end += start
			format := tpl[start+8 : end]
			width, _ := strconv.Atoi(strings.TrimSuffix(format, "d"))
			padded := fmt.Sprintf("%0*d", width, n)
			tpl = strings.ReplaceAll(tpl, "$Number%"+format+"$", padded)
		}
	}
	return tpl
}

func expandSegmentList(base string, list *SegmentList) []string {
	var urls []string
	for _, su := range list.SegmentURLs {
		urls = append(urls, resolve(base, su.Media))
	}
	return urls
}

func dedup(in []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, s := range in {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

/* ---------- Main ---------- */

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <local.mpd>\n", os.Args[0])
		os.Exit(1)
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading file: %v\n", err)
		os.Exit(1)
	}

	var mpd MPD
	if err := xml.Unmarshal(data, &mpd); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing MPD: %v\n", err)
		os.Exit(1)
	}

	const originalURL = "http://test.test/test.mpd"

	rootBase := resolve(originalURL, mpd.BaseURL)
	periodBase := resolve(rootBase, mpd.Period.BaseURL)
	periodDur := parseDuration(mpd.Period.Duration)

	out := map[string][]string{}

	for _, as := range mpd.Period.AdaptationSets {
		adaptBase := resolve(periodBase, as.BaseURL)
		for _, rep := range as.Representations {
			repBase := resolve(adaptBase, rep.BaseURL)
			repID := rep.ID

			var segs []string

			switch {
			case rep.SegmentTemplate != nil:
				segs = expandTemplateBased(repBase, rep.SegmentTemplate, repID, periodDur, string(data))
			case as.SegmentTemplate != nil:
				segs = expandTemplateBased(repBase, as.SegmentTemplate, repID, periodDur, string(data))
			case rep.SegmentList != nil:
				segs = expandSegmentList(repBase, rep.SegmentList)
			case as.SegmentList != nil:
				segs = expandSegmentList(repBase, as.SegmentList)
			default:
				segs = []string{repBase}
			}

			out[repID] = dedup(segs)
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func expandTemplateBased(base string, tpl *SegmentTemplate, id string, periodDur float64, doc string) []string {
	media := tpl.Media
	if media == "" {
		media = "$RepresentationID$/$Number$.m4s"
	}
	timescale := tpl.Timescale
	if timescale == 0 {
		timescale = 1
	}
	startNumber := tpl.StartNumber
	if startNumber == 0 && !strings.Contains(doc, `startNumber="0"`) {
		startNumber = 1
	}

	var segs []string

	if len(tpl.Timeline) > 0 {
		time := 0
		for _, s := range tpl.Timeline {
			if s.T != 0 {
				time = s.T
			}
			repeats := 1 + s.R
			for r := 0; r < repeats; r++ {
				seg := resolve(base, expandTemplate(media, id, 0, time))
				segs = append(segs, seg)
				time += s.D
			}
		}
		return segs
	}

	endNumber := tpl.EndNumber
	if endNumber == 0 && tpl.Duration > 0 && periodDur > 0 {
		endNumber = int((periodDur * float64(timescale)) / float64(tpl.Duration))
		if (periodDur*float64(timescale))/float64(tpl.Duration) != float64(int((periodDur*float64(timescale))/float64(tpl.Duration))) {
			endNumber++
		}
		if endNumber < startNumber {
			endNumber = startNumber
		}
	}
	if endNumber == 0 {
		endNumber = startNumber
	}

	for n := startNumber; n <= endNumber; n++ {
		seg := resolve(base, expandTemplate(media, id, n, 0))
		segs = append(segs, seg)
	}
	return segs
}
