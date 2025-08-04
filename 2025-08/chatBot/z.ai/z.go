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
)

const initialMPDURL = "http://test.test/test.mpd"

type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   BaseURL                   string   `xml:"BaseURL"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
   Periods                   []Period `xml:"Period"`
}

type Period struct {
   Start          string          `xml:"start,attr"`    // Start time in ISO 8601 format
   Duration       string          `xml:"duration,attr"` // Duration in ISO 8601 format
   BaseURL        string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   Representations []Representation `xml:"Representation"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   Bandwidth       int              `xml:"bandwidth,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   Duration        int              `xml:"duration,attr"`
   Timescale       int              `xml:"timescale,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"` // Added endNumber attribute
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentList struct {
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
   Range string `xml:"range,attr"` // Byte range for HTTP partial requests
}

type SegmentTimeline struct {
   S []TimelineElement `xml:"S"`
}

type TimelineElement struct {
   D int64 `xml:"d,attr"` // Duration in timescale units
   T int64 `xml:"t,attr"` // Start time in timescale units (optional)
   R int64 `xml:"r,attr"` // Repeat count (optional, default 0)
}

func main() {
   if len(os.Args) < 2 {
      log.Fatal("Usage: go run main.go <mpd_file_path>")
   }

   mpdFilePath := os.Args[1]
   data, err := os.ReadFile(mpdFilePath)
   if err != nil {
      log.Fatalf("Error reading MPD file: %v", err)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      log.Fatalf("Error parsing MPD XML: %v", err)
   }

   // Parse the initial MPD URL (from CLI)
   baseURL, err := url.Parse(initialMPDURL)
   if err != nil {
      log.Fatalf("Error parsing initial MPD URL: %v", err)
   }

   // Resolve MPD BaseURL if present
   mpdBaseURL := *baseURL
   if mpd.BaseURL != "" {
      mpdURL, err := url.Parse(mpd.BaseURL)
      if err != nil {
         log.Fatalf("Error parsing MPD BaseURL: %v", err)
      }
      resolvedURL := baseURL.ResolveReference(mpdURL)
      mpdBaseURL = *resolvedURL
   }

   result := make(map[string][]string)

   // Parse MPD duration
   mpdDurationSec, err := parseDuration(mpd.MediaPresentationDuration)
   if err != nil {
      log.Fatalf("Error parsing MPD duration: %v", err)
   }

   // Precompute period start times and durations
   periodStarts := make([]float64, len(mpd.Periods))
   periodDurations := make([]float64, len(mpd.Periods))

   for i, period := range mpd.Periods {
      // Parse period start time (default to 0 if not specified)
      if period.Start == "" {
         if i == 0 {
            periodStarts[i] = 0
         } else {
            periodStarts[i] = periodStarts[i-1] + periodDurations[i-1]
         }
      } else {
         startSec, err := parseDuration(period.Start)
         if err != nil {
            log.Printf("Error parsing Period %d start time: %v", i, err)
            continue
         }
         periodStarts[i] = startSec
      }

      // Parse period duration if specified
      if period.Duration != "" {
         durationSec, err := parseDuration(period.Duration)
         if err != nil {
            log.Printf("Error parsing Period %d duration: %v", i, err)
            continue
         }
         periodDurations[i] = durationSec
      } else {
         // Calculate period duration from start times
         if i < len(mpd.Periods)-1 {
            // For non-last periods, duration is next period's start minus current period's start
            nextStart := periodStarts[i+1]
            if mpd.Periods[i+1].Start != "" {
               nextStart, err = parseDuration(mpd.Periods[i+1].Start)
               if err != nil {
                  log.Printf("Error parsing Period %d start time: %v", i+1, err)
                  continue
               }
            }
            periodDurations[i] = nextStart - periodStarts[i]
         } else {
            // For last period, duration is MPD duration minus period start
            periodDurations[i] = mpdDurationSec - periodStarts[i]
         }
      }
   }

   for periodIdx, period := range mpd.Periods {
      // Resolve Period BaseURL relative to MPD BaseURL
      periodBaseURL := mpdBaseURL
      if period.BaseURL != "" {
         periodURL, err := url.Parse(period.BaseURL)
         if err != nil {
            log.Printf("Error parsing Period BaseURL: %v", err)
            continue
         }
         resolvedURL := mpdBaseURL.ResolveReference(periodURL)
         periodBaseURL = *resolvedURL
      }

      for _, adaptationSet := range period.AdaptationSets {
         // Resolve AdaptationSet BaseURL relative to Period BaseURL
         adaptationBaseURL := periodBaseURL
         if adaptationSet.BaseURL != "" {
            adaptationURL, err := url.Parse(adaptationSet.BaseURL)
            if err != nil {
               log.Printf("Error parsing AdaptationSet BaseURL: %v", err)
               continue
            }
            resolvedURL := periodBaseURL.ResolveReference(adaptationURL)
            adaptationBaseURL = *resolvedURL
         }

         // Get SegmentTemplate and SegmentList from AdaptationSet if available
         adaptationSegmentTemplate := adaptationSet.SegmentTemplate
         adaptationSegmentList := adaptationSet.SegmentList

         for _, rep := range adaptationSet.Representations {
            // Resolve Representation BaseURL relative to AdaptationSet BaseURL
            repBaseURL := adaptationBaseURL
            if rep.BaseURL != "" {
               repURL, err := url.Parse(rep.BaseURL)
               if err != nil {
                  log.Printf("Error parsing Representation BaseURL for %s: %v", rep.ID, err)
                  continue
               }
               resolvedURL := adaptationBaseURL.ResolveReference(repURL)
               repBaseURL = *resolvedURL
            }

            // Use Representation's SegmentList if available, otherwise inherit from AdaptationSet
            segmentList := rep.SegmentList
            if segmentList == nil {
               segmentList = adaptationSegmentList
            }

            // Use Representation's SegmentTemplate if available, otherwise inherit from AdaptationSet
            segmentTemplate := rep.SegmentTemplate
            if segmentTemplate == nil {
               segmentTemplate = adaptationSegmentTemplate
            }

            var segments []string

            // Check if we have any segment information
            hasSegmentInfo := false

            // Add initialization segment if it exists (from SegmentList or SegmentTemplate)
            if segmentList != nil && segmentList.Initialization != nil {
               initURL := generateInitializationURLFromList(segmentList.Initialization, repBaseURL)
               if initURL != "" {
                  segments = append(segments, initURL)
                  hasSegmentInfo = true
               }
            } else if segmentTemplate != nil && segmentTemplate.Initialization != "" {
               initURL := generateInitializationURL(segmentTemplate, rep, repBaseURL)
               if initURL != "" {
                  segments = append(segments, initURL)
                  hasSegmentInfo = true
               }
            }

            // Generate segments based on SegmentList, SegmentTimeline, or fixed duration
            if segmentList != nil {
               mediaSegments := generateSegmentsFromList(segmentList, repBaseURL)
               segments = append(segments, mediaSegments...)
               hasSegmentInfo = true
            } else if segmentTemplate != nil {
               if segmentTemplate.SegmentTimeline != nil {
                  mediaSegments := generateSegmentsFromTimeline(segmentTemplate, rep, repBaseURL)
                  segments = append(segments, mediaSegments...)
                  hasSegmentInfo = true
               } else {
                  // Use period duration for segment count calculation
                  mediaSegments := generateSegmentsFromDuration(segmentTemplate, rep, periodDurations[periodIdx], repBaseURL)
                  segments = append(segments, mediaSegments...)
                  hasSegmentInfo = true
               }
            }

            // If no segment information was found, use the resolved BaseURL directly
            if !hasSegmentInfo {
               segments = []string{repBaseURL.String()}
            }

            // Append segments to existing list for this Representation ID, or create new entry
            if existingSegments, exists := result[rep.ID]; exists {
               result[rep.ID] = append(existingSegments, segments...)
            } else {
               result[rep.ID] = segments
            }
         }
      }
   }

   jsonData, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      log.Fatalf("Error generating JSON: %v", err)
   }

   fmt.Println(string(jsonData))
}

func generateInitializationURLFromList(init *Initialization, baseURL url.URL) string {
   if init.SourceURL == "" {
      return ""
   }

   // Parse and resolve the initialization URL
   initURL, err := url.Parse(init.SourceURL)
   if err != nil {
      log.Printf("Error parsing initialization URL: %v", err)
      return ""
   }
   resolvedURL := baseURL.ResolveReference(initURL)
   return resolvedURL.String()
}

func generateInitializationURL(segmentTemplate *SegmentTemplate, rep Representation, baseURL url.URL) string {
   // Replace placeholders in initialization URL
   // Initialization typically uses RepresentationID and Bandwidth, but not Number or Time
   initURLStr := replacePlaceholders(segmentTemplate.Initialization, 0, rep.Bandwidth, rep.ID, 0)

   // Parse and resolve the initialization URL
   initURL, err := url.Parse(initURLStr)
   if err != nil {
      log.Printf("Error parsing initialization URL for representation %s: %v", rep.ID, err)
      return ""
   }
   resolvedURL := baseURL.ResolveReference(initURL)
   return resolvedURL.String()
}

func generateSegmentsFromList(segmentList *SegmentList, baseURL url.URL) []string {
   var segments []string

   for _, segmentURL := range segmentList.SegmentURLs {
      // Parse and resolve the segment URL
      mediaURL, err := url.Parse(segmentURL.Media)
      if err != nil {
         log.Printf("Error parsing segment URL: %v", err)
         continue
      }
      resolvedURL := baseURL.ResolveReference(mediaURL)
      segments = append(segments, resolvedURL.String())
   }

   return segments
}

func generateSegmentsFromTimeline(segmentTemplate *SegmentTemplate, rep Representation, baseURL url.URL) []string {
   var segments []string

   // Default startNumber to 1 if not specified
   startNumber := segmentTemplate.StartNumber
   if startNumber == 0 {
      startNumber = 1
   }
   currentSegmentNumber := startNumber
   currentTime := int64(0) // Track current time across S elements

   // Default timescale to 1 if not specified
   timescale := segmentTemplate.Timescale
   if timescale == 0 {
      timescale = 1
   }

   for _, s := range segmentTemplate.SegmentTimeline.S {
      // If this S element has a specific start time, use it
      if s.T != 0 {
         currentTime = s.T
      }

      // Calculate repeat count (default to 0 if not specified)
      repeatCount := s.R
      if repeatCount < 0 {
         repeatCount = 0
      }

      // Generate segments for this timeline element
      for i := int64(0); i <= repeatCount; i++ {
         // Replace placeholders in media URL with current time
         mediaURLStr := replacePlaceholders(segmentTemplate.Media, currentSegmentNumber, rep.Bandwidth, rep.ID, currentTime)

         // Parse and resolve the media URL
         mediaURL, err := url.Parse(mediaURLStr)
         if err != nil {
            log.Printf("Error parsing media URL for representation %s: %v", rep.ID, err)
            continue
         }
         resolvedURL := baseURL.ResolveReference(mediaURL)
         segments = append(segments, resolvedURL.String())

         // Increment segment number and time for next segment
         currentSegmentNumber++
         currentTime += s.D
      }
   }

   return segments
}

func generateSegmentsFromDuration(segmentTemplate *SegmentTemplate, rep Representation, periodDurationSec float64, baseURL url.URL) []string {
   var totalSegments int

   // Default startNumber to 1 if not specified
   startNumber := segmentTemplate.StartNumber
   if startNumber == 0 {
      startNumber = 1
   }

   // Default timescale to 1 if not specified
   timescale := segmentTemplate.Timescale
   if timescale == 0 {
      timescale = 1
   }

   // If endNumber is present, use it to determine segment count
   if segmentTemplate.EndNumber > 0 {
      totalSegments = segmentTemplate.EndNumber - startNumber + 1
   } else {
      // Calculate segment count using period duration: ceil(PeriodDurationInSeconds * timescale / duration)
      if segmentTemplate.Duration > 0 {
         totalSegments = int(math.Ceil(float64(periodDurationSec*float64(timescale)) / float64(segmentTemplate.Duration)))
      } else {
         log.Printf("Missing duration for representation %s", rep.ID)
         return nil
      }
   }

   // Generate segment URLs
   segments := make([]string, 0, totalSegments)
   for i := startNumber; i < startNumber+totalSegments; i++ {
      // Calculate segment time (in timescale units)
      segmentTime := int64(i-startNumber) * int64(segmentTemplate.Duration)

      // Replace placeholders in media URL
      mediaURLStr := replacePlaceholders(segmentTemplate.Media, i, rep.Bandwidth, rep.ID, segmentTime)

      // Parse and resolve the media URL
      mediaURL, err := url.Parse(mediaURLStr)
      if err != nil {
         log.Printf("Error parsing media URL for representation %s: %v", rep.ID, err)
         continue
      }
      resolvedURL := baseURL.ResolveReference(mediaURL)
      segments = append(segments, resolvedURL.String())
   }

   return segments
}

func parseDuration(dur string) (float64, error) {
   re := regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?`)
   matches := re.FindStringSubmatch(dur)
   if matches == nil {
      return 0, fmt.Errorf("invalid duration format: %s", dur)
   }

   var hours, minutes, seconds float64
   if matches[1] != "" {
      hours, _ = strconv.ParseFloat(matches[1], 64)
   }
   if matches[2] != "" {
      minutes, _ = strconv.ParseFloat(matches[2], 64)
   }
   if matches[3] != "" {
      seconds, _ = strconv.ParseFloat(matches[3], 64)
   }

   return hours*3600 + minutes*60 + seconds, nil
}

func replacePlaceholders(template string, number, bandwidth int, repID string, time int64) string {
   re := regexp.MustCompile(`\$(RepresentationID|Number|Bandwidth|Time)(?:%0(\d+)d)?\$`)
   return re.ReplaceAllStringFunc(template, func(match string) string {
      parts := re.FindStringSubmatch(match)
      if len(parts) < 2 {
         return match
      }

      var value string
      switch parts[1] {
      case "RepresentationID":
         value = repID
      case "Number":
         if len(parts) > 2 && parts[2] != "" {
            width, _ := strconv.Atoi(parts[2])
            value = fmt.Sprintf("%0*d", width, number)
         } else {
            value = strconv.Itoa(number)
         }
      case "Bandwidth":
         value = strconv.Itoa(bandwidth)
      case "Time":
         if len(parts) > 2 && parts[2] != "" {
            width, _ := strconv.Atoi(parts[2])
            value = fmt.Sprintf("%0*d", width, time)
         } else {
            value = strconv.FormatInt(time, 10)
         }
      }

      return value
   })
}
