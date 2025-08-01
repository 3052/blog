package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
)

const defaultBase = "http://test.test/test.mpd"

type MPD struct {
   XMLName         xml.Name         `xml:"MPD"`
   BaseURL         []string         `xml:"BaseURL"`
   Periods         []Period         `xml:"Period"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type Period struct {
   BaseURL         []string         `xml:"BaseURL"`
   Duration        string           `xml:"duration,attr"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   AdaptationSets  []AdaptationSet  `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentList struct {
   Initialization Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL   `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

type SegmentTemplate struct {
   Timescale       int64            `xml:"timescale,attr"`
   Duration        int64            `xml:"duration,attr"`
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     int64            `xml:"startNumber,attr"`
   EndNumber       *int64           `xml:"endNumber,attr"`
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

func main() {
   if len(os.Args) < 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }
   data, err := ioutil.ReadFile(os.Args[1])
   if err != nil {
      panic(err)
   }
   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      panic(err)
   }
   baseURL, err := url.Parse(defaultBase)
   if err != nil {
      panic(err)
   }
   result := make(map[string][]string)
   for _, period := range mpd.Periods {
      periodDur := parseDuration(period.Duration)
      for _, adap := range period.AdaptationSets {
         for _, rep := range adap.Representations {
            key := rep.ID
            curBase := baseURL
            curBase = applyFirstBase(curBase, mpd.BaseURL)
            curBase = applyFirstBase(curBase, period.BaseURL)
            curBase = applyFirstBase(curBase, adap.BaseURL)
            curBase = applyFirstBase(curBase, rep.BaseURL)

            // Only BaseURL => single entry
            if rep.SegmentList == nil && rep.SegmentTemplate == nil && len(rep.BaseURL) > 0 {
               result[key] = append(result[key], curBase.String())
               continue
            }

            tmpl := inheritTemplate(&rep, &adap, &period, &mpd)
            var urls []string
            if rep.SegmentList != nil {
               // Initialization segment
               u0 := substituteAndResolve(curBase, rep.SegmentList.Initialization.SourceURL, rep.ID, 0)
               urls = append(urls, u0)
               // Explicit segment URLs
               for _, seg := range rep.SegmentList.SegmentURLs {
                  u := substituteAndResolve(curBase, seg.Media, rep.ID, 0)
                  urls = append(urls, u)
               }
            } else if tmpl != nil {
               scale := tmpl.Timescale
               if scale == 0 {
                  scale = 1
               }
               start := tmpl.StartNumber
               if start == 0 {
                  start = 1
               }
               // Initialization from template
               if tmpl.Initialization != "" {
                  u := substituteAndResolve(curBase, tmpl.Initialization, rep.ID, 0)
                  urls = append(urls, u)
               }

               // SegmentTimeline present
               if tmpl.SegmentTimeline != nil {
                  var tcur int64
                  num := start
                  for i, seg := range tmpl.SegmentTimeline.S {
                     // repeat count
                     count := int64(1)
                     if seg.R != nil {
                        count = *seg.R + 1
                     }
                     // initial time
                     if i == 0 && seg.T != nil {
                        tcur = *seg.T
                     }
                     // enumerate
                     for j := int64(0); j < count; j++ {
                        var idx int64
                        if strings.Contains(tmpl.Media, "$Time$") {
                           idx = tcur
                           tcur += seg.D
                        } else {
                           idx = num
                           num++
                        }
                        u := substituteAndResolve(curBase, tmpl.Media, rep.ID, idx)
                        urls = append(urls, u)
                     }
                  }
               } else if tmpl.EndNumber != nil {
                  // numeric range
                  for n := start; n <= *tmpl.EndNumber; n++ {
                     u := substituteAndResolve(curBase, tmpl.Media, rep.ID, n)
                     urls = append(urls, u)
                  }
               } else if tmpl.Duration > 0 {
                  // computed count: ceil(periodDur * scale / duration)
                  durCount := int64(math.Ceil(periodDur * float64(scale) / float64(tmpl.Duration)))
                  for k := int64(0); k < durCount; k++ {
                     n := start + k
                     u := substituteAndResolve(curBase, tmpl.Media, rep.ID, n)
                     urls = append(urls, u)
                  }
               }
            }

            result[key] = append(result[key], urls...)
         }
      }
   }
   // output JSON
   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   if err := enc.Encode(result); err != nil {
      panic(err)
   }
}

func applyFirstBase(cur *url.URL, bases []string) *url.URL {
   if len(bases) == 0 {
      return cur
   }
   u, err := url.Parse(strings.TrimSpace(bases[0]))
   if err != nil {
      return cur
   }
   return cur.ResolveReference(u)
}

func inheritTemplate(rep *Representation, adap *AdaptationSet, per *Period, mpd *MPD) *SegmentTemplate {
   if rep.SegmentTemplate != nil {
      return rep.SegmentTemplate
   }
   if adap.SegmentTemplate != nil {
      return adap.SegmentTemplate
   }
   if per.SegmentTemplate != nil {
      return per.SegmentTemplate
   }
   return mpd.SegmentTemplate
}

func substituteAndResolve(base *url.URL, template, repID string, numberOrTime int64) string {
   s := strings.ReplaceAll(template, "$RepresentationID$", repID)

   // $Number$ with optional padding
   reNum := regexp.MustCompile(`\$Number(?:%0(\d+)d)?\$`)
   s = reNum.ReplaceAllStringFunc(s, func(m string) string {
      parts := reNum.FindStringSubmatch(m)
      if len(parts) == 2 && parts[1] != "" {
         w, _ := strconv.Atoi(parts[1])
         return fmt.Sprintf("%0*d", w, numberOrTime)
      }
      return strconv.FormatInt(numberOrTime, 10)
   })

   // $Time$
   if strings.Contains(s, "$Time$") {
      s = strings.ReplaceAll(s, "$Time$", strconv.FormatInt(numberOrTime, 10))
   }

   u, err := url.Parse(s)
   if err != nil {
      return s
   }
   return base.ResolveReference(u).String()
}

func parseDuration(d string) float64 {
   r := regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?$`)
   m := r.FindStringSubmatch(d)
   if m == nil {
      return 0
   }
   h := parseFloatOrZero(m[1])
   min := parseFloatOrZero(m[2])
   s := parseFloatOrZero(m[3])
   return h*3600 + min*60 + s
}

func parseFloatOrZero(s string) float64 {
   if s == "" {
      return 0
   }
   v, _ := strconv.ParseFloat(s, 64)
   return v
}

func (st *SegmentTemplate) SegmentTimelineOrDuration() int64 {
   if st.SegmentTimeline != nil && len(st.SegmentTimeline.S) > 0 {
      return st.SegmentTimeline.S[0].D
   }
   return 1
}
