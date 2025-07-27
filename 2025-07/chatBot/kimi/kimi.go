package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
   "time"
)

// ---------- MPD structures ----------

type MPD struct {
   XMLName                  xml.Name `xml:"MPD"`
   MediaPresentationDurAttr string   `xml:"mediaPresentationDuration,attr"`
   Period                   []Period `xml:"Period"`
}

type Period struct {
   DurAttr string  `xml:"duration,attr"`
   BaseURL string  `xml:"BaseURL"`
   Adapt   []Adapt `xml:"AdaptationSet"`
}

type Adapt struct {
   BaseURL        string           `xml:"BaseURL"`
   Segment        *Segment         `xml:"SegmentTemplate"`
   Representation []Representation `xml:"Representation"`
}

type Representation struct {
   ID      string   `xml:"id,attr"`
   BaseURL string   `xml:"BaseURL"`
   Segment *Segment `xml:"SegmentTemplate"`
}

type Segment struct {
   Media     string `xml:"media,attr"`
   Init      string `xml:"initialization,attr"`
   StartNum  string `xml:"startNumber,attr"`
   Timescale string `xml:"timescale,attr"`
   Duration  string `xml:"duration,attr"`
   Timeline  []S    `xml:"SegmentTimeline>S"`
}

type S struct {
   T string `xml:"t,attr"`
   D string `xml:"d,attr"`
   R string `xml:"r,attr"`
}

// ---------- helpers ----------

func must(err error) {
   if err != nil {
      fmt.Fprintf(os.Stderr, "error: %v\n", err)
      os.Exit(1)
   }
}

// parseISO8601Duration parses PT[n]H[n]M[n]S
func parseISO8601Duration(d string) (time.Duration, error) {
   if d == "" {
      return 0, nil
   }
   re := regexp.MustCompile(`^PT(?:(\d+(?:\.\d+)?)H)?(?:(\d+(?:\.\d+)?)M)?(?:(\d+(?:\.\d+)?)S)?$`)
   m := re.FindStringSubmatch(d)
   if m == nil {
      return 0, fmt.Errorf("unsupported duration: %s", d)
   }
   var sec float64
   if m[1] != "" {
      h, _ := strconv.ParseFloat(m[1], 64)
      sec += h * 3600
   }
   if m[2] != "" {
      min, _ := strconv.ParseFloat(m[2], 64)
      sec += min * 60
   }
   if m[3] != "" {
      s, _ := strconv.ParseFloat(m[3], 64)
      sec += s
   }
   return time.Duration(sec * float64(time.Second)), nil
}

// segmentTemplate returns effective template
func segmentTemplate(adapt *Adapt, rep *Representation) *Segment {
   if rep.Segment != nil {
      return rep.Segment
   }
   return adapt.Segment
}

// expandTemplate handles $RepresentationID$, $Number$, $Time$, %0xd
func expandTemplate(tmpl string, repID string, num int64, time int64) string {
   tmpl = strings.ReplaceAll(tmpl, "$RepresentationID$", repID)
   tmpl = strings.ReplaceAll(tmpl, "$Number$", fmt.Sprintf("%d", num))
   tmpl = strings.ReplaceAll(tmpl, "$Time$", fmt.Sprintf("%d", time))
   re := regexp.MustCompile(`%(\d+)d`)
   tmpl = re.ReplaceAllStringFunc(tmpl, func(s string) string {
      width, err := strconv.Atoi(s[1 : len(s)-1])
      must(err)
      return fmt.Sprintf("%0*d", width, num)
   })
   return tmpl
}

// ---------- main ----------

func main() {
   if len(os.Args) < 2 {
      fmt.Fprintln(os.Stderr, "usage: mpdexpand <path/to/manifest.mpd>")
      os.Exit(1)
   }
   mpdPath := os.Args[1]

   data, err := os.ReadFile(mpdPath)
   must(err)

   var mpd MPD
   must(xml.Unmarshal(data, &mpd))

   rootURL, err := url.Parse("http://test.test/test.mpd")
   must(err)

   out := make(map[string][]string)

   // presentation duration (fallback)
   var presentationDur time.Duration
   if mpd.MediaPresentationDurAttr != "" {
      presentationDur, err = parseISO8601Duration(mpd.MediaPresentationDurAttr)
      must(err)
   }

   for _, period := range mpd.Period {
      periodDur := presentationDur
      if period.DurAttr != "" {
         periodDur, err = parseISO8601Duration(period.DurAttr)
         must(err)
      }

      for _, adapt := range period.Adapt {
         for _, rep := range adapt.Representation {
            tmpl := segmentTemplate(&adapt, &rep)
            if tmpl == nil {
               continue
            }

            // Build absolute base URL
            base := *rootURL
            if period.BaseURL != "" {
               pu, err := url.Parse(period.BaseURL)
               must(err)
               base = *base.ResolveReference(pu)
            }
            if adapt.BaseURL != "" {
               au, err := url.Parse(adapt.BaseURL)
               must(err)
               base = *base.ResolveReference(au)
            }
            if rep.BaseURL != "" {
               ru, err := url.Parse(rep.BaseURL)
               must(err)
               base = *base.ResolveReference(ru)
            }

            // ----- Initialization segment -----
            if tmpl.Init != "" {
               init := expandTemplate(tmpl.Init, rep.ID, 0, 0)
               initURL, err := url.Parse(init)
               must(err)
               out[rep.ID] = append(out[rep.ID], base.ResolveReference(initURL).String())
            }

            // ----- Media segments -----
            if tmpl.Media == "" {
               continue
            }

            startNum := int64(1)
            if tmpl.StartNum != "" {
               startNum, err = strconv.ParseInt(tmpl.StartNum, 10, 64)
               must(err)
            }

            // SegmentTimeline present
            if len(tmpl.Timeline) > 0 {
               segNum := startNum
               var time int64
               for _, s := range tmpl.Timeline {
                  d, err := strconv.ParseInt(s.D, 10, 64)
                  must(err)
                  repeat := int64(0)
                  if s.R != "" {
                     repeat, err = strconv.ParseInt(s.R, 10, 64)
                     must(err)
                  }
                  if s.T != "" {
                     time, err = strconv.ParseInt(s.T, 10, 64)
                     must(err)
                  }
                  // total segments = 1 + @r
                  count := repeat + 1
                  for i := int64(0); i < count; i++ {
                     media := expandTemplate(tmpl.Media, rep.ID, segNum, time)
                     mediaURL, err := url.Parse(media)
                     must(err)
                     out[rep.ID] = append(out[rep.ID], base.ResolveReference(mediaURL).String())
                     segNum++
                     time += d
                  }
               }
               continue
            }

            // Duration-only mode
            duration := int64(0)
            if tmpl.Duration != "" {
               duration, err = strconv.ParseInt(tmpl.Duration, 10, 64)
               must(err)
            }
            timescale := int64(1)
            if tmpl.Timescale != "" {
               timescale, err = strconv.ParseInt(tmpl.Timescale, 10, 64)
               must(err)
            }
            if duration == 0 || periodDur == 0 {
               continue
            }

            durTicks := int64(periodDur.Seconds() * float64(timescale))
            numSeg := (durTicks + duration - 1) / duration
            for i := int64(0); i < numSeg; i++ {
               segNum := startNum + i
               time := i * duration
               media := expandTemplate(tmpl.Media, rep.ID, segNum, time)
               mediaURL, err := url.Parse(media)
               must(err)
               out[rep.ID] = append(out[rep.ID], base.ResolveReference(mediaURL).String())
            }
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetEscapeHTML(false)
   must(enc.Encode(out))
}
