package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "math"
   "net/url"
   "os"
   "strconv"
   "strings"
)

// MPD structures
type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   BaseURL                   string   `xml:"BaseURL"`
   Period                    []Period `xml:"Period"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
}

type Period struct {
   BaseURL       string          `xml:"BaseURL"`
   AdaptationSet []AdaptationSet `xml:"AdaptationSet"`
   Duration      string          `xml:"duration,attr"`
   ID            string          `xml:"id,attr"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   Representation  []Representation `xml:"Representation"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   Bandwidth       string           `xml:"bandwidth,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
}

type SegmentTemplate struct {
   Media                  string           `xml:"media,attr"`
   Initialization         string           `xml:"initialization,attr"`
   Duration               string           `xml:"duration,attr"`
   Timescale              string           `xml:"timescale,attr"`
   StartNumber            string           `xml:"startNumber,attr"`
   EndNumber              string           `xml:"endNumber,attr"`
   PresentationTimeOffset string           `xml:"presentationTimeOffset,attr"`
   SegmentTimeline        *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T string `xml:"t,attr"`
   D string `xml:"d,attr"`
   R string `xml:"r,attr"`
}

type SegmentList struct {
   Duration       string          `xml:"duration,attr"`
   Timescale      string          `xml:"timescale,attr"`
   Initialization *Initialization `xml:"Initialization"`
   SegmentURL     []SegmentURL    `xml:"SegmentURL"`
}

