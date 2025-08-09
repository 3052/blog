package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
)

// DASH MPD structures
type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"` // Can be relative or absolute
   Periods []Period `xml:"Period"`
}

type Period struct {
   ID             string          `xml:"id,attr"`
   Duration       string          `xml:"duration,attr"`
   BaseURL        string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   ID              string           `xml:"id,attr"`
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
   Initialization *Initialization `xml:"Initialization"`
   SegmentURL     []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

type SegmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     uint64           `xml:"startNumber,attr"`
   EndNumber       uint64           `xml:"endNumber,attr"`
   Duration        uint64           `xml:"duration,attr"`
   Timescale       uint64           `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []SegmentTimelineEntry `xml:"S"`
}

type SegmentTimelineEntry struct {
   T uint64 `xml:"t,attr,omitempty"`
   D uint64 `xml:"d,attr"`
   R uint64 `xml:"r,attr,omitempty"`
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }

   mpdPath := os.Args[1]
   file, err := os.Open(mpdPath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
      os.Exit(1)
   }
   defer file.Close()

   content, err := io.ReadAll(file)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
      os.Exit(1)
   }

   // The effective URL of the MPD document (used to resolve relative BaseURLs)
   mpdDocumentURL, err := url.Parse("http://test.test/test.mpd")
   if err != nil {
      panic("invalid base MPD URL")
   }

   var mpd MPD
   if err := xml.Unmarshal(content, &mpd); err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD XML: %v\n", err)
      os.Exit(1)
   }

   // Resolve MPD's own BaseURL relative to the MPD document URL
   baseURL := resolveURL(mpd.BaseURL, mpdDocumentURL).String()
   if baseURL == "" {
      baseURL = mpdDocumentURL.String() // fallback
   }

   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      // Resolve Period BaseURL relative to current base
      periodBase := resolveURL(period.BaseURL, mustParseURL(baseURL)).String()
      if periodBase == "" {
         periodBase = baseURL
      }

      periodSeconds := parsePeriodDuration(period.Duration)

      for _, as := range period.AdaptationSets {
         asBase := resolveURL(as.BaseURL, mustParseURL(periodBase)).String()
         if asBase == "" {
            asBase = periodBase
         }

         asTemplate := as.SegmentTemplate

         for _, rep := range as.Representations {
            repBase := resolveURL(rep.BaseURL, mustParseURL(asBase)).String()
            if repBase == "" {
               repBase = asBase
            }

            segments := resolveSegments(rep, asTemplate, repBase, periodSeconds)
            result[rep.ID] = append(result[rep.ID], segments...)
         }
      }
   }

   jsonData, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error generating JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonData))
}

// resolveURL resolves a possibly relative URL against the base
func resolveURL(raw string, base *url.URL) *url.URL {
   if raw == "" {
      return base
   }
   u, err := url.Parse(raw)
   if err != nil {
      return base
   }
   return base.ResolveReference(u)
}

// mustParseURL is a helper for constants
func mustParseURL(raw string) *url.URL {
   u, err := url.Parse(raw)
   if err != nil {
      panic(err)
   }
   return u
}

var numberFormatRegex = regexp.MustCompile(`\$Number%0(\d+)d\$`)

func resolveSegments(rep Representation, asTemplate *SegmentTemplate, baseURL string, periodSeconds float64) []string {
   var segments []string
   base, _ := url.Parse(baseURL)

   // Case 1: SegmentList
   if rep.SegmentList != nil {
      if rep.SegmentList.Initialization != nil && rep.SegmentList.Initialization.SourceURL != "" {
         u, err := url.Parse(rep.SegmentList.Initialization.SourceURL)
         if err == nil {
            segments = append(segments, base.ResolveReference(u).String())
         }
      }
      for _, seg := range rep.SegmentList.SegmentURL {
         if seg.Media == "" {
            continue
         }
         u, err := url.Parse(seg.Media)
         if err != nil {
            continue
         }
         segments = append(segments, base.ResolveReference(u).String())
      }
      return segments
   }

   // Case 2: SegmentTemplate
   template := rep.SegmentTemplate
   if template == nil {
      template = asTemplate
   }

   if template == nil {
      if len(segments) == 0 {
         segments = append(segments, baseURL)
      }
      return segments
   }

   // Default startNumber to 1
   startNumber := template.StartNumber
   if startNumber == 0 {
      startNumber = 1
   }

   // Add initialization
   if template.Initialization != "" {
      initPath := template.Initialization
      initPath = strings.ReplaceAll(initPath, "$RepresentationID$", rep.ID)
      initPath = strings.ReplaceAll(initPath, "$Bandwidth$", rep.ID)
      u, err := url.Parse(initPath)
      if err == nil {
         segments = append(segments, base.ResolveReference(u).String())
      }
   }

   if template.Media == "" {
      if len(segments) == 0 {
         segments = append(segments, baseURL)
      }
      return segments
   }

   // Determine number of segments
   var numSegments uint64
   if template.EndNumber > 0 && startNumber <= template.EndNumber {
      numSegments = template.EndNumber - startNumber + 1
   } else if template.SegmentTimeline != nil {
      times := expandSegmentTimeline(template.SegmentTimeline.S, startNumber)
      numSegments = uint64(len(times))
   } else if template.Duration > 0 && periodSeconds > 0 {
      timescale := template.Timescale
      if timescale == 0 {
         timescale = 1
      }
      totalUnits := periodSeconds * float64(timescale)
      numSegments = uint64(math.Ceil(totalUnits / float64(template.Duration)))
   } else {
      numSegments = 5
   }

   // Expand timeline if present
   var times []uint64
   if template.SegmentTimeline != nil {
      times = expandSegmentTimeline(template.SegmentTimeline.S, startNumber)
   }

   // Generate media segments
   media := template.Media
   for i := uint64(0); i < numSegments; i++ {
      num := startNumber + i
      segPath := media

      // Handle $Number%0Nd$
      if matches := numberFormatRegex.FindStringSubmatch(segPath); len(matches) > 1 {
         width, _ := strconv.Atoi(matches[1])
         formatted := fmt.Sprintf("%0*d", width, num)
         segPath = numberFormatRegex.ReplaceAllString(segPath, formatted)
      } else {
         segPath = strings.ReplaceAll(segPath, "$Number$", strconv.FormatUint(num, 10))
      }

      // Handle $Time$
      if i < uint64(len(times)) {
         segPath = strings.ReplaceAll(segPath, "$Time$", strconv.FormatUint(times[i], 10))
      } else {
         segPath = strings.ReplaceAll(segPath, "$Time$", strconv.FormatUint(num, 10))
      }

      // Handle $RepresentationID$
      segPath = strings.ReplaceAll(segPath, "$RepresentationID$", rep.ID)

      u, err := url.Parse(segPath)
      if err != nil {
         continue
      }
      segments = append(segments, base.ResolveReference(u).String())
   }

   return segments
}

func expandSegmentTimeline(entries []SegmentTimelineEntry, startNumber uint64) []uint64 {
   var times []uint64
   time := uint64(0)
   for _, entry := range entries {
      if entry.T != 0 {
         time = entry.T
      }
      count := 1 + entry.R
      for i := uint64(0); i < count; i++ {
         times = append(times, time)
         time += entry.D
      }
   }
   if len(times) == 0 {
      if startNumber > 0 {
         times = append(times, startNumber)
      } else {
         times = append(times, 0)
      }
   }
   return times
}

func parsePeriodDuration(duration string) float64 {
   if duration == "" {
      return 0.0
   }
   re := regexp.MustCompile(`^PT(?:(\d+(?:\.\d+)?)S)?(?:(\d+)M)?(?:(\d+)H)?$`)
   matches := re.FindStringSubmatch(strings.ToUpper(duration))
   if len(matches) == 0 {
      return 0.0
   }
   var seconds float64
   if matches[1] != "" {
      sec, _ := strconv.ParseFloat(matches[1], 64)
      seconds += sec
   }
   if matches[2] != "" {
      min, _ := strconv.Atoi(matches[2])
      seconds += float64(min) * 60
   }
   if matches[3] != "" {
      hrs, _ := strconv.Atoi(matches[3])
      seconds += float64(hrs) * 3600
   }
   return seconds
}
