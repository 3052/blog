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

// ---------- XML structs ----------

type MPD struct {
	XMLName  xml.Name `xml:"MPD"`
	BaseURL  string   `xml:"BaseURL"`
	Period   Period   `xml:"Period"`
	Xmlns    string   `xml:"xmlns,attr"`
	TypeAttr string   `xml:"type,attr"` // only used to skip "dynamic" if you wish
}

type Period struct {
	BaseURL string `xml:"BaseURL"`
	Adapt   []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
	BaseURL        string           `xml:"BaseURL"`
	SegmentTmpl    *SegmentTemplate `xml:"SegmentTemplate"`
	Representations []Representation `xml:"Representation"`
}

type Representation struct {
	ID       string           `xml:"id,attr"`
	BaseURL  string           `xml:"BaseURL"`
	SegmentT *SegmentTemplate `xml:"SegmentTemplate"` // may override parent AdaptationSet
}

type SegmentTemplate struct {
	Media      string `xml:"media,attr"`
	Init       string `xml:"initialization,attr"` // ignored here
	Timescale  int    `xml:"timescale,attr"`      // ignored – we use @d directly
	StartNum   int    `xml:"startNumber,attr"`
	EndNum     *int   `xml:"endNumber,attr"` // pointer => nil if absent
	Timeline   []S    `xml:"SegmentTimeline>S"`
}

type S struct {
	D int `xml:"d,attr"`
	R int `xml:"r,attr"` // default 0
}

// ---------- helpers ----------

// resolveBase resolves relative URLs (§5 of the spec)
func resolveBase(parent, child string) string {
	if child == "" {
		return parent
	}
	u, err := url.Parse(child)
	if err != nil {
		return child
	}
	base, err := url.Parse(parent)
	if err != nil {
		return child
	}
	return base.ResolveReference(u).String()
}

// ---------- main ----------

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s local.mpd\n", os.Args[0])
		os.Exit(1)
	}

	// 1. read local file
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}

	// 2. unmarshal
	var mpd MPD
	if err := xml.Unmarshal(data, &mpd); err != nil {
		fmt.Fprintln(os.Stderr, "xml:", err)
		os.Exit(1)
	}

	// 3. resolve MPD@BaseURL
	//    The fixed baseUrl variable from requirement #4.
	fixedBase := "http://test.test/test.mpd"
	mpdBaseURL := resolveBase(fixedBase, mpd.BaseURL)

	// 4. resolve Period@BaseURL
	periodBase := resolveBase(mpdBaseURL, mpd.Period.BaseURL)

	// 5. build map
	out := make(map[string][]string)

	for _, as := range mpd.Period.Adapt {
		asBase := resolveBase(periodBase, as.BaseURL)

		for _, rep := range as.Representations {
			repBase := resolveBase(asBase, rep.BaseURL)

			// Representation@BaseURL exists => treat it as single-segment
			if rep.BaseURL != "" {
				out[rep.ID] = []string{repBase}
				continue
			}

			// 6. pick SegmentTemplate (Representation overrides AdaptationSet)
			st := rep.SegmentT
			if st == nil {
				st = as.SegmentTmpl
			}
			if st == nil {
				continue // no template
			}

			tmplBase := resolveBase(asBase, "")

			// 7. build URLs
			startNum := st.StartNum
			if startNum == 0 {
				startNum = 1 // requirement #13
			}

			// Requirement #8: SegmentTimeline handling
			var urls []string
			if len(st.Timeline) > 0 {
				startTime := 0
				num := startNum
				for _, s := range st.Timeline {
					count := 1 + s.R // requirement #14
					for i := 0; i < count; i++ {
						media := st.Media
						media = strings.ReplaceAll(media, "$RepresentationID$", rep.ID)
						media = strings.ReplaceAll(media, "$Time$", strconv.Itoa(startTime))
						media = strings.ReplaceAll(media, "$Number$", strconv.Itoa(num))

						segURL := resolveBase(tmplBase, media)
						urls = append(urls, segURL)

						startTime += s.D
						num++
					}
				}
			} else if st.EndNum != nil {
				// Use @endNumber if supplied
				for n := startNum; n <= *st.EndNum; n++ {
					media := st.Media
					media = strings.ReplaceAll(media, "$RepresentationID$", rep.ID)
					media = strings.ReplaceAll(media, "$Number$", strconv.Itoa(n))
					// $Time$ is not used when no SegmentTimeline
					segURL := resolveBase(tmplBase, media)
					urls = append(urls, segURL)
				}
			} else {
				// No timeline and no endNumber => infinite list impossible.
				// Skip silently (or error if you prefer).
			}

			out[rep.ID] = urls
		}
	}

	// 8. json.Marshal and print
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
}
