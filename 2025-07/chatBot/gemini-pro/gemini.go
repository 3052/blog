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
   "regexp"
   "strconv"
   "strings"
)

// --- XML Structs for MPD Parsing ---

type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
   Periods                   []Period `xml:"Period"`
}

type Period struct {
   XMLName        xml.Name        `xml:"Period"`
   BaseURL        string          `xml:"BaseURL"` // Added to respect Period-level BaseURL
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   XMLName         xml.Name         `xml:"AdaptationSet"`
   Representations []Representation `xml:"Representation"`
   SegmentTemplate SegmentTemplate  `xml:"SegmentTemplate"`
}

type Representation struct {
   XMLName         xml.Name        `xml:"Representation"`
   ID              string          `xml:"id,attr"`
   SegmentTemplate SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentTemplate struct {
   XMLName         xml.Name        `xml:"SegmentTemplate"`
   Timescale       int             `xml:"timescale,attr"`
   Media           string          `xml:"media,attr"`
   Initialization  string          `xml:"initialization,attr"`
   StartNumber     int             `xml:"startNumber,attr"`
   Duration        int             `xml:"duration,attr"`
   SegmentTimeline SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   XMLName  xml.Name `xml:"SegmentTimeline"`
   Segments []S      `xml:"S"`
}

type S struct {
   Time     *uint64 `xml:"t,attr"`
   Duration uint64  `xml:"d,attr"`
   Repeat   *int    `xml:"r,attr"`
}

// --- Helper Functions ---

var durationRegex = regexp.MustCompile(`^PT(?:(\d+(?:\.\d+)?)H)?(?:(\d+(?:\.\d+)?)M)?(?:(\d+(?:\.\d+)?)S)?$`)

func parseISODuration(durationStr string) (float64, error) {
   if !strings.HasPrefix(durationStr, "PT") {
      return 0, fmt.Errorf("invalid ISO 8601 duration format: missing 'PT' prefix in '%s'", durationStr)
   }
   matches := durationRegex.FindStringSubmatch(durationStr)
   if matches == nil {
      return 0, fmt.Errorf("could not parse ISO 8601 duration: %s", durationStr)
   }

   var totalSeconds float64
   if h, err := strconv.ParseFloat(matches[1], 64); err == nil {
      totalSeconds += h * 3600
   }
   if m, err := strconv.ParseFloat(matches[2], 64); err == nil {
      totalSeconds += m * 60
   }
   if s, err := strconv.ParseFloat(matches[3], 64); err == nil {
      totalSeconds += s
   }
   return totalSeconds, nil
}

// --- Main Execution ---

func main() {
   // 1. Setup and file parsing
   if len(os.Args) < 2 {
      log.Fatalf("Usage: go run main.go <mpd_file_path>")
   }
   mpdPath := os.Args[1]

   xmlFile, err := ioutil.ReadFile(mpdPath)
   if err != nil {
      log.Fatalf("Error reading MPD file: %v", err)
   }

   var mpd MPD
   if err := xml.Unmarshal(xmlFile, &mpd); err != nil {
      log.Fatalf("Error unmarshaling XML: %v", err)
   }

   // This is the top-level URL for the manifest file itself.
   manifestBaseURL, err := url.Parse("http://test.test/test.mpd")
   if err != nil {
      log.Fatalf("Error parsing base URL: %v", err)
   }

   segmentURLs := make(map[string][]string)

   // 2. Iterate through the MPD structure
   for _, period := range mpd.Periods {
      // --- MODIFICATION START: Determine the effective Base URL for this Period ---
      effectiveBaseURL := manifestBaseURL
      if period.BaseURL != "" {
         periodBase, err := url.Parse(period.BaseURL)
         if err != nil {
            log.Printf("Warning: Could not parse Period BaseURL '%s'. Using manifest base URL.", period.BaseURL)
         } else {
            // Resolve the Period's BaseURL relative to the manifest's URL.
            effectiveBaseURL = manifestBaseURL.ResolveReference(periodBase)
         }
      }
      // --- MODIFICATION END ---

      for _, adaptationSet := range period.AdaptationSets {
         for _, rep := range adaptationSet.Representations {
            finalTemplate := rep.SegmentTemplate
            if finalTemplate.Media == "" {
               finalTemplate = adaptationSet.SegmentTemplate
            }
            if finalTemplate.Media == "" {
               log.Printf("Warning: Skipping Representation '%s' due to missing SegmentTemplate.", rep.ID)
               continue
            }

            var urls []string

            // Construct initialization segment URL using the *effective* base URL
            if finalTemplate.Initialization != "" {
               initPath := strings.ReplaceAll(finalTemplate.Initialization, "$RepresentationID$", rep.ID)
               resolvedURL, _ := effectiveBaseURL.Parse(initPath)
               urls = append(urls, resolvedURL.String())
            }

            // Generate media segment URLs using the *effective* base URL
            if len(finalTemplate.SegmentTimeline.Segments) > 0 {
               segmentNumber := finalTemplate.StartNumber
               var currentTime uint64 = 0

               for _, s := range finalTemplate.SegmentTimeline.Segments {
                  if s.Time != nil {
                     currentTime = *s.Time
                  }
                  repeatCount := 0
                  if s.Repeat != nil {
                     repeatCount = *s.Repeat
                  }
                  for i := 0; i <= repeatCount; i++ {
                     mediaPath := finalTemplate.Media
                     mediaPath = strings.ReplaceAll(mediaPath, "$RepresentationID$", rep.ID)
                     mediaPath = strings.ReplaceAll(mediaPath, "$Number$", strconv.Itoa(segmentNumber))
                     mediaPath = strings.ReplaceAll(mediaPath, "$Time$", strconv.FormatUint(currentTime, 10))

                     resolvedURL, _ := effectiveBaseURL.Parse(mediaPath)
                     urls = append(urls, resolvedURL.String())

                     segmentNumber++
                     currentTime += s.Duration
                  }
               }
            } else { // Fallback to duration-based calculation
               if finalTemplate.Duration == 0 || mpd.MediaPresentationDuration == "" {
                  log.Printf("Warning: Skipping Rep '%s'. Missing SegmentTimeline and not enough info for fallback.", rep.ID)
                  continue
               }
               totalDurationSec, err := parseISODuration(mpd.MediaPresentationDuration)
               if err != nil {
                  log.Fatalf("Error parsing mediaPresentationDuration: %v", err)
               }

               segmentDurationSec := float64(finalTemplate.Duration) / float64(finalTemplate.Timescale)
               numSegments := int(math.Ceil(totalDurationSec / segmentDurationSec))

               for i := 0; i < numSegments; i++ {
                  segmentNumber := finalTemplate.StartNumber + i
                  mediaPath := strings.ReplaceAll(finalTemplate.Media, "$RepresentationID$", rep.ID)
                  mediaPath = strings.ReplaceAll(mediaPath, "$Number$", strconv.Itoa(segmentNumber))

                  resolvedURL, _ := effectiveBaseURL.Parse(mediaPath)
                  urls = append(urls, resolvedURL.String())
               }
            }
            segmentURLs[rep.ID] = urls
         }
      }
   }

   // 3. Marshal and print the final JSON object
   jsonOutput, err := json.MarshalIndent(segmentURLs, "", "  ")
   if err != nil {
      log.Fatalf("Error marshaling to JSON: %v", err)
   }

   fmt.Println(string(jsonOutput))
}
