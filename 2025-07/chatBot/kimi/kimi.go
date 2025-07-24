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

// ---------- MPD structs ------------------------------------------------------

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL string          `xml:"BaseURL"`
   Sets    []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Reps            []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string            `xml:"id,attr"`
   BaseURL         string            `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate  `xml:"SegmentTemplate"`
}

type SegmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   Timescale       int              `xml:"timescale,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   EndNumber       *int             `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T int `xml:"t,attr"`
   D int `xml:"d,attr"`
   R int `xml:"r,attr"` // default 0
}

// ---------- helpers ----------------------------------------------------------

func joinPath(base, p string) string {
   u, _ := url.Parse(base)
   r, _ := url.Parse(p)
   return u.ResolveReference(r).String()
}

func fillTemplate(tpl string, repID string, n int, t int64) string {
   s := tpl
   s = strings.ReplaceAll(s, "$RepresentationID$", repID)
   s = strings.ReplaceAll(s, "$Number$", strconv.Itoa(n))
   s = strings.ReplaceAll(s, "$Time$", strconv.FormatInt(t, 10))

   // %0xd padding
   for {
      start := strings.Index(s, "%0")
      if start == -1 {
         break
      }
      end := strings.Index(s[start:], "d")
      if end == -1 {
         break
      }
      width, _ := strconv.Atoi(s[start+2 : start+end])
      s = strings.Replace(s, s[start:start+end+1], fmt.Sprintf("%0*d", width, n), 1)
   }
   return s
}

// effectiveTemplate returns the lowest-level SegmentTemplate (Representation wins, else AdaptationSet)
func effectiveTemplate(as AdaptationSet, rep Representation) *SegmentTemplate {
   if rep.SegmentTemplate != nil {
      return rep.SegmentTemplate
   }
   return as.SegmentTemplate
}

// ---------- main -------------------------------------------------------------

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run main.go local.mpd")
		os.Exit(1)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var mpd MPD
	if err := xml.NewDecoder(f).Decode(&mpd); err != nil {
		panic(err)
	}

	original, _ := url.Parse("http://test.test/test.mpd")
	out := make(map[string][]string)

	for _, p := range mpd.Periods {
		periodBase := original.String()
		if p.BaseURL != "" {
			periodBase = joinPath(periodBase, p.BaseURL)
		}

		for _, as := range p.Sets {
			asBase := periodBase
			if as.BaseURL != "" {
				asBase = joinPath(asBase, as.BaseURL)
			}

			for _, rep := range as.Reps {
				repBase := asBase
				if rep.BaseURL != "" {
					repBase = joinPath(repBase, rep.BaseURL)
				}

				st := effectiveTemplate(as, rep)

				// CASE 1: Representation has no SegmentTemplate
				if st == nil {
					// Just emit the single BaseURL
					out[rep.ID] = []string{repBase}
					continue
				}

				// CASE 2: Expand SegmentTemplate
				startN := 1
				if st.StartNumber != nil {
					startN = *st.StartNumber
				}

				var segments []string

				if st.SegmentTimeline != nil {
					segNum := startN
					time := int64(0)
					for _, s := range st.SegmentTimeline.S {
						t := int64(s.T)
						if t != 0 {
							time = t
						}
						for i := 0; i <= s.R; i++ {
							media := fillTemplate(st.Media, rep.ID, segNum, time)
							segments = append(segments, joinPath(repBase, media))
							time += int64(s.D)
							segNum++
						}
					}
				} else {
					endN := startN
					if st.EndNumber != nil {
						endN = *st.EndNumber
					}
					for n := startN; n <= endN; n++ {
						media := fillTemplate(st.Media, rep.ID, n, 0)
						segments = append(segments, joinPath(repBase, media))
					}
				}

				out[rep.ID] = segments
			}
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		panic(err)
	}
}
