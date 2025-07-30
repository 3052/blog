package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "log"
   "math"
   "net/url"
   "os"
   "strconv"
   "strings"
) // removed regexp import per request

const defaultBase = "http://test.test/test.mpd"

// MPD structures
type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL *string  `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   XMLName        xml.Name        `xml:"Period"`
   DurationStr    string          `xml:"duration,attr"` // ISO8601 (e.g., "PT10S")
   BaseURL        *string         `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         *string          `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         *string          `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   Segments       []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   Duration        int64            `xml:"duration,attr"` // in timescale units
   Timescale       int64            `xml:"timescale,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   EndNumber       *int             `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T *int64 `xml:"t,attr"`
   D int64  `xml:"d,attr"`
   R *int   `xml:"r,attr"`
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }
   path := os.Args[1]

   data, err := ioutil.ReadFile(path)
   if err != nil {
      log.Fatalf("Failed to read MPD file: %v", err)
   }
   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      log.Fatalf("Failed to parse MPD XML: %v", err)
   }

   baseURL, err := url.Parse(defaultBase)
   if err != nil {
      log.Fatalf("Invalid default base URL: %v", err)
   }
   if mpd.BaseURL != nil {
      baseURL = resolveRef(baseURL, mpd.BaseURL)
   }

   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      secs := parseDuration(period.DurationStr)
      periodBase := resolveRef(baseURL, period.BaseURL)
      for _, adapt := range period.AdaptationSets {
         adaptBase := resolveRef(periodBase, adapt.BaseURL)
         for _, rep := range adapt.Representations {
            repBase := resolveRef(adaptBase, rep.BaseURL)

            tmpl := rep.SegmentTemplate
            if tmpl == nil {
               tmpl = adapt.SegmentTemplate
            }
            // default timescale to 1 if not specified
            if tmpl != nil && tmpl.Timescale == 0 {
               tmpl.Timescale = 1
            }

            sub := func(tpl string, num int, tm int64) string {
               out := ""
               for {

                  i := strings.Index(tpl, "$")
                  if i < 0 {
                     out += tpl
                     break
                  }
                  out += tpl[:i]
                  tpl = tpl[i+1:]
                  j := strings.Index(tpl, "$")
                  if j < 0 {
                     out += "$" + tpl
                     break
                  }
                  part := tpl[:j]
                  tpl = tpl[j+1:]
                  ka := strings.SplitN(part, "%", 2)
                  key := ka[0]
                  fmtSpec := ""
                  if len(ka) == 2 {
                     fmtSpec = "%" + ka[1]
                  }
                  var repStr string
                  switch key {
                  case "RepresentationID":
                     repStr = rep.ID
                  case "Number":
                     if fmtSpec != "" {
                        repStr = fmt.Sprintf(fmtSpec, num)
                     } else {
                        repStr = strconv.Itoa(num)
                     }
                  case "Time":
                     if fmtSpec != "" {
                        repStr = fmt.Sprintf(fmtSpec, tm)
                     } else {
                        repStr = strconv.FormatInt(tm, 10)
                     }
                  default:
                     repStr = "$" + part + "$"
                  }
                  out += repStr
               }
               return out
            }

            segs := []string{}
            if rep.SegmentList != nil {
               if rep.SegmentList.Initialization != nil {
                  segs = append(segs, resolveRef(repBase, &rep.SegmentList.Initialization.SourceURL).String())
               }
               for _, s := range rep.SegmentList.Segments {
                  segs = append(segs, resolveRef(repBase, &s.Media).String())
               }
            } else if tmpl != nil {
               if tmpl.Initialization != "" {
                  urlStr := sub(tmpl.Initialization, 0, 0)
                  segs = append(segs, resolveRef(repBase, &urlStr).String())
               }
               if tmpl.SegmentTimeline != nil {
                  cur := int64(0)
                  for _, s := range tmpl.SegmentTimeline.S {
                     cnt := 1
                     if s.R != nil {
                        cnt = *s.R + 1
                     }
                     if s.T != nil {
                        cur = *s.T
                     }
                     for i := 0; i < cnt; i++ {
                        urlStr := sub(tmpl.Media, tmplStartNumber(tmpl), cur)
                        segs = append(segs, resolveRef(repBase, &urlStr).String())
                        cur += s.D
                     }
                  }
               } else if tmpl.StartNumber != nil && tmpl.EndNumber != nil {
                  for n := *tmpl.StartNumber; n <= *tmpl.EndNumber; n++ {
                     urlStr := sub(tmpl.Media, n, 0)
                     segs = append(segs, resolveRef(repBase, &urlStr).String())
                  }
               } else if tmpl.SegmentTimeline == nil && tmpl.EndNumber == nil && tmpl.Duration > 0 && tmpl.Timescale > 0 {
                  count := int(math.Ceil(secs * float64(tmpl.Timescale) / float64(tmpl.Duration)))
                  for i := 0; i < count; i++ {
                     urlStr := sub(tmpl.Media, tmplStartNumber(tmpl)+i, 0)
                     segs = append(segs, resolveRef(repBase, &urlStr).String())
                  }
               }
            }

            if len(segs) == 0 {
               segs = append(segs, repBase.String())
            }
            result[rep.ID] = append(result[rep.ID], segs...)
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   if err := enc.Encode(result); err != nil {
      log.Fatalf("JSON encode error: %v", err)
   }
}

// resolveRef wraps url.Parse + ResolveReference
func resolveRef(base *url.URL, rel *string) *url.URL {
   if rel == nil {
      return base
   }
   ref, err := url.Parse(*rel)
   if err != nil {
      log.Fatalf("Invalid URL: %v", err)
   }
   return base.ResolveReference(ref)
}

// parseDuration converts ISO8601 PTnS to seconds
// parseDuration converts ISO8601 duration (e.g., PT2H13M19.040S) to seconds
// parseDuration converts ISO8601 duration (e.g., PT2H13M19.040S) to seconds
// parseDuration converts ISO8601 duration (e.g., PT2H13M19.040S) to seconds without regex
func parseDuration(s string) float64 {
   s = strings.TrimPrefix(s, "P")
   if idx := strings.Index(s, "T"); idx >= 0 {
      s = s[idx+1:]
   }
   var total float64
   // parse hours
   if hIdx := strings.Index(s, "H"); hIdx >= 0 {
      hVal := s[:hIdx]
      s = s[hIdx+1:]
      hours, _ := strconv.ParseFloat(hVal, 64)
      total += hours * 3600
   }
   // parse minutes
   if mIdx := strings.Index(s, "M"); mIdx >= 0 {
      mVal := s[:mIdx]
      s = s[mIdx+1:]
      mins, _ := strconv.ParseFloat(mVal, 64)
      total += mins * 60
   }
   // parse seconds
   if secIdx := strings.Index(s, "S"); secIdx >= 0 {
      secVal := s[:secIdx]
      secs, _ := strconv.ParseFloat(secVal, 64)
      total += secs
   }
   return total
}

// tmplStartNumber returns the startNumber or defaults to 1
func tmplStartNumber(t *SegmentTemplate) int {
   if t.StartNumber != nil {
      return *t.StartNumber
   }
   return 1
}
