package main

import (
   "encoding/json"
   "encoding/xml"
   "errors"
   "fmt"
   "math"
   "net/url"
   "os"
   "path/filepath"
   "regexp"
   "strconv"
   "strings"
   "time"
)

const fixedBase = "http://test.test/test.mpd"

// ====== MPD model (single BaseURL per level) ======

type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   Type                      string   `xml:"type,attr"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
   BaseURL                   BaseURL  `xml:"BaseURL"`
   Periods                   []Period `xml:"Period"`
}

type Period struct {
   ID       string          `xml:"id,attr"`
   Duration string          `xml:"duration,attr"` // Period@duration
   BaseURL  BaseURL         `xml:"BaseURL"`
   AS       []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   ID       string           `xml:"id,attr"`
   BaseURL  BaseURL          `xml:"BaseURL"`
   Template *SegmentTemplate `xml:"SegmentTemplate"`
   List     *SegmentList     `xml:"SegmentList"`
   Reps     []Representation `xml:"Representation"`
}

type Representation struct {
   ID        string           `xml:"id,attr"`
   Bandwidth *int64           `xml:"bandwidth,attr"`
   BaseURL   BaseURL          `xml:"BaseURL"`
   Template  *SegmentTemplate `xml:"SegmentTemplate"`
   List      *SegmentList     `xml:"SegmentList"`
}

type BaseURL struct {
   URL string `xml:",chardata"`
}

type SegmentTemplate struct {
   Initialization string           `xml:"initialization,attr"`
   Media          string           `xml:"media,attr"`
   StartNumber    *int64           `xml:"startNumber,attr"`
   Timescale      *int64           `xml:"timescale,attr"`
   Duration       *int64           `xml:"duration,attr"`
   EndNumber      *int64           `xml:"endNumber,attr"`
   Timeline       *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T *int64 `xml:"t,attr"`
   D int64  `xml:"d,attr"`
   R *int64 `xml:"r,attr"`
}

type SegmentList struct {
   Timescale      *int64          `xml:"timescale,attr"`
   Duration       *int64          `xml:"duration,attr"`
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media      string `xml:"media,attr"`
   MediaRange string `xml:"mediaRange,attr"`
   IndexRange string `xml:"indexRange,attr"`
}

// ====== main ======

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "usage: %s <mpd_file_path>\n", filepath.Base(os.Args[0]))
      os.Exit(2)
   }
   data, err := os.ReadFile(os.Args[1])
   if err != nil {
      fmt.Fprintln(os.Stderr, "open:", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Fprintln(os.Stderr, "xml:", err)
      os.Exit(1)
   }

   base := mustParseBase(fixedBase)
   mpdBase := inheritBase(base, mpd.BaseURL)

   // MPD duration fallback
   mpdDur := 0.0
   if mpd.MediaPresentationDuration != "" {
      if d, err := parseISODuration(mpd.MediaPresentationDuration); err == nil {
         mpdDur = d.Seconds()
      }
   }

   out := map[string][]string{}

   for pi, p := range mpd.Periods {
      pBase := inheritBase(mpdBase, p.BaseURL)

      pDur := 0.0
      if p.Duration != "" {
         if d, err := parseISODuration(p.Duration); err == nil {
            pDur = d.Seconds()
         }
      } else if mpdDur > 0 && len(mpd.Periods) > 0 {
         pDur = mpdDur / float64(len(mpd.Periods))
      }

      for _, as := range p.AS {
         asBase := inheritBase(pBase, as.BaseURL)
         for _, r := range as.Reps {
            repBase := inheritBase(asBase, r.BaseURL)

            repID := r.ID
            if repID == "" {
               repID = fmt.Sprintf("p%d_as%s_rep", pi, nz(as.ID, "x"))
            }
            bw := ""
            if r.Bandwidth != nil {
               bw = strconv.FormatInt(*r.Bandwidth, 10)
            }

            tpl := pickTemplate(r.Template, as.Template)
            lst := pickList(r.List, as.List)

            var urls []string

            // SegmentList (explicit URLs)
            if lst != nil {
               if lst.Initialization != nil && lst.Initialization.SourceURL != "" {
                  if u, ok := resolve(repBase, lst.Initialization.SourceURL); ok {
                     urls = append(urls, u)
                  }
               }
               for _, su := range lst.SegmentURLs {
                  if su.Media != "" {
                     if u, ok := resolve(repBase, su.Media); ok {
                        urls = append(urls, u)
                     }
                  } else {
                     // Only ranges → still output the resource URL once per entry
                     urls = append(urls, repBase.String())
                  }
               }
            } else if tpl != nil {
               // Initialization (if present)
               if tpl.Initialization != "" {
                  initURL := expandTemplate(tpl.Initialization, map[string]string{
                     "RepresentationID": repID,
                     "Bandwidth":        bw,
                     "Number":           "",
                     "Time":             "",
                  })
                  if u, ok := resolve(repBase, initURL); ok {
                     urls = append(urls, u)
                  }
               }

               // Segments
               start := int64(1)
               if tpl.StartNumber != nil {
                  start = *tpl.StartNumber
               }
               hasEnd := tpl.EndNumber != nil
               end := int64(0)
               if hasEnd {
                  end = *tpl.EndNumber
               }
               ts := int64(1)
               if tpl.Timescale != nil && *tpl.Timescale > 0 {
                  ts = *tpl.Timescale
               }

               type seg struct {
                  num  int64
                  time *int64
               }
               var segs []seg

               // SegmentTimeline
               if tl := tpl.Timeline; tl != nil && len(tl.S) > 0 {
                  num := start
                  var tcur int64
                  for i, s := range tl.S {
                     if s.D <= 0 {
                        continue
                     }
                     if s.T != nil {
                        tcur = *s.T
                     } else if i == 0 {
                        tcur = 0
                     }
                     reps := int64(0)
                     if s.R != nil {
                        reps = *s.R
                        if reps < 0 {
                           if hasEnd {
                              reps = end - num
                           } else if pDur > 0 {
                              remain := int64(pDur*float64(ts)) - tcur
                              if remain > 0 {
                                 reps = remain/s.D - 1
                              }
                           }
                           if reps < 0 {
                              reps = 0
                           }
                        }
                     }
                     for rct := int64(0); rct <= reps; rct++ {
                        if hasEnd && num > end {
                           break
                        }
                        segs = append(segs, seg{num: num, time: int64Ptr(tcur)})
                        num++
                        tcur += s.D // $Time$ increases by S@d for each iteration
                     }
                     if hasEnd && len(segs) > 0 && segs[len(segs)-1].num >= end {
                        break
                     }
                  }
               } else if tpl.Duration != nil && *tpl.Duration > 0 {
                  segDur := *tpl.Duration
                  if hasEnd {
                     for n := start; n <= end; n++ {
                        segs = append(segs, seg{num: n})
                     }
                  } else {
                     count := int64(1)
                     if pDur > 0 {
                        numTicks := int64(math.Ceil(pDur * float64(ts))) // avoid truncation
                        count = ceilDiv(numTicks, segDur)
                        if count < 1 {
                           count = 1
                        }
                     }
                     for i := int64(0); i < count; i++ {
                        segs = append(segs, seg{num: start + i})
                     }
                  }
               } else if hasEnd {
                  // Number-based addressing without duration/timeline
                  for n := start; n <= end; n++ {
                     segs = append(segs, seg{num: n})
                  }
               }

               // Expand media template
               if tpl.Media != "" {
                  for _, s := range segs {
                     vars := map[string]string{
                        "RepresentationID": repID,
                        "Bandwidth":        bw,
                        "Number":           strconv.FormatInt(s.num, 10),
                        "Time":             "",
                     }
                     if s.time != nil {
                        vars["Time"] = strconv.FormatInt(*s.time, 10)
                     }
                     m := expandTemplate(tpl.Media, vars)
                     if u, ok := resolve(repBase, m); ok {
                        urls = append(urls, u)
                     }
                  }
               }
            }

            // Fallback: Representation only has BaseURL
            if len(urls) == 0 && tpl == nil && lst == nil {
               if u, ok := resolve(repBase, ""); ok {
                  out[repID] = append(out[repID], u)
                  continue
               }
            }

            if len(urls) > 0 {
               out[repID] = append(out[repID], urls...)
            } else if _, ok := out[repID]; !ok {
               out[repID] = []string{}
            }
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   if err := enc.Encode(out); err != nil {
      fmt.Fprintln(os.Stderr, "json:", err)
      os.Exit(1)
   }
}

// ====== helpers ======

func mustParseBase(s string) *url.URL {
   u, err := url.Parse(s)
   if err != nil {
      panic(err)
   }
   return u
}

func inheritBase(base *url.URL, b BaseURL) *url.URL {
   last := strings.TrimSpace(b.URL)
   if last == "" {
      return base
   }
   r, err := url.Parse(last)
   if err != nil {
      return base
   }
   return base.ResolveReference(r)
}

func resolve(base *url.URL, ref string) (string, bool) {
   r, err := url.Parse(ref)
   if err != nil {
      return "", false
   }
   return base.ResolveReference(r).String(), true
}

// PnDTnHnMnS (days→86400s). Enough for MPD/Period durations we use.
func parseISODuration(s string) (time.Duration, error) {
   if s == "" || !strings.HasPrefix(s, "P") {
      return 0, errors.New("invalid duration")
   }
   rem := s[1:]
   var days, hours, mins int64
   var secs float64
   num := ""
   flush := func(sfx byte) error {
      if num == "" {
         return nil
      }
      v, err := strconv.ParseFloat(num, 64)
      if err != nil {
         return err
      }
      switch sfx {
      case 'D':
         days += int64(v)
      case 'H':
         hours += int64(v)
      case 'M':
         mins += int64(v)
      case 'S':
         secs += v
      default:
         return fmt.Errorf("unknown field %c", sfx)
      }
      num = ""
      return nil
   }
   for i := 0; i < len(rem); i++ {
      c := rem[i]
      if (c >= '0' && c <= '9') || c == '.' {
         num += string(c)
         continue
      }
      if err := flush(c); err != nil {
         return 0, err
      }
   }
   if num != "" {
      return 0, fmt.Errorf("dangling number in duration: %s", s)
   }
   total := days*86400 + hours*3600 + mins*60
   return time.Duration((float64(total) + secs) * float64(time.Second)), nil
}

func ceilDiv(a, b int64) int64 {
   if b == 0 {
      return 0
   }
   return (a + b - 1) / b
}

func int64Ptr(v int64) *int64 { return &v }

// $Var$ and $Var%fmt$ (e.g., $Number%08d$). Unknown vars left intact. Supports $$ → $.
func expandTemplate(tmpl string, vars map[string]string) string {
   t := strings.ReplaceAll(tmpl, "$$", "\x00DOLLAR\x00")
   re := regexp.MustCompile(`\$(\w+)(?:%([^$]+))?\$`)
   t = re.ReplaceAllStringFunc(t, func(m string) string {
      ms := re.FindStringSubmatch(m)
      key, fmtSpec := ms[1], ms[2]
      val, ok := vars[key]
      if !ok {
         return m
      }
      if fmtSpec != "" {
         if i, err := strconv.ParseInt(val, 10, 64); err == nil {
            return fmt.Sprintf("%"+fmtSpec, i)
         }
      }
      return val
   })
   return strings.ReplaceAll(t, "\x00DOLLAR\x00", "$")
}

func pickTemplate(rep, as *SegmentTemplate) *SegmentTemplate {
   if rep != nil {
      return rep
   }
   return as
}

func pickList(rep, as *SegmentList) *SegmentList {
   if rep != nil {
      return rep
   }
   return as
}

func nz(a, b string) string {
   if a != "" {
      return a
   }
   return b
}
