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
   BaseURL *string  `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL        *string         `xml:"BaseURL"`
   Duration       string          `xml:"duration,attr"` // ISO8601 duration string
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         *string          `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         *string          `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   Timescale          uint64           `xml:"timescale,attr"`
   InitializationAttr string           `xml:"initialization,attr"`
   Initialization     *InitSegment     `xml:"Initialization"`
   Media              string           `xml:"media,attr"`
   Duration           uint64           `xml:"duration,attr"` // segment duration in timescale units
   StartNumber        *int64           `xml:"startNumber,attr"`
   EndNumber          *int64           `xml:"endNumber,attr"`
   SegmentTimeline    *SegmentTimeline `xml:"SegmentTimeline"`
}

type InitSegment struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T *uint64 `xml:"t,attr"`
   D uint64  `xml:"d,attr"`
   R *int64  `xml:"r,attr"`
}

type SegmentList struct {
   Timescale      uint64          `xml:"timescale,attr"`
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: go run main.go <mpd_file_path>\n")
      os.Exit(1)
   }
   mpdPath := os.Args[1]

   f, err := os.Open(mpdPath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error opening MPD file: %v\n", err)
      os.Exit(1)
   }
   defer f.Close()

   decoder := xml.NewDecoder(f)
   var mpd MPD
   if err := decoder.Decode(&mpd); err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD XML: %v\n", err)
      os.Exit(1)
   }

   baseMPDURL := "http://test.test/test.mpd"

   result := make(map[string][]string)

   mpdBaseURL := resolveBaseURL(baseMPDURL, mpd.BaseURL)

   for _, period := range mpd.Periods {
      periodBaseURL := resolveBaseURL(mpdBaseURL, period.BaseURL)

      periodDurationSeconds := -1.0
      if period.Duration != "" {
         if dur, err := parseDurationISO8601(period.Duration); err == nil {
            periodDurationSeconds = dur
         }
      }

      for _, adSet := range period.AdaptationSets {
         adSetBaseURL := resolveBaseURL(periodBaseURL, adSet.BaseURL)
         for _, rep := range adSet.Representations {
            repBaseURL := resolveBaseURL(adSetBaseURL, rep.BaseURL)

            segTemplate := inheritSegmentTemplate(rep.SegmentTemplate, adSet.SegmentTemplate)
            segList := inheritSegmentList(rep.SegmentList, adSet.SegmentList)

            timescale := uint64(1)
            if segTemplate != nil && segTemplate.Timescale != 0 {
               timescale = segTemplate.Timescale
            } else if segList != nil && segList.Timescale != 0 {
               timescale = segList.Timescale
            }

            var segments []string

            // Initialization segment (prefer initialization attribute over element)
            initSegment := ""
            if segTemplate != nil {
               if segTemplate.InitializationAttr != "" {
                  initSegment = resolveURLTemplate(repBaseURL, segTemplate.InitializationAttr, rep.ID, 0, 0)
               } else if segTemplate.Initialization != nil {
                  initSegment = resolveURLTemplate(repBaseURL, segTemplate.Initialization.SourceURL, rep.ID, 0, 0)
               }
            }
            if initSegment == "" && segList != nil && segList.Initialization != nil {
               initSegment = resolveURL(repBaseURL, segList.Initialization.SourceURL)
            }
            if initSegment != "" {
               segments = append(segments, initSegment)
            }

            if segList != nil {
               // Use SegmentList segment URLs
               for _, segURL := range segList.SegmentURLs {
                  fullURL := resolveURL(repBaseURL, segURL.Media)
                  segments = append(segments, fullURL)
               }
            } else if segTemplate != nil {
               startNumber := int64(1)
               if segTemplate.StartNumber != nil {
                  startNumber = *segTemplate.StartNumber
               }
               endNumber := int64(-1)
               if segTemplate.EndNumber != nil {
                  endNumber = *segTemplate.EndNumber
               }

               if segTemplate.SegmentTimeline != nil && len(segTemplate.SegmentTimeline.S) > 0 {
                  segmentsTimeline := buildSegmentsFromTimeline(repBaseURL, segTemplate, rep.ID, segTemplate.SegmentTimeline.S, timescale)
                  segments = append(segments, segmentsTimeline...)
               } else {
                  var count int
                  if endNumber != -1 && endNumber >= startNumber {
                     count = int(endNumber - startNumber + 1)
                  } else if segTemplate.Duration != 0 && periodDurationSeconds > 0 {
                     count = int(math.Ceil(periodDurationSeconds * float64(timescale) / float64(segTemplate.Duration)))
                  } else {
                     count = 10
                  }

                  for i := 0; i < count; i++ {
                     num := startNumber + int64(i)
                     urlStr := resolveURLTemplate(repBaseURL, segTemplate.Media, rep.ID, num, 0)
                     segments = append(segments, urlStr)
                  }
               }
            } else {
               // Fallback: use Representation BaseURL as single segment URL
               segments = append(segments, repBaseURL)
            }

            // Append segments to the existing slice for this Representation ID
            result[rep.ID] = append(result[rep.ID], segments...)
         }
      }
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   if err := enc.Encode(result); err != nil {
      fmt.Fprintf(os.Stderr, "Error encoding JSON output: %v\n", err)
      os.Exit(1)
   }
}

// resolveBaseURL resolves currentBase against relativeBase (if any)
func resolveBaseURL(currentBase string, relativeBase *string) string {
   if relativeBase == nil {
      return currentBase
   }
   rel := strings.TrimSpace(*relativeBase)
   if rel == "" {
      return currentBase
   }
   baseURL, err := url.Parse(currentBase)
   if err != nil {
      return rel
   }
   refURL, err := url.Parse(rel)
   if err != nil {
      return rel
   }
   return baseURL.ResolveReference(refURL).String()
}

func inheritSegmentTemplate(rep, adSet *SegmentTemplate) *SegmentTemplate {
   if rep != nil {
      if rep.Media == "" && adSet != nil {
         rep.Media = adSet.Media
      }
      if rep.InitializationAttr == "" && adSet != nil {
         rep.InitializationAttr = adSet.InitializationAttr
      }
      if rep.Initialization == nil && adSet != nil {
         rep.Initialization = adSet.Initialization
      }
      if rep.StartNumber == nil && adSet != nil {
         rep.StartNumber = adSet.StartNumber
      }
      if rep.EndNumber == nil && adSet != nil {
         rep.EndNumber = adSet.EndNumber
      }
      if rep.Timescale == 0 && adSet != nil {
         rep.Timescale = adSet.Timescale
      }
      if rep.SegmentTimeline == nil && adSet != nil {
         rep.SegmentTimeline = adSet.SegmentTimeline
      }
      if rep.Duration == 0 && adSet != nil {
         rep.Duration = adSet.Duration
      }
      return rep
   }
   return adSet
}

func inheritSegmentList(rep, adSet *SegmentList) *SegmentList {
   if rep != nil {
      return rep
   }
   return adSet
}

func resolveURL(base, rel string) string {
   rel = strings.TrimSpace(rel)
   if rel == "" {
      return base
   }
   baseURL, err := url.Parse(base)
   if err != nil {
      return rel
   }
   refURL, err := url.Parse(rel)
   if err != nil {
      return rel
   }
   return baseURL.ResolveReference(refURL).String()
}

var substitutionRegexp = regexp.MustCompile(`\$(\w+)(%0(\d+)d)?\$`)

// resolveURLTemplate substitutes $RepresentationID$, $Number$, $Time$ with optional printf formatting
func resolveURLTemplate(baseURL, template, repID string, number int64, time uint64) string {
   if template == "" {
      return baseURL
   }

   result := substitutionRegexp.ReplaceAllStringFunc(template, func(m string) string {
      matches := substitutionRegexp.FindStringSubmatch(m)
      if len(matches) < 2 {
         return m
      }
      varName := matches[1]
      format := ""
      if len(matches) >= 4 && matches[3] != "" {
         format = "%0" + matches[3] + "d"
      }
      switch varName {
      case "RepresentationID":
         return repID
      case "Number":
         if format != "" {
            return fmt.Sprintf(format, number)
         }
         return strconv.FormatInt(number, 10)
      case "Time":
         if format != "" {
            return fmt.Sprintf(format, time)
         }
         return strconv.FormatUint(time, 10)
      default:
         return m
      }
   })

   return resolveURL(baseURL, result)
}

// buildSegmentsFromTimeline expands SegmentTimeline into URLs
func buildSegmentsFromTimeline(baseURL string, segTemplate *SegmentTemplate, repID string, timeline []S, timescale uint64) []string {
   var segments []string
   var currentTime uint64

   startNumber := int64(1)
   if segTemplate.StartNumber != nil {
      startNumber = *segTemplate.StartNumber
   }

   segmentIndex := startNumber

   for i, s := range timeline {
      r := int64(0)
      if s.R != nil {
         r = *s.R
      }
      if r == -1 {
         r = 10 // limit infinite repeats to 10
      }

      if s.T != nil {
         currentTime = *s.T
      } else if i == 0 {
         currentTime = 0
      }

      repeatCount := r + 1
      for j := int64(0); j < repeatCount; j++ {
         urlStr := resolveURLTemplate(baseURL, segTemplate.Media, repID, segmentIndex, currentTime)
         segments = append(segments, urlStr)

         segmentIndex++
         currentTime += s.D
      }
   }

   return segments
}

var iso8601durationRegex = regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?`)

// parseDurationISO8601 parses simple ISO8601 durations like PT30S, PT1H2M3.5S to seconds float64
func parseDurationISO8601(s string) (float64, error) {
   m := iso8601durationRegex.FindStringSubmatch(s)
   if m == nil {
      return 0, fmt.Errorf("invalid ISO 8601 duration: %s", s)
   }
   var h, mnt int
   var sec float64
   if m[1] != "" {
      h, _ = strconv.Atoi(m[1])
   }
   if m[2] != "" {
      mnt, _ = strconv.Atoi(m[2])
   }
   if m[3] != "" {
      sec, _ = strconv.ParseFloat(m[3], 64)
   }
   return float64(h)*3600 + float64(mnt)*60 + sec, nil
}
