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
)

// #############################################################################
// ## 📜 XML Struct Definitions for MPEG-DASH MPD
// #############################################################################

// MPD is the root element of the Media Presentation Description.
type MPD struct {
   XMLName                   xml.Name  `xml:"MPD"`
   MediaPresentationDuration string    `xml:"mediaPresentationDuration,attr"`
   BaseURLs                  []BaseURL `xml:"BaseURL"`
   Periods                   []Period  `xml:"Period"`
}

// Period represents a Period of content.
type Period struct {
   Duration       string          `xml:"duration,attr"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
   BaseURL        *BaseURL        `xml:"BaseURL"`
}

// AdaptationSet is a set of interchangeable Representations.
type AdaptationSet struct {
   Representations []Representation `xml:"Representation"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Initialization  *Initialization  `xml:"Initialization"`
}

// Representation is a specific version of the content (e.g., a certain bitrate).
type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURLs        []BaseURL        `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Initialization  *Initialization  `xml:"Initialization"`
}

// BaseURL specifies a base URL for relative paths.
type BaseURL struct {
   Value string `xml:",chardata"`
}

// SegmentTemplate defines a template for generating segment URLs.
type SegmentTemplate struct {
   Timescale       uint64           `xml:"timescale,attr"`
   StartNumber     uint64           `xml:"startNumber,attr"`
   EndNumber       uint64           `xml:"endNumber,attr"`
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   Duration        uint64           `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline provides an explicit list of segments and their timings.
type SegmentTimeline struct {
   Segments []S `xml:"S"`
}

// S represents a single segment or a series of repeated segments in a timeline.
type S struct {
   T uint64 `xml:"t,attr"` // Time
   D uint64 `xml:"d,attr"` // Duration
   R int64  `xml:"r,attr"` // Repeat count
}

// SegmentList provides an explicit list of segment URLs.
type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

// Initialization defines the URL for an initialization segment.
type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

// SegmentURL defines the URL for a media segment.
type SegmentURL struct {
   Media string `xml:"media,attr"`
}

// #############################################################################
// ## 🚀 Main Execution
// #############################################################################

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintln(os.Stderr, "Usage: go run main.go <path_to_mpd_file>")
      os.Exit(1)
   }

   mpdFilePath := os.Args[1]
   file, err := os.Open(mpdFilePath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error opening file '%s': %v\n", mpdFilePath, err)
      os.Exit(1)
   }
   defer file.Close()

   byteValue, err := io.ReadAll(file)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(byteValue, &mpd); err != nil {
      fmt.Fprintf(os.Stderr, "Error unmarshalling XML: %v\n", err)
      os.Exit(1)
   }

   results, err := processMPD(mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error processing MPD: %v\n", err)
      os.Exit(1)
   }

   jsonOutput, err := json.MarshalIndent(results, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshalling JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonOutput))
}

// #############################################################################
// ## 🧠 Core MPD Processing Logic
// #############################################################################

// processMPD iterates through the MPD structure to generate segment URLs.
func processMPD(mpd MPD) (map[string][]string, error) {
   results := make(map[string][]string)
   hardcodedBase := "http://test.test/test.mpd"

   docBase, err := url.Parse(hardcodedBase)
   if err != nil {
      return nil, fmt.Errorf("invalid hardcoded base URL: %w", err)
   }

   for i := range mpd.Periods {
      period := &mpd.Periods[i]
      // Resolve Period-level base URL
      periodBase := docBase
      if period.BaseURL != nil {
         periodBase, err = resolveBase(periodBase, period.BaseURL.Value)
         if err != nil {
            return nil, fmt.Errorf("failed to resolve Period BaseURL: %w", err)
         }
      }

      for _, as := range period.AdaptationSets {
         for _, rep := range as.Representations {
            initURL, mediaURLs, err := processRepresentation(mpd, period, periodBase, as, rep)
            if err != nil {
               return nil, fmt.Errorf("error processing representation '%s': %w", rep.ID, err)
            }

            if _, exists := results[rep.ID]; !exists && initURL != "" {
               results[rep.ID] = []string{initURL}
            }

            results[rep.ID] = append(results[rep.ID], mediaURLs...)
         }
      }
   }

   return results, nil
}

