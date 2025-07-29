package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io"
   "log"
   "math"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
)

// The hardcoded top-level base URL for resolving all relative paths.
const manifestBaseURL = "http://test.test/test.mpd"

// #region XML Data Structures
// These structs map to the MPEG-DASH MPD XML structure.

// MPD is the root element of the manifest.
type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
   Periods                   []Period `xml:"Period"`
}

// Period represents a period of content.
type Period struct {
   BaseURLS       []BaseURL       `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// BaseURL specifies a base URL.
type BaseURL struct {
   Value string `xml:",chardata"`
}

// AdaptationSet contains a set of interchangeable representations.
// It can define elements that are inherited by its Representations.
type AdaptationSet struct {
   Representations []Representation `xml:"Representation"`
   SegmentTemplate SegmentTemplate  `xml:"SegmentTemplate"`
   SegmentList     SegmentList      `xml:"SegmentList"`
   Initialization  Initialization   `xml:"Initialization"`
}

// Representation describes a specific version of the content.
// It can have its own BaseURL, which overrides any higher-level bases.
type Representation struct {
   ID              string          `xml:"id,attr"`
   BaseURLS        []BaseURL       `xml:"BaseURL"`
   SegmentTemplate SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     SegmentList     `xml:"SegmentList"`
   Initialization  Initialization  `xml:"Initialization"`
}

// Initialization specifies the init segment URL.
type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

// SegmentTemplate defines a template for generating segment URLs.
type SegmentTemplate struct {
   Timescale       uint64          `xml:"timescale,attr"`
   Duration        uint64          `xml:"duration,attr"`
   Initialization  string          `xml:"initialization,attr"`
   Media           string          `xml:"media,attr"`
   SegmentTimeline SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline provides an explicit list of segments and their timings.
type SegmentTimeline struct {
   S []S `xml:"S"`
}

// S defines a segment with its start time, duration, and repeat count.
type S struct {
   Time     *uint64 `xml:"t,attr"` // Time
   Duration uint64  `xml:"d,attr"` // Duration
   Repeat   *int    `xml:"r,attr"` // Repeat count
}

// SegmentList provides an explicit list of segment URLs.
// It can contain its own Initialization element.
type SegmentList struct {
   Timescale      uint64         `xml:"timescale,attr"`
   Duration       uint64         `xml:"duration,attr"`
   Initialization Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL   `xml:"SegmentURL"`
}

// SegmentURL contains the URL for a single media segment.
type SegmentURL struct {
   Media string `xml:"media,attr"`
}

// #endregion

// iso8601DurationRegex parses ISO 8601 duration strings (e.g., "PT1M30.5S").
var iso8601DurationRegex = regexp.MustCompile(`^PT(?:(\d+(?:\.\d+)?)H)?(?:(\d+(?:\.\d+)?)M)?(?:(\d+(?:\.\d+)?)S)?$`)

// parseISODuration converts an ISO 8601 duration string into total seconds.
func parseISODuration(durationStr string) (float64, error) {
   if !strings.HasPrefix(durationStr, "PT") || len(durationStr) == 2 {
      return 0, fmt.Errorf("invalid ISO 8601 duration format: %s", durationStr)
   }
   matches := iso8601DurationRegex.FindStringSubmatch(durationStr)
   if matches == nil {
      return 0, fmt.Errorf("unsupported or invalid ISO 8601 duration format: %s", durationStr)
   }
   var totalSeconds float64
   if matches[1] != "" {
      h, _ := strconv.ParseFloat(matches[1], 64)
      totalSeconds += h * 3600
   }
   if matches[2] != "" {
      m, _ := strconv.ParseFloat(matches[2], 64)
      totalSeconds += m * 60
   }
   if matches[3] != "" {
      s, _ := strconv.ParseFloat(matches[3], 64)
      totalSeconds += s
   }
   return totalSeconds, nil
}

// resolveURL resolves a relative path against a base URL.
func resolveURL(base *url.URL, relativePath string) string {
   relativeURL, err := url.Parse(relativePath)
   if err != nil {
      return relativePath
   }
   return base.ResolveReference(relativeURL).String()
}

func main() {
   // --- 1. Execution: Parse Command-Line Argument ---
   if len(os.Args) < 2 {
      log.Fatalf("Usage: %s <path_to_mpd_file>", os.Args[0])
   }
   mpdFilePath := os.Args[1]

   xmlFile, err := os.Open(mpdFilePath)
   if err != nil {
      log.Fatalf("Error opening file %s: %v", mpdFilePath, err)
   }
   defer xmlFile.Close()

   byteValue, err := io.ReadAll(xmlFile)
   if err != nil {
      log.Fatalf("Error reading file %s: %v", mpdFilePath, err)
   }

   var mpd MPD
   if err := xml.Unmarshal(byteValue, &mpd); err != nil {
      log.Fatalf("Error parsing XML: %v", err)
   }

   // --- 2. URL Resolution: Setup Base URLs ---
   mainBase, err := url.Parse(manifestBaseURL)
   if err != nil {
      log.Fatalf("Error parsing main base URL: %v", err)
   }

   outputResults := make(map[string][]string)

   for _, period := range mpd.Periods {
      // Establish the base URL for the current Period
      periodBase := mainBase
      if len(period.BaseURLS) > 0 {
         periodRelativePath := period.BaseURLS[0].Value
         periodBase = mainBase.ResolveReference(&url.URL{Path: periodRelativePath})
      }

      for _, adaptationSet := range period.AdaptationSets {
         for _, rep := range adaptationSet.Representations {
            repID := rep.ID
            var segmentURLs []string

            // 📜 --- Base URL and Inheritance Logic --- 📜
            representationBase := periodBase
            if len(rep.BaseURLS) > 0 {
               repRelativePath := rep.BaseURLS[0].Value
               representationBase = periodBase.ResolveReference(&url.URL{Path: repRelativePath})
            }

            effectiveTpl := adaptationSet.SegmentTemplate
            if rep.SegmentTemplate.Timescale != 0 {
               effectiveTpl.Timescale = rep.SegmentTemplate.Timescale
            }
            if rep.SegmentTemplate.Duration != 0 {
               effectiveTpl.Duration = rep.SegmentTemplate.Duration
            }
            if rep.SegmentTemplate.Initialization != "" {
               effectiveTpl.Initialization = rep.SegmentTemplate.Initialization
            }
            if rep.SegmentTemplate.Media != "" {
               effectiveTpl.Media = rep.SegmentTemplate.Media
            }
            if len(rep.SegmentTemplate.SegmentTimeline.S) > 0 {
               effectiveTpl.SegmentTimeline = rep.SegmentTemplate.SegmentTimeline
            }

            effectiveList := adaptationSet.SegmentList
            if len(rep.SegmentList.SegmentURLs) > 0 {
               effectiveList = rep.SegmentList
            }

            effectiveTopLevelInit := adaptationSet.Initialization
            if rep.Initialization.SourceURL != "" {
               effectiveTopLevelInit = rep.Initialization
            }

            // --- Handle Initialization Segment ---
            var initURL string
            if effectiveTopLevelInit.SourceURL != "" {
               initURL = effectiveTopLevelInit.SourceURL
            } else if effectiveList.Initialization.SourceURL != "" {
               initURL = effectiveList.Initialization.SourceURL
            } else if effectiveTpl.Initialization != "" {
               initURL = effectiveTpl.Initialization
            }

            if initURL != "" {
               replacer := strings.NewReplacer("$RepresentationID$", repID)
               initPath := replacer.Replace(initURL)
               segmentURLs = append(segmentURLs, resolveURL(representationBase, initPath))
            }

            // --- 3. Segment Generation (with precedence) ---
            hasTimeline := len(effectiveTpl.SegmentTimeline.S) > 0
            hasSegmentList := len(effectiveList.SegmentURLs) > 0
            hasDurationTemplate := effectiveTpl.Duration > 0 && effectiveTpl.Timescale > 0 && mpd.MediaPresentationDuration != ""

            if hasTimeline {
               // Method 1: <SegmentTimeline>
               var currentTime uint64 = 0
               var segmentNumber int = 1
               for _, s := range effectiveTpl.SegmentTimeline.S {
                  if s.Time != nil {
                     currentTime = *s.Time
                  }
                  repeatCount := 0
                  if s.Repeat != nil {
                     repeatCount = *s.Repeat
                  }
                  for i := 0; i <= repeatCount; i++ {
                     replacer := strings.NewReplacer(
                        "$RepresentationID$", repID,
                        "$Number$", strconv.Itoa(segmentNumber),
                        "$Time$", strconv.FormatUint(currentTime, 10),
                     )
                     mediaPath := replacer.Replace(effectiveTpl.Media)
                     segmentURLs = append(segmentURLs, resolveURL(representationBase, mediaPath))
                     currentTime += s.Duration
                     segmentNumber++
                  }
               }
            } else if hasSegmentList {
               // Method 2: <SegmentList>
               for _, segURL := range effectiveList.SegmentURLs {
                  if segURL.Media != "" {
                     segmentURLs = append(segmentURLs, resolveURL(representationBase, segURL.Media))
                  }
               }
            } else if hasDurationTemplate {
               // Method 3: Fallback using SegmentTemplate@duration
               totalDuration, err := parseISODuration(mpd.MediaPresentationDuration)
               if err != nil {
                  log.Printf("Warning: Could not parse MPD duration '%s' for '%s'. Skipping. Error: %v", mpd.MediaPresentationDuration, repID, err)
                  continue
               }
               segmentDuration := float64(effectiveTpl.Duration) / float64(effectiveTpl.Timescale)
               if segmentDuration <= 0 {
                  log.Printf("Warning: Invalid segment duration for '%s'. Skipping.", repID)
                  continue
               }
               numSegments := int(math.Ceil(totalDuration / segmentDuration))
               for i := 1; i <= numSegments; i++ {
                  replacer := strings.NewReplacer("$RepresentationID$", repID, "$Number$", strconv.Itoa(i))
                  mediaPath := replacer.Replace(effectiveTpl.Media)
                  segmentURLs = append(segmentURLs, resolveURL(representationBase, mediaPath))
               }
            } else if len(rep.BaseURLS) > 0 {
               // Method 4: Final fallback using the Representation's own <BaseURL> elements as segments.
               for _, baseURL := range rep.BaseURLS {
                  if baseURL.Value != "" {
                     // Resolve against the Period's base, as these BaseURLs are the segments themselves,
                     // not a new base for other relative paths.
                     segmentURLs = append(segmentURLs, resolveURL(periodBase, baseURL.Value))
                  }
               }
            }

            if len(segmentURLs) > 0 {
               outputResults[repID] = segmentURLs
            }
         }
      }
   }

   // --- 4. Output: Print final JSON object to stdout ---
   jsonOutput, err := json.MarshalIndent(outputResults, "", "  ")
   if err != nil {
      log.Fatalf("Error generating JSON output: %v", err)
   }

   fmt.Println(string(jsonOutput))
}
