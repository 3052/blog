package main

import (
   "encoding/json"
   "encoding/xml"
   "errors"
   "fmt"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
   "time"
)

const fallbackBase = "http://test.test/test.mpd"

type MPD struct {
   BaseURL string   `xml:"BaseURL"`
   Period  []Period `xml:"Period"`
}
type Period struct {
   ID        string       `xml:"id,attr"`
   Duration  string       `xml:"duration,attr"`
   BaseURL   string       `xml:"BaseURL"`
   AdaptSets []Adaptation `xml:"AdaptationSet"`
}
type Adaptation struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Rep             []Representation `xml:"Representation"`
}
type Representation struct {
   ID              string           `xml:"id,attr"`
   Bandwidth       int64            `xml:"bandwidth,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}
type SegmentTemplate struct {
   Initialization string           `xml:"initialization,attr"`
   Media          string           `xml:"media,attr"`
   Timescale      *int64           `xml:"timescale,attr"`
   Duration       *int64           `xml:"duration,attr"`
   StartNumber    *int64           `xml:"startNumber,attr"`
   EndNumber      *int64           `xml:"endNumber,attr"`
   Timeline       *SegmentTimeline `xml:"SegmentTimeline"`
}
type SegmentTimeline struct {
   S []STime `xml:"S"`
}
type STime struct {
   T *int64 `xml:"t,attr"`
   D int64  `xml:"d,attr"`
   R *int64 `xml:"r,attr"`
}
type SegmentList struct {
   BaseURL        string          `xml:"BaseURL"`
   Initialization *Initialization `xml:"Initialization"`
   SegmentURL     []SegmentURL    `xml:"SegmentURL"`
}
type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}
type SegmentURL struct {
   Media string `xml:"media,attr"`
}

func main() {
   if len(os.Args) < 2 {
      fmt.Fprintln(os.Stderr, "Usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }

   base, _ := url.Parse(fallbackBase)

   data, err := os.ReadFile(os.Args[1])
   must(err)

   var mpd MPD
   must(xml.Unmarshal(data, &mpd))

   mpdBase := applyBase(base, mpd.BaseURL)
   out := map[string][]string{}

   for _, p := range mpd.Period {
      pBase := applyBase(mpdBase, p.BaseURL)
      pDur, _ := parseISODuration(p.Duration)

      for _, as := range p.AdaptSets {
         asBase := applyBase(pBase, as.BaseURL)

         for _, r := range as.Rep {
            if strings.TrimSpace(r.ID) == "" {
               continue
            }
            repBase := applyBase(asBase, r.BaseURL)

            sl := pickSL(r.SegmentList, as.SegmentList)
            if sl != nil && (sl.Initialization != nil || len(sl.SegmentURL) > 0) {
               slBase := applyBase(repBase, sl.BaseURL)
               if sl.Initialization != nil && strings.TrimSpace(sl.Initialization.SourceURL) != "" {
                  out[r.ID] = append(out[r.ID], resolve(slBase, sl.Initialization.SourceURL))
               }
               for _, su := range sl.SegmentURL {
                  if strings.TrimSpace(su.Media) != "" {
                     out[r.ID] = append(out[r.ID], resolve(slBase, su.Media))
                  }
               }
               continue
            }

            st := pickST(r.SegmentTemplate, as.SegmentTemplate)
            if st == nil || (strings.TrimSpace(st.Media) == "" && (st.Timeline == nil || len(st.Timeline.S) == 0)) {
               out[r.ID] = append(out[r.ID], repBase.String())
               continue
            }

            start := int64(1)
            if st.StartNumber != nil && *st.StartNumber > 0 {
               start = *st.StartNumber
            }

            addInit := func(firstT int64, useTime bool) {
               if strings.TrimSpace(st.Initialization) == "" {
                  return
               }
               u := expand(st.Initialization, r, 0, firstT, false, useTime)
               out[r.ID] = append(out[r.ID], resolve(repBase, u))
            }

            if st.Timeline != nil && len(st.Timeline.S) > 0 {
               times := expandTimeline(st, start, st.EndNumber)
               useTime := strings.Contains(st.Initialization, "$Time$")
               firstT := int64(0)
               if len(times) > 0 {
                  firstT = times[0]
               }
               addInit(firstT, useTime)

               seq := start
               for _, t := range times {
                  u := expand(st.Media, r, seq, t, true, true)
                  out[r.ID] = append(out[r.ID], resolve(repBase, u))
                  seq++
                  if st.EndNumber != nil && seq-1 >= *st.EndNumber {
                     break
                  }
               }
               continue
            }

            addInit(0, false)

            var end *int64
            if st.EndNumber != nil {
               end = st.EndNumber
            } else if st.Duration != nil && pDur > 0 {
               ts := int64(1)
               if st.Timescale != nil && *st.Timescale > 0 {
                  ts = *st.Timescale
               }
               totalTicks := (float64(pDur) / float64(time.Second)) * float64(ts)
               cnt := int64(math.Ceil(totalTicks / float64(*st.Duration)))
               if cnt > 0 {
                  tmp := start + cnt - 1
                  end = &tmp
               }
            }

            if end != nil && *end >= start {
               n := start
               for n <= *end {
                  u := expand(st.Media, r, n, 0, true, false)
                  out[r.ID] = append(out[r.ID], resolve(repBase, u))
                  n++
               }
            }
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetEscapeHTML(false)
   enc.SetIndent("", "  ")
   must(enc.Encode(out))
}

func must(err error) {
   if err != nil {
      fmt.Fprintln(os.Stderr, err)
      os.Exit(1)
   }
}

func applyBase(parent *url.URL, base string) *url.URL {
   raw := strings.TrimSpace(base)
   if raw == "" {
      return parent
   }
   ref, err := url.Parse(raw)
   if err != nil {
      return parent
   }
   return parent.ResolveReference(ref)
}

func resolve(base *url.URL, refStr string) string {
   raw := strings.TrimSpace(refStr)
   if raw == "" {
      return base.String()
   }
   ref, err := url.Parse(raw)
   if err != nil {
      return base.String()
   }
   return base.ResolveReference(ref).String()
}

func pickSL(rep, as *SegmentList) *SegmentList {
   if rep != nil {
      return rep
   }
   return as
}

// EndNumber is not inherited when Rep has its own SegmentTemplate.
func pickST(rep, as *SegmentTemplate) *SegmentTemplate {
   if rep == nil && as == nil {
      return nil
   }
   if rep != nil {
      out := &SegmentTemplate{}
      if as != nil {
         out.Initialization = as.Initialization
         out.Media = as.Media
         out.Timescale = clone64(as.Timescale)
         out.Duration = clone64(as.Duration)
         out.StartNumber = clone64(as.StartNumber)
         if as.Timeline != nil {
            out.Timeline = &SegmentTimeline{S: append([]STime(nil), as.Timeline.S...)}
         }
      }
      if strings.TrimSpace(rep.Initialization) != "" {
         out.Initialization = rep.Initialization
      }
      if strings.TrimSpace(rep.Media) != "" {
         out.Media = rep.Media
      }
      if rep.Timescale != nil {
         out.Timescale = clone64(rep.Timescale)
      }
      if rep.Duration != nil {
         out.Duration = clone64(rep.Duration)
      }
      if rep.StartNumber != nil {
         out.StartNumber = clone64(rep.StartNumber)
      }
      if rep.EndNumber != nil {
         out.EndNumber = clone64(rep.EndNumber)
      }
      if rep.Timeline != nil && len(rep.Timeline.S) > 0 {
         out.Timeline = &SegmentTimeline{S: append([]STime(nil), rep.Timeline.S...)}
      }
      if strings.TrimSpace(out.Media) == "" && strings.TrimSpace(out.Initialization) == "" && out.Timeline == nil {
         return nil
      }
      return out
   }
   if strings.TrimSpace(as.Media) == "" && strings.TrimSpace(as.Initialization) == "" && as.Timeline == nil {
      return nil
   }
   out := &SegmentTemplate{
      Initialization: as.Initialization,
      Media:          as.Media,
      Timescale:      clone64(as.Timescale),
      Duration:       clone64(as.Duration),
      StartNumber:    clone64(as.StartNumber),
      EndNumber:      clone64(as.EndNumber),
   }
   if as.Timeline != nil {
      out.Timeline = &SegmentTimeline{S: append([]STime(nil), as.Timeline.S...)}
   }
   return out
}

func clone64(p *int64) *int64 {
   if p == nil {
      return nil
   }
   v := *p
   return &v
}

var tok = regexp.MustCompile(`\$(Number|Bandwidth|RepresentationID|Time)(%0(\d+)d)?\$`)

func expand(t string, r Representation, n, tm int64, haveN, haveT bool) string {
   const ph = "___DOLLAR___"
   t = strings.ReplaceAll(t, "$$", ph)
   s := tok.ReplaceAllStringFunc(t, func(m string) string {
      sub := tok.FindStringSubmatch(m)
      name := sub[1]
      width := 0
      if sub[3] != "" {
         w, err := strconv.Atoi(sub[3])
         if err == nil && w > 0 {
            width = w
         }
      }
      switch name {
      case "RepresentationID":
         return r.ID
      case "Bandwidth":
         if width > 0 {
            return fmt.Sprintf("%0*d", width, r.Bandwidth)
         }
         return strconv.FormatInt(r.Bandwidth, 10)
      case "Number":
         if !haveN {
            return m
         }
         if width > 0 {
            return fmt.Sprintf("%0*d", width, n)
         }
         return strconv.FormatInt(n, 10)
      case "Time":
         if !haveT {
            return m
         }
         if width > 0 {
            return fmt.Sprintf("%0*d", width, tm)
         }
         return strconv.FormatInt(tm, 10)
      }
      return m
   })
   return strings.ReplaceAll(s, ph, "$")
}

// Expand SegmentTimeline bounded only by next S@t or explicit endNumber.
// If r = -1 and no bound exists, we warn once and stop expansion.
func expandTimeline(st *SegmentTemplate, start int64, endNumber *int64) []int64 {
   var times []int64
   var seq int64
   cur := int64(0)

   for i := range st.Timeline.S {
      s := st.Timeline.S[i]
      if s.D <= 0 {
         continue
      }
      if s.T != nil {
         cur = *s.T
      } else if len(times) == 0 && i == 0 {
         cur = 0
      }

      rep := int64(0)
      if s.R != nil {
         r := *s.R
         if r >= 0 {
            rep = r
         } else {
            until := int64(-1)
            if i+1 < len(st.Timeline.S) && st.Timeline.S[i+1].T != nil {
               until = *st.Timeline.S[i+1].T
            } else if endNumber != nil && *endNumber > 0 {
               remaining := (*endNumber - start + 1) - seq
               if remaining > 0 {
                  rep = remaining - 1
               } else {
                  rep = -1
               }
            }
            if until >= 0 {
               delta := until - cur
               if delta <= 0 {
                  rep = 0
               } else {
                  n := delta / s.D
                  if delta%s.D != 0 {
                     n++
                  }
                  if n > 0 {
                     rep = n - 1
                  } else {
                     rep = 0
                  }
               }
            }
         }
      }

      if s.R != nil && *s.R < 0 && rep == -1 {
         fmt.Fprintln(os.Stderr, "warning: SegmentTimeline has r=-1 without next S@t or endNumber bound; truncating")
         break
      }

      k := int64(0)
      for k <= rep {
         times = append(times, cur)
         seq++
         cur += s.D
         if endNumber != nil && *endNumber > 0 && (start+seq-1) >= *endNumber {
            return times
         }
         k++
      }
   }

   return times
}

// Minimal ISO-8601 duration parser for PnDTnHnMnS (subset).
func parseISODuration(s string) (time.Duration, error) {
   s = strings.TrimSpace(s)
   if s == "" {
      return 0, nil
   }
   re := regexp.MustCompile(`^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?)?$`)
   m := re.FindStringSubmatch(s)
   if m == nil {
      if strings.HasPrefix(s, "PT") && strings.HasSuffix(s, "S") {
         num := strings.TrimSuffix(strings.TrimPrefix(s, "PT"), "S")
         f, err := strconv.ParseFloat(num, 64)
         if err != nil {
            return 0, err
         }
         return time.Duration(f * float64(time.Second)), nil
      }
      return 0, errors.New("unsupported ISO-8601 duration: " + s)
   }
   var days, hours, mins int64
   var secs float64
   if m[1] != "" {
      days, _ = strconv.ParseInt(m[1], 10, 64)
   }
   if m[2] != "" {
      hours, _ = strconv.ParseInt(m[2], 10, 64)
   }
   if m[3] != "" {
      mins, _ = strconv.ParseInt(m[3], 10, 64)
   }
   if m[4] != "" {
      secs, _ = strconv.ParseFloat(m[4], 64)
   }
   total := time.Duration(days)*24*time.Hour +
      time.Duration(hours)*time.Hour +
      time.Duration(mins)*time.Minute +
      time.Duration(secs*float64(time.Second))
   return total, nil
}