// processRepresentation handles a single Representation to generate its URL list.
func processRepresentation(mpd MPD, period *Period, periodBase *url.URL, as AdaptationSet, rep Representation) (string, []string, error) {
   var err error

   repBase := periodBase
   for _, baseURL := range rep.BaseURLs {
      repBase, err = resolveBase(repBase, baseURL.Value)
      if err != nil {
         return "", nil, fmt.Errorf("failed to resolve Representation BaseURL: %w", err)
      }
   }

   effectiveST := rep.SegmentTemplate
   if effectiveST == nil {
      effectiveST = as.SegmentTemplate
   }

   if effectiveST != nil && effectiveST.Timescale == 0 {
      effectiveST.Timescale = 1
   }

   effectiveSL := rep.SegmentList
   if effectiveSL == nil {
      effectiveSL = as.SegmentList
   }

   initURL, err := findInitURL(rep, effectiveSL, effectiveST, repBase)
   if err != nil {
      return "", nil, err
   }

   mediaURLs, err := generateSegmentURLs(mpd, period, periodBase, repBase, rep, effectiveST, effectiveSL)
   if err != nil {
      return "", nil, err
   }

   return initURL, mediaURLs, nil
}

// #############################################################################
// ## 📑 Segment Generation Logic
// #############################################################################

func findInitURL(rep Representation, effectiveSL *SegmentList, effectiveST *SegmentTemplate, repBase *url.URL) (string, error) {
   var initPath string

   if rep.Initialization != nil && rep.Initialization.SourceURL != "" {
      initPath = rep.Initialization.SourceURL
   } else if effectiveSL != nil && effectiveSL.Initialization != nil && effectiveSL.Initialization.SourceURL != "" {
      initPath = effectiveSL.Initialization.SourceURL
   } else if effectiveST != nil && effectiveST.Initialization != "" {
      initPath = effectiveST.Initialization
   }

   if initPath != "" {
      tpl := substitutePlaceholders(initPath, rep.ID, 0, 0)
      return resolvePath(repBase, tpl)
   }

   return "", nil
}

func generateSegmentURLs(mpd MPD, period *Period, periodBase, repBase *url.URL, rep Representation, effectiveST *SegmentTemplate, effectiveSL *SegmentList) ([]string, error) {
   if effectiveST != nil && effectiveST.SegmentTimeline != nil {
      return generateTimelineSegments(effectiveST, rep.ID, repBase)
   }

   if effectiveSL != nil {
      return generateListSegments(effectiveSL, repBase)
   }

   if effectiveST != nil && effectiveST.Duration > 0 {
      if effectiveST.EndNumber > 0 {
         return generateNumberBasedTemplateSegments(effectiveST, rep.ID, repBase)
      }
      return generateDurationTemplateSegments(period.Duration, mpd.MediaPresentationDuration, effectiveST, rep.ID, repBase)
   }

   if len(rep.BaseURLs) > 0 {
      var urls []string
      for _, baseURL := range rep.BaseURLs {
         literalURL, err := resolvePath(periodBase, baseURL.Value)
         if err != nil {
            return nil, err
         }
         urls = append(urls, literalURL)
      }
      return urls, nil
   }

   return []string{}, nil
}

func generateTimelineSegments(st *SegmentTemplate, repID string, base *url.URL) ([]string, error) {
   var urls []string
   currentTime := uint64(0)
   currentNumber := st.StartNumber
   if currentNumber == 0 {
      currentNumber = 1
   }

   for _, s := range st.SegmentTimeline.Segments {
      if s.T > 0 {
         currentTime = s.T
      }

      mediaPath := substitutePlaceholders(st.Media, repID, currentNumber, currentTime)
      segmentURL, err := resolvePath(base, mediaPath)
      if err != nil {
         return nil, err
      }
      urls = append(urls, segmentURL)
      currentNumber++

      for i := int64(0); i < s.R; i++ {
         currentTime += s.D
         mediaPath := substitutePlaceholders(st.Media, repID, currentNumber, currentTime)
         segmentURL, err := resolvePath(base, mediaPath)
         if err != nil {
            return nil, err
         }
         urls = append(urls, segmentURL)
         currentNumber++
      }
      currentTime += s.D
   }
   return urls, nil
}

func generateListSegments(sl *SegmentList, base *url.URL) ([]string, error) {
   var urls []string
   for _, segmentURL := range sl.SegmentURLs {
      if segmentURL.Media != "" {
         resolvedURL, err := resolvePath(base, segmentURL.Media)
         if err != nil {
            return nil, err
         }
         urls = append(urls, resolvedURL)
      }
   }
   return urls, nil
}

