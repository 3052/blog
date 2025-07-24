package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// ------------------------------------------------------------------
// XML essentials
// ------------------------------------------------------------------

type MPD struct {
	XMLName  xml.Name `xml:"MPD"`
	BaseURL  string   `xml:"BaseURL"`
	Duration string   `xml:"mediaPresentationDuration,attr"`
	Periods  []Period `xml:"Period"`
}

type Period struct {
	Duration string       `xml:"duration,attr"`
	BaseURL  string       `xml:"BaseURL"`
	Adapt    []Adaptation `xml:"AdaptationSet"`
}

type Adaptation struct {
	BaseURL string           `xml:"BaseURL"`
	Tmpl    *SegmentTemplate `xml:"SegmentTemplate"`
	Reps    []Representation `xml:"Representation"`
}

type Representation struct {
	ID   string           `xml:"id,attr"`
	BURL string           `xml:"BaseURL"`
	Tmpl *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentTemplate struct {
	Media       string           `xml:"media,attr"`
	Duration    int              `xml:"duration,attr"`
	Timescale   int              `xml:"timescale,attr"`
	StartNumber *int             `xml:"startNumber,attr"` // pointer distinguishes missing vs 0
	EndNumber   int              `xml:"endNumber,attr"`
	Timeline    *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
	S []S `xml:"S"`
}

type S struct {
	T int `xml:"t,attr"`
	D int `xml:"d,attr"`
	R int `xml:"r,attr"`
}

// ------------------------------------------------------------------
// main
// ------------------------------------------------------------------

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: mpd2urls <path-to-mpd>")
		os.Exit(1)
	}
	mpdPath := os.Args[1]

	data, err := os.ReadFile(mpdPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read file: %v\n", err)
		os.Exit(1)
	}

	var mpd MPD
	if err := xml.Unmarshal(data, &mpd); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal mpd: %v\n", err)
		os.Exit(1)
	}

	base, err := url.Parse("http://test.test/test.mpd")
	if err != nil {
		fmt.Fprintf(os.Stderr, "initial url: %v\n", err)
		os.Exit(1)
	}

	result := map[string][]string{}

	for _, p := range mpd.Periods {
		periodBase := resolveBase(resolveBase(base, mpd.BaseURL), p.BaseURL)
		periodDur := parseISODuration(p.Duration)
		if periodDur == 0 {
			periodDur = parseISODuration(mpd.Duration)
		}

		for _, as := range p.Adapt {
			adaptBase := resolveBase(periodBase, as.BaseURL)

			for _, rep := range as.Reps {
				repBase := resolveBase(adaptBase, rep.BURL)

				tmpl := as.Tmpl
				if rep.Tmpl != nil {
					tmpl = rep.Tmpl
				}
				if tmpl == nil {
					abs := repBase.String()
					result[rep.ID] = appendUnique(result[rep.ID], abs)
					continue
				}

				media := tmpl.Media
				timescale := tmpl.Timescale
				if timescale == 0 {
					timescale = 1
				}

				startNumber := 1
				if tmpl.StartNumber != nil {
					startNumber = *tmpl.StartNumber
				}

				// ---- SegmentTimeline mode --------------------------------------
				if tmpl.Timeline != nil && len(tmpl.Timeline.S) > 0 {
					time := 0
					num := startNumber
					for _, s := range tmpl.Timeline.S {
						if s.T != 0 {
							time = s.T
						}
						repeat := 1 + s.R
						for i := 0; i < repeat; i++ {
							urlStr := resolveBase(repBase,
								expandTemplate(media, rep.ID, num, time, timescale)).String()
							result[rep.ID] = appendUnique(result[rep.ID], urlStr)
							time += s.D
							num++
						}
					}
					continue
				}

				// ---- Simple / derived @endNumber -------------------------------
				end := tmpl.EndNumber
				if end == 0 {
					if tmpl.Duration == 0 || periodDur == 0 {
						end = startNumber + 1000 // fallback
					} else {
						segments := math.Ceil(periodDur.Seconds() * float64(timescale) / float64(tmpl.Duration))
						end = startNumber + int(segments) - 1
					}
				}
				for n := startNumber; n <= end; n++ {
					urlStr := resolveBase(repBase,
						expandTemplate(media, rep.ID, n, 0, timescale)).String()
					result[rep.ID] = appendUnique(result[rep.ID], urlStr)
				}
			}
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "encode json: %v\n", err)
		os.Exit(1)
	}
}

// ------------------------------------------------------------------
// helpers
// ------------------------------------------------------------------

func resolveBase(base *url.URL, ref string) *url.URL {
	if ref == "" {
		return base
	}
	r, _ := url.Parse(ref)
	return base.ResolveReference(r)
}

func expandTemplate(tpl, rid string, num, time, timescale int) string {
	tpl = replaceToken(tpl, "Number", num)
	tpl = replaceToken(tpl, "Time", time)
	tpl = strings.ReplaceAll(tpl, "$RepresentationID$", rid)
	return tpl
}

func replaceToken(tpl, token string, val int) string {
	start := 0
	prefix := "$" + token
	for {
		i := strings.Index(tpl[start:], prefix)
		if i == -1 {
			break
		}
		i += start
		j := i + len(prefix)

		width := 0
		if j < len(tpl) && tpl[j] == '%' {
			k := j + 1
			for k < len(tpl) && tpl[k] >= '0' && tpl[k] <= '9' {
				k++
			}
			if k < len(tpl) && tpl[k] == 'd' {
				width, _ = strconv.Atoi(tpl[j+1 : k])
				j = k + 1
			}
		}
		if j < len(tpl) && tpl[j] == '$' {
			j++
		} else {
			start = i + 1
			continue
		}

		repl := strconv.Itoa(val)
		if width > 0 {
			repl = fmt.Sprintf("%0*d", width, val)
		}
		tpl = tpl[:i] + repl + tpl[j:]
		start = i + len(repl)
	}
	return tpl
}

func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

func parseISODuration(s string) time.Duration {
	if s == "" {
		return 0
	}
	s = strings.TrimPrefix(s, "PT")
	s = strings.TrimSuffix(s, "S")
	secs, _ := strconv.ParseFloat(s, 64)
	return time.Duration(secs * float64(time.Second))
}
