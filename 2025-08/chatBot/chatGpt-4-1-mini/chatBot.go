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

   MediaBase string // initial base URL for resolving URLs
}

type Period struct {
   BaseURL        string          `xml:"BaseURL"`
   Duration       string          `xml:"duration,attr"` // ISO8601 duration string
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`

   DurationSeconds float64 `xml:"-"` // parsed duration in seconds
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentBase struct {
   Initialization *Initialization `xml:"Initialization"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   Duration        int              `xml:"duration,attr"`
   Timescale       int              `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T *int `xml:"t,attr"`
   D int  `xml:"d,attr"`
   R *int `xml:"r,attr"`
}

type SegmentInfo struct {
   Number    int
   StartTime int
}

var varFormatRe = regexp.MustCompile(`\$(Number|Time|RepresentationID)(%[^$]+)?\$`)

func formatTemplateVars(s, repID string, number, timeVal int) string {
   return varFormatRe.ReplaceAllStringFunc(s, func(m string) string {
      parts := varFormatRe.FindStringSubmatch(m)
      if len(parts) < 2 {
         return m
      }
      varName := parts[1]
      format := parts[2]
      switch varName {
      case "Number":
         if format == "" {
            return strconv.Itoa(number)
         }
         return fmt.Sprintf(format, number)
      case "Time":
         if format == "" {
            return strconv.Itoa(timeVal)
         }
         return fmt.Sprintf(format, timeVal)
      case "RepresentationID":
         if format == "" {
            return repID
         }
         return fmt.Sprintf(format, repID)
      default:
         return m
      }
   })
}

func resolveURL(baseURL, refURL string) (string, error) {
   base, err := url.Parse(baseURL)
   if err != nil {
      return "", err
   }
   ref, err := url.Parse(refURL)
   if err != nil {
      return "", err
   }
   resolved := base.ResolveReference(ref)
   return resolved.String(), nil
}

// parseDurationISO8601 parses a subset of ISO8601 durations to seconds (float64)
func parseDurationISO8601(s string) (float64, error) {
   re := regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?$`)
   matches := re.FindStringSubmatch(s)
   if matches == nil {
      return 0, fmt.Errorf("unsupported ISO8601 duration format: %s", s)
   }
   var hours, minutes int
   var seconds float64
   var err error

   if matches[1] != "" {
      hours, err = strconv.Atoi(matches[1])
      if err != nil {
         return 0, err
      }
   }
   if matches[2] != "" {
      minutes, err = strconv.Atoi(matches[2])
      if err != nil {
         return 0, err
      }
   }
   if matches[3] != "" {
      seconds, err = strconv.ParseFloat(matches[3], 64)
      if err != nil {
         return 0, err
      }
   }

   total := float64(hours)*3600 + float64(minutes)*60 + seconds
   return total, nil
}

func generateSegments(tmpl *SegmentTemplate, periodDurationSeconds float64) []SegmentInfo {
   var segments []SegmentInfo
   startNum := tmpl.StartNumber
   if startNum == 0 {
      startNum = 1
   }
   timescale := tmpl.Timescale
   if timescale == 0 {
      timescale = 1
   }

   if tmpl.SegmentTimeline != nil {
      num := startNum
      var currentTime int
      for _, seg := range tmpl.SegmentTimeline.S {
         repeat := 0
         if seg.R != nil {
            repeat = *seg.R
         }
         if seg.T != nil {
            currentTime = *seg.T
         }
         for i := 0; i <= repeat; i++ {
            segments = append(segments, SegmentInfo{Number: num, StartTime: currentTime})
            currentTime += seg.D
            num++
         }
      }
   } else if tmpl.EndNumber > 0 && tmpl.EndNumber >= startNum {
      currentTime := 0
      for num := startNum; num <= tmpl.EndNumber; num++ {
         segments = append(segments, SegmentInfo{Number: num, StartTime: currentTime})
         currentTime += tmpl.Duration
      }
   } else if tmpl.Duration > 0 && timescale > 0 && periodDurationSeconds > 0 {
      segmentCount := int(math.Ceil(periodDurationSeconds * float64(timescale) / float64(tmpl.Duration)))
      currentTime := 0
      for num := startNum; num < startNum+segmentCount; num++ {
         segments = append(segments, SegmentInfo{Number: num, StartTime: currentTime})
         currentTime += tmpl.Duration
      }
   } else if tmpl.Duration > 0 {
      startTime := (startNum - 1) * tmpl.Duration
      segments = append(segments, SegmentInfo{Number: startNum, StartTime: startTime})
   } else {
      segments = append(segments, SegmentInfo{Number: startNum, StartTime: 0})
   }

   return segments
}