func generateDurationTemplateSegments(periodDuration, mpdDuration string, st *SegmentTemplate, repID string, base *url.URL) ([]string, error) {
   var totalDurationSecs float64
   var err error

   if periodDuration != "" {
      totalDurationSecs, err = parseISODuration(periodDuration)
   } else {
      totalDurationSecs, err = parseISODuration(mpdDuration)
   }
   if err != nil {
      return nil, fmt.Errorf("could not parse duration for segment calculation: %w", err)
   }

   var urls []string
   segmentDuration := float64(st.Duration) / float64(st.Timescale)
   if segmentDuration > 0 {
      numSegments := int(math.Ceil(totalDurationSecs / segmentDuration))
      start := st.StartNumber
      if start == 0 {
         start = 1
      }
      for i := 0; i < numSegments; i++ {
         number := start + uint64(i)
         mediaPath := substitutePlaceholders(st.Media, repID, number, 0)
         segmentURL, err := resolvePath(base, mediaPath)
         if err != nil {
            return nil, err
         }
         urls = append(urls, segmentURL)
      }
   }
   return urls, nil
}

func generateNumberBasedTemplateSegments(st *SegmentTemplate, repID string, base *url.URL) ([]string, error) {
   var urls []string
   start := st.StartNumber
   if start == 0 {
      start = 1
   }
   end := st.EndNumber

   if end < start {
      return nil, fmt.Errorf("endNumber (%d) cannot be less than startNumber (%d)", end, start)
   }

   numSegments := (end - start) + 1
   for i := uint64(0); i < numSegments; i++ {
      number := start + i
      mediaPath := substitutePlaceholders(st.Media, repID, number, 0)
      segmentURL, err := resolvePath(base, mediaPath)
      if err != nil {
         return nil, err
      }
      urls = append(urls, segmentURL)
   }
   return urls, nil
}

// #############################################################################
// ## 🛠️ Helper Functions
// #############################################################################

// Regular expression to find placeholders like $Number%05d$.
var placeholderRegex = regexp.MustCompile(`\$([a-zA-Z]+)(%0?([0-9]*)d)?\$`)

func resolveBase(base *url.URL, newBase string) (*url.URL, error) {
   newURL, err := url.Parse(newBase)
   if err != nil {
      return nil, err
   }
   return base.ResolveReference(newURL), nil
}

func resolvePath(base *url.URL, path string) (string, error) {
   pathURL, err := url.Parse(path)
   if err != nil {
      return "", err
   }
   return base.ResolveReference(pathURL).String(), nil
}

func substitutePlaceholders(template string, repID string, number, time uint64) string {
   return placeholderRegex.ReplaceAllStringFunc(template, func(match string) string {
      parts := placeholderRegex.FindStringSubmatch(match)
      identifier := parts[1]
      formatSpec := parts[2]

      switch identifier {
      case "RepresentationID":
         return repID
      case "Number":
         if formatSpec == "" {
            return strconv.FormatUint(number, 10)
         }
         return fmt.Sprintf(formatSpec, number)
      case "Time":
         if formatSpec == "" {
            return strconv.FormatUint(time, 10)
         }
         return fmt.Sprintf(formatSpec, time)
      default:
         return match // Return the original string if identifier is unknown
      }
   })
}

func parseISODuration(durationStr string) (float64, error) {
   if durationStr == "" {
      return 0, nil
   }

   durationRegex := regexp.MustCompile(`PT(?:([0-9.]+)H)?(?:([0-9.]+)M)?(?:([0-9.]+)S)?`)
   parts := durationRegex.FindStringSubmatch(durationStr)
   if parts == nil {
      return 0.0, fmt.Errorf("unsupported ISO 8601 duration format: '%s'", durationStr)
   }

   var totalSeconds float64
   if parts[1] != "" {
      h, _ := strconv.ParseFloat(parts[1], 64)
      totalSeconds += h * 3600
   }
   if parts[2] != "" {
      m, _ := strconv.ParseFloat(parts[2], 64)
      totalSeconds += m * 60
   }
   if parts[3] != "" {
      s, _ := strconv.ParseFloat(parts[3], 64)
      totalSeconds += s
   }

   return totalSeconds, nil
}
