package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "log"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
)

// -----------------------------------------------------------
// XML data structures (exactly mirror the DASH MPD schema)
// -----------------------------------------------------------

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL []string `xml:"BaseURL"`
   Period  []Period `xml:"Period"`
}

type Period struct {
   BaseURL       []string        `xml:"BaseURL"`
   Duration      string          `xml:"duration,attr"`
   AdaptationSet []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         []string         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representation  []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   Bandwidth       uint64           `xml:"bandwidth,attr"`
   BaseURL         []string         `xml:"BaseURL"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentBase struct {
   Initialization string           `xml:"initialization,attr"`
   InitElement    *SegmentBaseInit `xml:"Initialization"`
}
type SegmentBaseInit struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   Timescale       uint64           `xml:"timescale,attr"`
   Duration        uint64           `xml:"duration,attr"`
   StartNumber     *uint64          `xml:"startNumber,attr"` // <- pointer to detect missing attr
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
   EndNumber       uint64           `xml:"endNumber,attr,omitempty"`
}

type SegmentTimeline struct {
   S []SegmentTimelineS `xml:"S"`
}
type SegmentTimelineS struct {
   T *uint64 `xml:"t,attr,omitempty"`
   D uint64  `xml:"d,attr"`
   R *int64  `xml:"r,attr,omitempty"`
}

type SegmentList struct {
   SegmentURL     []SegmentURL               `xml:"SegmentURL"`
   Initialization *SegmentListInitialization `xml:"Initialization"`
}
type SegmentListInitialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

// -----------------------------------------------------------
// Placeholder handling
// -----------------------------------------------------------

var placeholderRe = regexp.MustCompile(`\$([A-Za-z]+)(%[0-9]+[sd])?\$`)

// applyPlaceholders applies $Number$, $RepresentationID$, $Bandwidth$ and
// $Time$ placeholders, optionally respecting a printf‑style format.
func applyPlaceholders(template string, segNum int, segTime uint64, rep *Representation) string {
   return placeholderRe.ReplaceAllStringFunc(template, func(match string) string {
      m := placeholderRe.FindStringSubmatch(match)
      if len(m) < 2 {
         return match
      }
      name := m[1]
      fmtPart := ""
      if len(m) > 2 {
         fmtPart = m[2]
      }

      var val interface{}
      numeric := false

      switch name {
      case "Number":
         val = segNum
         numeric = true
      case "RepresentationID":
         val = rep.ID
      case "Bandwidth":
         val = rep.Bandwidth
         numeric = true
      case "Time":
         val = segTime
         numeric = true
      default:
         return match
      }

      if numeric && fmtPart != "" {
         return fmt.Sprintf(fmtPart, val)
      }
      return fmt.Sprintf("%v", val)
   })
}

func replacePlaceholdersInURL(u string, segNum int, segTime uint64, rep *Representation) string {
   return applyPlaceholders(u, segNum, segTime, rep)
}

// -----------------------------------------------------------
// URL resolution helpers
// -----------------------------------------------------------

func resolveURL(base *url.URL, relStr string) (*url.URL, error) {
   relStr = strings.TrimSpace(relStr)
   if relStr == "" {
      return nil, fmt.Errorf("empty relative URL")
   }
   rel, err := url.Parse(relStr)
   if err != nil {
      return nil, err
   }
   return base.ResolveReference(rel), nil
}

func resolveBaseURL(parent *url.URL, relStr string) (*url.URL, error) {
   relStr = strings.TrimSpace(relStr)
   if relStr == "" {
      return parent, nil
   }
   rel, err := url.Parse(relStr)
   if err != nil {
      return nil, err
   }
   return parent.ResolveReference(rel), nil
}

// -----------------------------------------------------------
// Timeline helper
// -----------------------------------------------------------

func computeTimes(st *SegmentTimeline) []uint64 {
   segTimes := []uint64{}
   curTime := uint64(0)
   for _, entry := range st.S {
      baseTime := curTime
      if entry.T != nil {
         baseTime = *entry.T
      }
      repeats := int64(0)
      if entry.R != nil {
         repeats = *entry.R
         if repeats < 0 {
            repeats = 0
         }
      }
      for i := int64(0); i <= repeats; i++ {
         segTimes = append(segTimes, baseTime)
         baseTime += entry.D
      }
      curTime = baseTime
   }
   return segTimes
}

// -----------------------------------------------------------
// ISO‑8601 duration parser (used for period duration)
// -----------------------------------------------------------

func parseISODurationToSeconds(s string) (float64, error) {
   if !strings.HasPrefix(s, "PT") {
      return 0, fmt.Errorf("invalid ISO 8601 duration: %s", s)
   }
   s = s[2:]
   var hours, minutes, seconds float64
   re := regexp.MustCompile(`(\d+(?:\.\d+)?)([HMS])`)
   matches := re.FindAllStringSubmatch(s, -1)
   for _, m := range matches {
      value, _ := strconv.ParseFloat(m[1], 64)
      switch m[2] {
      case "H":
         hours = value
      case "M":
         minutes = value
      case "S":
         seconds = value
      }
   }
   return hours*3600 + minutes*60 + seconds, nil
}

// -----------------------------------------------------------
// Main entry point
// -----------------------------------------------------------

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }
   mpdPath := os.Args[1]

   mpdBytes, err := os.ReadFile(mpdPath)
   if err != nil {
      log.Fatalf("Failed to read MPD file: %v", err)
   }

   var mpd MPD
   if err := xml.Unmarshal(mpdBytes, &mpd); err != nil {
      log.Fatalf("Failed to parse MPD XML: %v", err)
   }

   // Base URL from which all relative URLs are resolved.
   baseURL, err := url.Parse("http://test.test/test.mpd")
   if err != nil {
      log.Fatalf("Invalid initial MPD URL: %v", err)
   }
   mpdBase := baseURL
   if len(mpd.BaseURL) > 0 {
      mpdBaseStr := replacePlaceholdersInURL(mpd.BaseURL[0], 0, 0, nil)
      mpdBase, err = resolveBaseURL(mpdBase, mpdBaseStr)
      if err != nil {
         log.Fatalf("Error resolving MPD BaseURL: %v", err)
      }
   }

   // Result map : Representation ID → list of resolved segment URLs
   result := make(map[string][]string)

   // ----------------------------------------------------------------
   // Walk Period → AdaptationSet → Representation
   // ----------------------------------------------------------------
   for _, period := range mpd.Period {
      periodBase := mpdBase
      if len(period.BaseURL) > 0 {
         pBaseStr := replacePlaceholdersInURL(period.BaseURL[0], 0, 0, nil)
         periodBase, err = resolveBaseURL(periodBase, pBaseStr)
         if err != nil {
            log.Fatalf("Error resolving Period BaseURL: %v", err)
         }
      }

      // Period duration (in seconds) – used only when we need to
      // calculate the number of segments from duration+timescale.
      periodSec := 0.0
      if period.Duration != "" {
         if ps, err := parseISODurationToSeconds(period.Duration); err == nil {
            periodSec = ps
         } else {
            log.Printf("Failed to parse period duration %q: %v", period.Duration, err)
         }
      }

      for _, as := range period.AdaptationSet {
         asBase := periodBase
         if len(as.BaseURL) > 0 {
            asBaseStr := replacePlaceholdersInURL(as.BaseURL[0], 0, 0, nil)
            asBase, err = resolveBaseURL(asBase, asBaseStr)
            if err != nil {
               log.Fatalf("Error resolving AdaptationSet BaseURL: %v", err)
            }
         }

         for _, rep := range as.Representation {
            repBase := asBase
            if len(rep.BaseURL) > 0 {
               rBaseStr := replacePlaceholdersInURL(rep.BaseURL[0], 0, 0, &rep)
               repBase, err = resolveBaseURL(repBase, rBaseStr)
               if err != nil {
                  log.Fatalf("Error resolving Representation BaseURL: %v", err)
               }
            }

            segmentURLs := []string{}

            // ---------------------------------------------------
            // 1. Initialization from SegmentBase
            // ---------------------------------------------------
            if rep.SegmentBase != nil {
               initURLStr := replacePlaceholdersInURL(rep.SegmentBase.Initialization, 0, 0, &rep)
               if initURLStr == "" && rep.SegmentBase.InitElement != nil {
                  initURLStr = replacePlaceholdersInURL(rep.SegmentBase.InitElement.SourceURL, 0, 0, &rep)
               }
               if initURLStr != "" {
                  if initURL, err := resolveURL(repBase, initURLStr); err == nil {
                     segmentURLs = append(segmentURLs, initURL.String())
                  } else {
                     log.Printf("Failed to resolve SegmentBase initialization: %v", err)
                  }
               }
            }

            // ---------------------------------------------------
            // 2. Determine applicable SegmentTemplate
            // ---------------------------------------------------
            var segTemplate *SegmentTemplate
            if rep.SegmentTemplate != nil {
               segTemplate = rep.SegmentTemplate
            } else if as.SegmentTemplate != nil {
               segTemplate = as.SegmentTemplate
            }

            // ---------------------------------------------------
            // 3. Compute segment times / number of segments
            // ---------------------------------------------------
            var startNum int
            var startNumUint uint64
            segTimes := []uint64{}
            if segTemplate != nil {
               // Default timescale to 1 when missing.
               if segTemplate.Timescale == 0 {
                  segTemplate.Timescale = 1
               }

               if segTemplate.StartNumber != nil {
                  startNumUint = *segTemplate.StartNumber
                  startNum = int(startNumUint)
               } else {
                  // Missing attribute → default to 1
                  startNumUint = 1
                  startNum = 1
               }

               if segTemplate.SegmentTimeline != nil && len(segTemplate.SegmentTimeline.S) > 0 {
                  segTimes = computeTimes(segTemplate.SegmentTimeline)
               } else if segTemplate.EndNumber > 0 && segTemplate.EndNumber >= startNumUint {
                  count := int(segTemplate.EndNumber - startNumUint + 1)
                  segTimes = make([]uint64, count)
                  if segTemplate.Duration > 0 {
                     for i := 0; i < count; i++ {
                        segTimes[i] = uint64(i) * segTemplate.Duration
                     }
                  }
               } else if segTemplate.Duration > 0 && periodSec > 0 {
                  numSeg := int(math.Ceil(periodSec * float64(segTemplate.Timescale) /
                     float64(segTemplate.Duration)))
                  segTimes = make([]uint64, numSeg)
                  for i := 0; i < numSeg; i++ {
                     segTimes[i] = uint64(i) * segTemplate.Duration
                  }
               } else {
                  segTimes = []uint64{0}
               }
            }

            // ---------------------------------------------------
            // 4. Initialization (via SegmentTemplate)
            // ---------------------------------------------------
            if segTemplate != nil && segTemplate.Initialization != "" {
               initTime := uint64(0)
               if len(segTimes) > 0 {
                  initTime = segTimes[0]
               }
               initURLStr := replacePlaceholdersInURL(segTemplate.Initialization, 0, initTime, &rep)
               if initURL, err := resolveURL(repBase, initURLStr); err == nil {
                  segmentURLs = append(segmentURLs, initURL.String())
               } else {
                  log.Printf("Failed to resolve SegmentTemplate initialization: %v", err)
               }
            }

            // ---------------------------------------------------
            // 5. Media segment URLs (via SegmentTemplate)
            // ---------------------------------------------------
            if segTemplate != nil && segTemplate.Media != "" && len(segTimes) > 0 {
               for i, segTime := range segTimes {
                  segNum := startNum + i
                  relSegURL := replacePlaceholdersInURL(segTemplate.Media, segNum, segTime, &rep)
                  if segURL, err := resolveURL(repBase, relSegURL); err == nil {
                     segmentURLs = append(segmentURLs, segURL.String())
                  } else {
                     log.Printf("Failed to resolve SegmentTemplate media URL for segment %d: %v", segNum, err)
                  }
               }
            }

            // ---------------------------------------------------
            // 6. Explicit SegmentList handling
            // ---------------------------------------------------
            if rep.SegmentList != nil {
               // Initialization via <SegmentList><Initialization sourceURL="…"/>
               if rep.SegmentList.Initialization != nil && rep.SegmentList.Initialization.SourceURL != "" {
                  initURLStr := replacePlaceholdersInURL(rep.SegmentList.Initialization.SourceURL, 0, 0, &rep)
                  if initURL, err := resolveURL(repBase, initURLStr); err == nil {
                     // Avoid duplication of an init already added above
                     if len(segmentURLs) == 0 || segmentURLs[0] != initURL.String() {
                        segmentURLs = append(segmentURLs, initURL.String())
                     }
                  } else {
                     log.Printf("Failed to resolve SegmentList initialization: %v", err)
                  }
               }
               // Individual <SegmentURL> entries
               for idx, su := range rep.SegmentList.SegmentURL {
                  if su.Media != "" {
                     mediaURLStr := replacePlaceholdersInURL(su.Media, idx+1, 0, &rep)
                     if segURL, err := resolveURL(repBase, mediaURLStr); err == nil {
                        segmentURLs = append(segmentURLs, segURL.String())
                     } else {
                        log.Printf("Failed to resolve SegmentList SegmentURL media: %v", err)
                     }
                  }
               }
            }

            // ---------------------------------------------------
            // 7. Fallback when the representation has only BaseURL
            // ---------------------------------------------------
            if len(segmentURLs) == 0 {
               segmentURLs = []string{repBase.String()}
            }

            // ---------------------------------------------------
            // 8. Merge URLs for representations that span multiple periods
            // ---------------------------------------------------
            if existing, ok := result[rep.ID]; ok {
               result[rep.ID] = append(existing, segmentURLs...)
            } else {
               result[rep.ID] = segmentURLs
            }
         }
      }
   }

   // ----------------------------------------------------------------
   // Output the final map as pretty‑printed JSON
   // ----------------------------------------------------------------
   if out, err := json.MarshalIndent(result, "", "  "); err == nil {
      fmt.Println(string(out))
   } else {
      log.Fatalf("Failed to marshal JSON: %v", err)
   }
}