func collectSegmentURLs(baseURL string, adapTemplate *SegmentTemplate, rep Representation, periodDurationSeconds float64) ([]string, error) {
   var urls []string

   if rep.SegmentBase != nil && rep.SegmentBase.Initialization != nil && rep.SegmentBase.Initialization.SourceURL != "" {
      u, err := resolveURL(baseURL, rep.SegmentBase.Initialization.SourceURL)
      if err != nil {
         return nil, err
      }
      urls = append(urls, u)
   }

   if rep.SegmentList != nil {
      if rep.SegmentList.Initialization != nil && rep.SegmentList.Initialization.SourceURL != "" {
         u, err := resolveURL(baseURL, rep.SegmentList.Initialization.SourceURL)
         if err != nil {
            return nil, err
         }
         urls = append(urls, u)
      }
      for _, seg := range rep.SegmentList.SegmentURLs {
         u, err := resolveURL(baseURL, seg.Media)
         if err != nil {
            return nil, err
         }
         urls = append(urls, u)
      }
   }

   if len(urls) > 0 {
      return urls, nil
   }

   var tmpl *SegmentTemplate
   if rep.SegmentTemplate != nil {
      tmpl = rep.SegmentTemplate
   } else {
      tmpl = adapTemplate
   }

   if tmpl != nil {
      if tmpl.Initialization != "" {
         u, err := resolveURL(baseURL, tmpl.Initialization)
         if err != nil {
            return nil, err
         }
         urls = append(urls, u)
      }

      segments := generateSegments(tmpl, periodDurationSeconds)

      for _, seg := range segments {
         if tmpl.Media != "" {
            mediaURL := formatTemplateVars(tmpl.Media, rep.ID, seg.Number, seg.StartTime)
            u, err := resolveURL(baseURL, mediaURL)
            if err != nil {
               return nil, err
            }
            urls = append(urls, u)
         }
      }

      if len(urls) > 0 {
         return urls, nil
      }
   }

   if baseURL != "" {
      return []string{baseURL}, nil
   }

   return nil, fmt.Errorf("no segment info or base URL found for representation %s", rep.ID)
}

func parseMPD(mpd *MPD) (map[string][]string, error) {
   result := make(map[string][]string)

   for i := range mpd.Periods {
      p := &mpd.Periods[i]

      periodBase := mpd.MediaBase
      if mpd.BaseURL != "" {
         b, err := resolveURL(mpd.MediaBase, strings.TrimSpace(mpd.BaseURL))
         if err == nil {
            periodBase = b
         }
      }
      if p.BaseURL != "" {
         b, err := resolveURL(periodBase, p.BaseURL)
         if err != nil {
            return nil, err
         }
         periodBase = b
      }

      if p.Duration != "" {
         dur, err := parseDurationISO8601(p.Duration)
         if err != nil {
            return nil, fmt.Errorf("failed to parse Period duration '%s': %v", p.Duration, err)
         }
         p.DurationSeconds = dur
      } else {
         p.DurationSeconds = 0
      }

      for _, adapSet := range p.AdaptationSets {
         adapBase := periodBase
         if adapSet.BaseURL != "" {
            b, err := resolveURL(adapBase, adapSet.BaseURL)
            if err != nil {
               return nil, err
            }
            adapBase = b
         }

         for _, rep := range adapSet.Representations {
            repBase := adapBase
            if rep.BaseURL != "" {
               b, err := resolveURL(repBase, rep.BaseURL)
               if err != nil {
                  return nil, err
               }
               repBase = b
            }

            urls, err := collectSegmentURLs(repBase, adapSet.SegmentTemplate, rep, p.DurationSeconds)
            if err != nil {
               return nil, err
            }

            if rep.ID != "" && len(urls) > 0 {
               result[rep.ID] = append(result[rep.ID], urls...)
            }
         }
      }
   }

   return result, nil
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }
   mpdFile := os.Args[1]

   data, err := os.ReadFile(mpdFile)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Failed to read MPD file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   err = xml.Unmarshal(data, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Failed to parse MPD XML: %v\n", err)
      os.Exit(1)
   }

   mpd.MediaBase = "http://test.test/test.mpd"

   result, err := parseMPD(&mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD: %v\n", err)
      os.Exit(1)
   }

   enc := json.NewEncoder(os.Stdout)
   enc.SetIndent("", "  ")
   err = enc.Encode(result)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Failed to encode JSON output: %v\n", err)
      os.Exit(1)
   }
}