type SegmentURL struct {
   Media      string `xml:"media,attr"`
   MediaRange string `xml:"mediaRange,attr"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
   Range     string `xml:"range,attr"`
}

type SegmentBase struct {
   Initialization *Initialization `xml:"Initialization"`
   IndexRange     string          `xml:"indexRange,attr"`
}

func main() {
   if len(os.Args) < 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }

   mpdFilePath := os.Args[1]
   mpdBaseURL := "http://test.test/test.mpd"

   // Read MPD file
   xmlData, err := ioutil.ReadFile(mpdFilePath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
      os.Exit(1)
   }

   // Parse MPD
   var mpd MPD
   err = xml.Unmarshal(xmlData, &mpd)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing XML: %v\n", err)
      os.Exit(1)
   }

   // Process and resolve URLs
   result := make(map[string][]string)

   for _, period := range mpd.Period {
      // Use Period duration if available, otherwise fall back to MPD duration
      periodDuration := period.Duration
      if periodDuration == "" {
         periodDuration = mpd.MediaPresentationDuration
      }

      for _, adaptationSet := range period.AdaptationSet {
         for _, representation := range adaptationSet.Representation {
            urls := resolveRepresentationURLs(
               mpdBaseURL,
               mpd.BaseURL,
               period.BaseURL,
               adaptationSet.BaseURL,
               representation,
               adaptationSet.SegmentTemplate,
               adaptationSet.SegmentList,
               periodDuration,
            )
            // Append to existing URLs for this representation ID
            result[representation.ID] = append(result[representation.ID], urls...)
         }
      }
   }

   // Output JSON
   jsonOutput, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error creating JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonOutput))
}

func resolveRepresentationURLs(
   mpdURL string,
   mpdBaseURL string,
   periodBaseURL string,
   adaptationSetBaseURL string,
   representation Representation,
   adaptationSetTemplate *SegmentTemplate,
   adaptationSetList *SegmentList,
   duration string,
) []string {
   var urls []string

   // Build base URL hierarchy
   baseURL := resolveBaseURL(mpdURL, mpdBaseURL, periodBaseURL, adaptationSetBaseURL, representation.BaseURL)

   // Priority: Representation > AdaptationSet
   template := representation.SegmentTemplate
   if template == nil {
      template = adaptationSetTemplate
   }

   list := representation.SegmentList
   if list == nil {
      list = adaptationSetList
   }

   if template != nil {
      urls = resolveSegmentTemplate(baseURL, template, representation.ID, representation.Bandwidth, duration)
   } else if list != nil {
      urls = resolveSegmentList(baseURL, list)
   } else if representation.BaseURL != "" {
      // Single segment representation
      urls = append(urls, baseURL)
   }

   return urls
}

func resolveBaseURL(mpdURL string, mpdBaseURL, periodBaseURL, adaptationSetBaseURL, representationBaseURL string) string {
   // Start with the MPD URL itself
   baseURL := mpdURL

   // Apply hierarchy of BaseURLs using only net/url.URL.ResolveReference
   if mpdBaseURL != "" {
      baseURL = resolveURL(baseURL, mpdBaseURL)
   }
   if periodBaseURL != "" {
      baseURL = resolveURL(baseURL, periodBaseURL)
   }
   if adaptationSetBaseURL != "" {
      baseURL = resolveURL(baseURL, adaptationSetBaseURL)
   }
   if representationBaseURL != "" {
      baseURL = resolveURL(baseURL, representationBaseURL)
   }

   return baseURL
}

func resolveURL(baseURL, relativeURL string) string {
   base, err := url.Parse(baseURL)
   if err != nil {
      return relativeURL
   }

   rel, err := url.Parse(relativeURL)
   if err != nil {
      return baseURL
   }

   resolved := base.ResolveReference(rel)
   return resolved.String()
}

func resolveSegmentTemplate(baseURL string, template *SegmentTemplate, representationID, bandwidth, duration string) []string {
   var urls []string

   // Add initialization segment if present
   if template.Initialization != "" {
      initURL := template.Initialization
      initURL = strings.ReplaceAll(initURL, "$RepresentationID$", representationID)
      initURL = strings.ReplaceAll(initURL, "$Bandwidth$", bandwidth)
      fullInitURL := resolveURL(baseURL, initURL)
      urls = append(urls, fullInitURL)
   }

   if template.SegmentTimeline != nil {
      // Timeline-based segments
      segmentURLs := resolveTimelineSegments(baseURL, template, representationID, bandwidth)
      urls = append(urls, segmentURLs...)
   } else if template.EndNumber != "" {
      // Use explicit endNumber if provided
      segmentURLs := resolveNumberRangeSegments(baseURL, template, representationID, bandwidth)
      urls = append(urls, segmentURLs...)
   } else if template.Duration != "" {
      // Duration-based segments
      segmentURLs := resolveDurationSegments(baseURL, template, representationID, bandwidth, duration)
      urls = append(urls, segmentURLs...)
   }

   return urls
}

func resolveTimelineSegments(baseURL string, template *SegmentTemplate, representationID, bandwidth string) []string {
   var urls []string

   if template.SegmentTimeline == nil {
      return urls
   }

   segmentNumber := int(parseInt(template.StartNumber, 1))
   currentTime := parseInt(template.PresentationTimeOffset, 0)

   for _, s := range template.SegmentTimeline.S {
      duration := parseInt(s.D, 0)
      repeat := int(parseInt(s.R, 0))

      if s.T != "" {
         currentTime = parseInt(s.T, currentTime)
      }

      for i := 0; i <= repeat; i++ {
         if template.Media != "" {
            url := template.Media
            url = strings.ReplaceAll(url, "$RepresentationID$", representationID)
            url = strings.ReplaceAll(url, "$Bandwidth$", bandwidth)
            url = strings.ReplaceAll(url, "$Number$", strconv.Itoa(segmentNumber))
            url = strings.ReplaceAll(url, "$Time$", strconv.FormatInt(currentTime, 10))

            // Handle format strings
            url = replaceFormatStrings(url, segmentNumber, currentTime)

            fullURL := resolveURL(baseURL, url)
            urls = append(urls, fullURL)
         }

         segmentNumber++
         currentTime += duration
      }
   }

   return urls
}

func resolveNumberRangeSegments(baseURL string, template *SegmentTemplate, representationID, bandwidth string) []string {
   var urls []string

   startNumber := int(parseInt(template.StartNumber, 1))
   endNumber := int(parseInt(template.EndNumber, -1))

   if endNumber < 0 {
      return urls
   }

   for segmentNumber := startNumber; segmentNumber <= endNumber; segmentNumber++ {
      if template.Media != "" {
         url := template.Media
         url = strings.ReplaceAll(url, "$RepresentationID$", representationID)
         url = strings.ReplaceAll(url, "$Bandwidth$", bandwidth)
         url = strings.ReplaceAll(url, "$Number$", strconv.Itoa(segmentNumber))

         // For number-based templates, $Time$ is typically calculated as:
         // (segmentNumber - startNumber) * duration
         if strings.Contains(url, "$Time$") && template.Duration != "" {
            duration := parseInt(template.Duration, 0)
            presentationTimeOffset := parseInt(template.PresentationTimeOffset, 0)
            segmentTime := int64(segmentNumber-startNumber)*duration - presentationTimeOffset
            url = strings.ReplaceAll(url, "$Time$", strconv.FormatInt(segmentTime, 10))
         }

         // Handle format strings
         url = replaceFormatStrings(url, segmentNumber, 0)

         fullURL := resolveURL(baseURL, url)
         urls = append(urls, fullURL)
      }
   }

   return urls
}

func resolveDurationSegments(baseURL string, template *SegmentTemplate, representationID, bandwidth, duration string) []string {
   var urls []string

   segmentDuration := parseInt(template.Duration, 0)
   if segmentDuration == 0 {
      return urls
   }

   timescale := parseInt(template.Timescale, 1)
   startNumber := int(parseInt(template.StartNumber, 1))
   presentationTimeOffset := parseInt(template.PresentationTimeOffset, 0)

   // Parse duration (Period duration or MPD duration)
   totalDurationSeconds := parseDuration(duration)
   if totalDurationSeconds == 0 {
      // Default to generating 100 segments if duration unknown
      totalDurationSeconds = float64(segmentDuration * 100 / timescale)
   }

   // Calculate number of segments using ceil(PeriodDurationInSeconds * timescale / duration)
   numSegments := int(math.Ceil(totalDurationSeconds * float64(timescale) / float64(segmentDuration)))

   for i := 0; i < numSegments; i++ {
      segmentNumber := startNumber + i
      segmentTime := int64(i)*segmentDuration - presentationTimeOffset

      if template.Media != "" {
         url := template.Media
         url = strings.ReplaceAll(url, "$RepresentationID$", representationID)
         url = strings.ReplaceAll(url, "$Bandwidth$", bandwidth)
         url = strings.ReplaceAll(url, "$Number$", strconv.Itoa(segmentNumber))
         url = strings.ReplaceAll(url, "$Time$", strconv.FormatInt(segmentTime, 10))

         // Handle format strings
         url = replaceFormatStrings(url, segmentNumber, segmentTime)

         fullURL := resolveURL(baseURL, url)
         urls = append(urls, fullURL)
      }
   }

   return urls
}

func resolveSegmentList(baseURL string, list *SegmentList) []string {
   var urls []string

   // Add initialization segment if present
   if list.Initialization != nil && list.Initialization.SourceURL != "" {
      fullURL := resolveURL(baseURL, list.Initialization.SourceURL)
      urls = append(urls, fullURL)
   }

   for _, segment := range list.SegmentURL {
      if segment.Media != "" {
         fullURL := resolveURL(baseURL, segment.Media)
         urls = append(urls, fullURL)
      }
   }

   return urls
}

func replaceFormatStrings(url string, number int, time int64) string {
   // Handle $Number%0Xd$ format
   if strings.Contains(url, "$Number%") {
      parts := strings.Split(url, "$Number%")
      if len(parts) > 1 {
         endParts := strings.Split(parts[1], "$")
         if len(endParts) > 0 {
            format := endParts[0]
            if strings.HasSuffix(format, "d") {
               widthStr := strings.TrimSuffix(format[1:], "d")
               width := parseInt(widthStr, 0)
               if width > 0 {
                  formatted := fmt.Sprintf("%0*d", width, number)
                  url = strings.ReplaceAll(url, "$Number%"+format+"$", formatted)
               }
            }
         }
      }
   }

   // Handle $Time%0Xd$ format
   if strings.Contains(url, "$Time%") {
      parts := strings.Split(url, "$Time%")
      if len(parts) > 1 {
         endParts := strings.Split(parts[1], "$")
         if len(endParts) > 0 {
            format := endParts[0]
            if strings.HasSuffix(format, "d") {
               widthStr := strings.TrimSuffix(format[1:], "d")
               width := parseInt(widthStr, 0)
               if width > 0 {
                  formatted := fmt.Sprintf("%0*d", width, time)
                  url = strings.ReplaceAll(url, "$Time%"+format+"$", formatted)
               }
            }
         }
      }
   }

   return url
}

func parseInt(s string, defaultValue int64) int64 {
   if s == "" {
      return defaultValue
   }
   val, err := strconv.ParseInt(s, 10, 64)
   if err != nil {
      return defaultValue
   }
   return val
}

func parseDuration(duration string) float64 {
   // Parse ISO 8601 duration (PT1H2M10.5S format)
   if !strings.HasPrefix(duration, "PT") {
      return 0
   }

   duration = strings.TrimPrefix(duration, "PT")
   totalSeconds := 0.0

   // Hours
   if idx := strings.Index(duration, "H"); idx != -1 {
      hours, _ := strconv.ParseFloat(duration[:idx], 64)
      totalSeconds += hours * 3600
      duration = duration[idx+1:]
   }

   // Minutes
   if idx := strings.Index(duration, "M"); idx != -1 {
      minutes, _ := strconv.ParseFloat(duration[:idx], 64)
      totalSeconds += minutes * 60
      duration = duration[idx+1:]
   }

   // Seconds
   if idx := strings.Index(duration, "S"); idx != -1 {
      seconds, _ := strconv.ParseFloat(duration[:idx], 64)
      totalSeconds += seconds
   }

   return totalSeconds
}
