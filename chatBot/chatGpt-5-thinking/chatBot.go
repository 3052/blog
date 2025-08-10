package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
)

const baseMPDURL = "http://test.test/test.mpd"

// ===== XML model (minimal but practical) =====

type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
   BaseURL                   string   `xml:"BaseURL"`
   Periods                   []Period `xml:"Period"`
}

type Period struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   AdaptationSets  []AdaptationSet  `xml:"AdaptationSet"`
   // Prefer Period's own duration if possible
   Duration string `xml:"duration,attr"`
   Start    string `xml:"start,attr"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   Bandwidth       int64            `xml:"bandwidth,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     *int64           `xml:"startNumber,attr"` // pointer: detect missing vs "0"
   EndNumber       *int64           `xml:"endNumber,attr"`   // pointer: present => cap last segment
   Timescale       int64            `xml:"timescale,attr"`
   Duration        int64            `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
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
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media      string `xml:"media,attr"`
   MediaRange string `xml:"mediaRange,attr"`
   Index      string `xml:"index,attr"`
   IndexRange string `xml:"indexRange,attr"`
}

// ===== Utilities =====

func trim(s string) string { return strings.TrimSpace(s) }

// Resolve ref against base using ONLY net/url's ResolveReference.
func resolveURL(baseStr, refStr string) (string, error) {
   baseStr = trim(baseStr)
   refStr = trim(refStr)

   bu, err := url.Parse(baseStr)
   if err != nil {
      return "", fmt.Errorf("invalid base URL %q: %w", baseStr, err)
   }
   if refStr == "" {
      return bu.String(), nil
   }
   ru, err := url.Parse(refStr)
   if err != nil {
      return "", fmt.Errorf("invalid ref URL %q: %w", refStr, err)
   }
   return bu.ResolveReference(ru).String(), nil
}

// Combine hierarchical BaseURL chain: MPD -> Period -> Adaptation -> Representation
func chainBaseURL(mpdBase string, mpd *MPD, p *Period, a *AdaptationSet, r *Representation) (string, error) {
   base := mpdBase
   var err error

   if u := trim(mpd.BaseURL); u != "" {
      base, err = resolveURL(base, u)
      if err != nil {
         return "", err
      }
   }
   if p != nil {
      if u := trim(p.BaseURL); u != "" {
         base, err = resolveURL(base, u)
         if err != nil {
            return "", err
         }
      }
   }
   if a != nil {
      if u := trim(a.BaseURL); u != "" {
         base, err = resolveURL(base, u)
         if err != nil {
            return "", err
         }
      }
   }
   if r != nil {
      if u := trim(r.BaseURL); u != "" {
         base, err = resolveURL(base, u)
         if err != nil {
            return "", err
         }
      }
   }
   return base, nil
}

// Merge SegmentTemplate inheritance: Period -> AdaptationSet -> Representation
// Respect explicit startNumber="0" by only defaulting when StartNumber is nil.
func mergeTemplates(parent, child *SegmentTemplate) *SegmentTemplate {
   if parent == nil && child == nil {
      return nil
   }
   var out SegmentTemplate
   if parent != nil {
      out = *parent
   }
   if child != nil {
      if child.Initialization != "" {
         out.Initialization = child.Initialization
      }
      if child.Media != "" {
         out.Media = child.Media
      }
      if child.StartNumber != nil {
         out.StartNumber = child.StartNumber
      }
      if child.EndNumber != nil {
         out.EndNumber = child.EndNumber
      }
      if child.Timescale != 0 {
         out.Timescale = child.Timescale
      }
      if child.Duration != 0 {
         out.Duration = child.Duration
      }
      if child.SegmentTimeline != nil {
         out.SegmentTimeline = child.SegmentTimeline
      }
   }
   // Defaults
   if out.StartNumber == nil {
      def := int64(1) // missing -> default 1
      out.StartNumber = &def
   }
   if out.Timescale == 0 {
      out.Timescale = 1
   }
   return &out
}

// ISO 8601 duration parser (supports PnDTnHnMnS)
var isoDurRe = regexp.MustCompile(`^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?)?$`)

func parseISODuration(s string) float64 {
   s = strings.TrimSpace(s)
   if s == "" {
      return 0
   }
   m := isoDurRe.FindStringSubmatch(s)
   if m == nil {
      return 0
   }
   toFloat := func(x string) float64 {
      if x == "" {
         return 0
      }
      v, _ := strconv.ParseFloat(x, 64)
      return v
   }
   days := toFloat(m[1])
   hours := toFloat(m[2])
   mins := toFloat(m[3])
   secs := toFloat(m[4])
   return days*86400 + hours*3600 + mins*60 + secs
}

// Template substitution supporting $RepresentationID$, $Bandwidth$, $Number$, $Time$
// and printf-like zero-padding (e.g., $Number%05d$)
var tplRe = regexp.MustCompile(`\$(RepresentationID|Bandwidth|Number|Time)(?:%0?(\d+)d)?\$`)

func fillTemplate(tpl string, rep Representation, number *int64, timeVal *int64) string {
   return tplRe.ReplaceAllStringFunc(tpl, func(match string) string {
      sub := tplRe.FindStringSubmatch(match)
      key, padStr := sub[1], sub[2]
      var valStr string
      switch key {
      case "RepresentationID":
         valStr = url.PathEscape(rep.ID)
      case "Bandwidth":
         valStr = strconv.FormatInt(rep.Bandwidth, 10)
      case "Number":
         n := int64(0)
         if number != nil {
            n = *number
         }
         valStr = strconv.FormatInt(n, 10)
      case "Time":
         t := int64(0)
         if timeVal != nil {
            t = *timeVal
         }
         valStr = strconv.FormatInt(t, 10)
      }
      if padStr != "" {
         width, _ := strconv.Atoi(padStr)
         if len(valStr) < width {
            valStr = strings.Repeat("0", width-len(valStr)) + valStr
         }
      }
      return valStr
   })
}

// Build times from SegmentTimeline S elements
func expandTimeline(st *SegmentTimeline) []int64 {
   if st == nil {
      return nil
   }
   var times []int64
   var cur int64 = 0
   first := true
   for _, s := range st.S {
      if first {
         if s.T != nil {
            cur = *s.T
         } else {
            cur = 0
         }
         first = false
      } else if s.T != nil {
         cur = *s.T
      }
      reps := int64(0)
      if s.R != nil {
         reps = *s.R
         if reps < 0 {
            reps = 0 // don't attempt infinite lists
         }
      }
      count := reps + 1
      for i := int64(0); i < count; i++ {
         times = append(times, cur)
         cur += s.D
      }
   }
   return times
}

// Determine a Period's duration (seconds) if possible.
// Priority: Period@duration > (nextPeriod@start - this@start) > MPD@mediaPresentationDuration - start.
func periodDurationSeconds(mpd *MPD, pi int, mpdTotal float64) float64 {
   p := &mpd.Periods[pi]

   if d := parseISODuration(p.Duration); d > 0 {
      return d
   }
   curStart := parseISODuration(p.Start)
   if pi+1 < len(mpd.Periods) {
      nextStart := parseISODuration(mpd.Periods[pi+1].Start)
      if nextStart > curStart && curStart >= 0 {
         return nextStart - curStart
      }
   }
   if mpdTotal > 0 && curStart >= 0 && mpdTotal > curStart {
      return mpdTotal - curStart
   }
   return 0
}

// ===== Core logic =====

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: go run main.go <mpd_file_path>\n")
      os.Exit(1)
   }
   mpdPath := os.Args[1]
   data, err := os.ReadFile(mpdPath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading MPD: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD XML: %v\n", err)
      os.Exit(1)
   }

   result := make(map[string][]string)
   totalDurSeconds := parseISODuration(mpd.MediaPresentationDuration)

   // Walk MPD -> Period -> AdaptationSet -> Representation
   for pi := range mpd.Periods {
      period := &mpd.Periods[pi]
      thisPeriodDur := periodDurationSeconds(&mpd, pi, totalDurSeconds)

      for ai := range period.AdaptationSets {
         aset := &period.AdaptationSets[ai]
         for ri := range aset.Representations {
            rep := &aset.Representations[ri]
            if strings.TrimSpace(rep.ID) == "" {
               continue
            }

            base, err := chainBaseURL(baseMPDURL, &mpd, period, aset, rep)
            if err != nil {
               fmt.Fprintf(os.Stderr, "Error resolving BaseURL chain: %v\n", err)
               os.Exit(1)
            }

            // Effective SegmentTemplate via inheritance
            tpl := mergeTemplates(period.SegmentTemplate, aset.SegmentTemplate)
            tpl = mergeTemplates(tpl, rep.SegmentTemplate)

            var urls []string

            // 1) Initialization (SegmentList or Template)
            if rep.SegmentList != nil && rep.SegmentList.Initialization != nil && rep.SegmentList.Initialization.SourceURL != "" {
               initURL, err := resolveURL(base, rep.SegmentList.Initialization.SourceURL)
               if err != nil {
                  fmt.Fprintf(os.Stderr, "Error resolving initialization URL: %v\n", err)
                  os.Exit(1)
               }
               urls = append(urls, initURL)
            } else if tpl != nil && tpl.Initialization != "" {
               filled := fillTemplate(tpl.Initialization, *rep, nil, nil)
               initURL, err := resolveURL(base, filled)
               if err != nil {
                  fmt.Fprintf(os.Stderr, "Error resolving initialization template URL: %v\n", err)
                  os.Exit(1)
               }
               urls = append(urls, initURL)
            }

            // 2) Media segments
            if rep.SegmentList != nil && len(rep.SegmentList.SegmentURLs) > 0 {
               for _, su := range rep.SegmentList.SegmentURLs {
                  if su.Media != "" {
                     u, err := resolveURL(base, su.Media)
                     if err != nil {
                        fmt.Fprintf(os.Stderr, "Error resolving SegmentList media URL: %v\n", err)
                        os.Exit(1)
                     }
                     urls = append(urls, u)
                  }
               }
            } else if tpl != nil && tpl.Media != "" {
               // Prefer SegmentTimeline if present
               if tpl.SegmentTimeline != nil && len(tpl.SegmentTimeline.S) > 0 {
                  times := expandTimeline(tpl.SegmentTimeline)
                  num := *tpl.StartNumber
                  for _, t := range times {
                     filled := fillTemplate(tpl.Media, *rep, &num, &t)
                     u, err := resolveURL(base, filled)
                     if err != nil {
                        fmt.Fprintf(os.Stderr, "Error resolving Media URL from timeline: %v\n", err)
                        os.Exit(1)
                     }
                     urls = append(urls, u)
                     num++
                  }
               } else {
                  // Number-based addressing (no timeline)
                  // We may have Duration (for count), EndNumber (for cap), or both.
                  var countFromDur int // 0 means unknown
                  if tpl.Duration > 0 {
                     segSec := float64(tpl.Duration) / float64(max64(1, tpl.Timescale))
                     if segSec > 0 {
                        durForCount := thisPeriodDur
                        if durForCount <= 0 {
                           durForCount = totalDurSeconds
                        }
                        if durForCount > 0 {
                           countFromDur = int(math.Ceil(durForCount / segSec))
                        }
                     }
                  }

                  start := *tpl.StartNumber
                  var countFromEnd int // 0 means unknown
                  if tpl.EndNumber != nil {
                     if *tpl.EndNumber >= start {
                        cfe := (*tpl.EndNumber - start + 1)
                        if cfe > 0 && cfe < math.MaxInt32 {
                           countFromEnd = int(cfe)
                        }
                     } else {
                        countFromEnd = 0
                     }
                  }

                  // Decide final count
                  finalCount := 0
                  switch {
                  case countFromDur > 0 && countFromEnd > 0:
                     if countFromDur < countFromEnd {
                        finalCount = countFromDur
                     } else {
                        finalCount = countFromEnd
                     }
                  case countFromDur > 0:
                     finalCount = countFromDur
                  case countFromEnd > 0:
                     finalCount = countFromEnd
                  }

                  // Emit segments
                  if finalCount > 0 {
                     num := start
                     for i := 0; i < finalCount; i++ {
                        filled := fillTemplate(tpl.Media, *rep, &num, nil)
                        u, err := resolveURL(base, filled)
                        if err != nil {
                           fmt.Fprintf(os.Stderr, "Error resolving Media URL: %v\n", err)
                           os.Exit(1)
                        }
                        urls = append(urls, u)
                        num++
                     }
                  }
               }
            }

            // 3) Fallback: Representation has only BaseURL (no segments)
            if len(urls) == 0 {
               noSegTemplate := (tpl == nil) ||
                  (tpl.Media == "" && tpl.Initialization == "" && tpl.Duration == 0 &&
                     (tpl.SegmentTimeline == nil || len(tpl.SegmentTimeline.S) == 0))
               noSegList := (rep.SegmentList == nil || len(rep.SegmentList.SegmentURLs) == 0)
               if noSegTemplate && noSegList {
                  urls = append(urls, base)
               }
            }

            // 4) Append to existing entries for same Representation ID (across Periods)
            if len(urls) > 0 {
               result[rep.ID] = append(result[rep.ID], urls...)
            }
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   if err := enc.Encode(result); err != nil {
      fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
      os.Exit(1)
   }
}

func max64(a, b int64) int64 {
   if a > b {
      return a
   }
   return b
}
