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

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL        string          `xml:"BaseURL"`
   Duration       string          `xml:"duration,attr"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentList struct {
   Initialization Initialization `xml:"Initialization"`
   Segments       []SegmentURL   `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

// Note: StartNumber is a pointer so we can tell "missing" (nil) from explicit "0".
type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   Timescale       int              `xml:"timescale,attr"`
   Duration        int              `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T int64 `xml:"t,attr"`
   D int64 `xml:"d,attr"`
   R int64 `xml:"r,attr"`
}

// varPattern matches $Number$ or $Number%08d$ (and Time variants).
var varPattern = regexp.MustCompile(`\$(Number|Time)(%0?[0-9]+d)?\$`)

func replaceVars(tmpl, repID string, number int, t int64) string {
   // Always replace RepresentationID first
   tmpl = strings.ReplaceAll(tmpl, "$RepresentationID$", repID)

   // Replace Number/Time with optional printf style
   return varPattern.ReplaceAllStringFunc(tmpl, func(m string) string {
      parts := varPattern.FindStringSubmatch(m)
      if len(parts) < 2 {
         return m
      }
      name := parts[1]
      format := parts[2]
      if format == "" {
         format = "%d"
      }
      switch name {
      case "Number":
         return fmt.Sprintf(format, number)
      case "Time":
         return fmt.Sprintf(format, t)
      default:
         return m
      }
   })
}

func joinURL(base, ref string) string {
   u, err := url.Parse(base)
   if err != nil {
      return ref
   }
   r, err := url.Parse(ref)
   if err != nil {
      return ref
   }
   return u.ResolveReference(r).String()
}

func fallbackBase(parent, child string) string {
   if child != "" {
      return joinURL(parent, child)
   }
   return parent
}

// parseISODuration: minimal support for PT...S (e.g. PT30S or PT1.5S).
// If more complete ISO8601 durations are needed (PT1H2M3.5S), this should be extended.
func parseISODuration(d string) float64 {
   if d == "" {
      return 0
   }
   if strings.HasPrefix(d, "PT") && strings.HasSuffix(d, "S") {
      num := strings.TrimSuffix(strings.TrimPrefix(d, "PT"), "S")
      if v, err := strconv.ParseFloat(num, 64); err == nil {
         return v
      }
   }
   return 0
}

func getStartNumber(tmpl *SegmentTemplate) int {
   // If attribute is missing (nil) => default 1
   // If attribute present (including explicit 0) => use that value
   if tmpl.StartNumber == nil {
      return 1
   }
   return *tmpl.StartNumber
}

func expandSegmentTemplate(tmpl *SegmentTemplate, repID, base string, periodDurSec float64) []string {
   var segs []string

   // timescale defaults to 1 when missing
   timescale := tmpl.Timescale
   if timescale == 0 {
      timescale = 1
   }

   startNumber := getStartNumber(tmpl)

   // Initialization (may use $Number$ etc.)
   if tmpl.Initialization != "" {
      segs = append(segs, joinURL(base, replaceVars(tmpl.Initialization, repID, startNumber, 0)))
   }

   // SegmentTimeline (priority)
   if tmpl.SegmentTimeline != nil && len(tmpl.SegmentTimeline.S) > 0 {
      number := startNumber
      // currentTime initial value: if first S has T use it; else start at 0
      currentTime := int64(0)
      if len(tmpl.SegmentTimeline.S) > 0 && tmpl.SegmentTimeline.S[0].T != 0 {
         currentTime = tmpl.SegmentTimeline.S[0].T
      }
      for _, s := range tmpl.SegmentTimeline.S {
         if s.T != 0 {
            currentTime = s.T
         }
         repeat := s.R
         if repeat < 0 {
            // negative repeat: repeat until next S or until period end — approximate via period duration if available
            if s.D > 0 && periodDurSec > 0 {
               est := int64(math.Ceil(periodDurSec*float64(timescale)/float64(s.D))) - 1
               if est < 0 {
                  est = 0
               }
               repeat = est
            } else {
               // fallback: treat as single occurrence
               repeat = 0
            }
         }
         for i := int64(0); i <= repeat; i++ {
            segs = append(segs, joinURL(base, replaceVars(tmpl.Media, repID, number, currentTime)))
            currentTime += s.D
            number++
         }
      }
      return segs
   }

   // endNumber explicit range
   if tmpl.Media != "" && tmpl.EndNumber > 0 {
      number := startNumber
      for i := number; i <= tmpl.EndNumber; i++ {
         segs = append(segs, joinURL(base, replaceVars(tmpl.Media, repID, i, 0)))
      }
      return segs
   }

   // duration+timescale fallback (when SegmentTimeline & endNumber missing)
   if tmpl.Media != "" && tmpl.Duration > 0 && periodDurSec > 0 {
      count := int(math.Ceil(periodDurSec * float64(timescale) / float64(tmpl.Duration)))
      number := startNumber
      for i := 0; i < count; i++ {
         segs = append(segs, joinURL(base, replaceVars(tmpl.Media, repID, number+i, 0)))
      }
      return segs
   }

   return segs
}

func main() {
   if len(os.Args) < 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }

   rootBase := "http://test.test/test.mpd"

   data, err := os.ReadFile(os.Args[1])
   if err != nil {
      fmt.Println("Error reading MPD file:", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Println("Error parsing MPD XML:", err)
      os.Exit(1)
   }

   results := make(map[string][]string)

   // mpd.BaseURL (if present) chains on rootBase; period-level resolution uses the result
   for _, period := range mpd.Periods {
      periodBase := fallbackBase(rootBase, mpd.BaseURL)
      periodBase = fallbackBase(periodBase, period.BaseURL)
      periodDurSec := parseISODuration(period.Duration)

      for _, aset := range period.AdaptationSets {
         asetBase := fallbackBase(periodBase, aset.BaseURL)

         for _, rep := range aset.Representations {
            repBase := fallbackBase(asetBase, rep.BaseURL)

            var segs []string

            // SegmentList (explicit URLs)
            if rep.SegmentList != nil {
               if rep.SegmentList.Initialization.SourceURL != "" {
                  segs = append(segs, joinURL(repBase, rep.SegmentList.Initialization.SourceURL))
               }
               for _, s := range rep.SegmentList.Segments {
                  segs = append(segs, joinURL(repBase, s.Media))
               }
            }

            // SegmentTemplate (rep-level overrides aset-level)
            var tmpl *SegmentTemplate
            if rep.SegmentTemplate != nil {
               tmpl = rep.SegmentTemplate
            } else {
               tmpl = aset.SegmentTemplate
            }
            if tmpl != nil {
               segs = append(segs, expandSegmentTemplate(tmpl, rep.ID, repBase, periodDurSec)...)
            }

            // If representation had only BaseURL and no segment info, use resolved base directly
            if rep.SegmentList == nil && rep.SegmentTemplate == nil && rep.BaseURL != "" {
               segs = append(segs, repBase)
            }

            // Append (don't overwrite) if same Representation ID appears in multiple Periods
            results[rep.ID] = append(results[rep.ID], segs...)
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   if err := enc.Encode(results); err != nil {
      fmt.Println("Error encoding JSON:", err)
      os.Exit(1)
   }
}
