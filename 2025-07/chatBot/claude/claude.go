package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "net/url"
   "os"
   "strconv"
   "strings"
)

// MPD structure definitions
type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   BaseURL                   string   `xml:"BaseURL"`
   Periods                   []Period `xml:"Period"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
}

type Period struct {
   XMLName        xml.Name        `xml:"Period"`
   BaseURL        string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   XMLName         xml.Name         `xml:"AdaptationSet"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   XMLName         xml.Name         `xml:"Representation"`
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   XMLName         xml.Name         `xml:"SegmentTemplate"`
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   Duration        int              `xml:"duration,attr"`
   Timescale       int              `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   XMLName xml.Name `xml:"SegmentTimeline"`
   S       []S      `xml:"S"`
}

type S struct {
   XMLName xml.Name `xml:"S"`
   T       *uint64  `xml:"t,attr"`
   D       uint64   `xml:"d,attr"`
   R       *int     `xml:"r,attr"`
}

type SegmentList struct {
   XMLName        xml.Name        `xml:"SegmentList"`
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
   XMLName   xml.Name `xml:"Initialization"`
   SourceURL string   `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   XMLName xml.Name `xml:"SegmentURL"`
   Media   string   `xml:"media,attr"`
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }

   mpdFile := os.Args[1]

   // Parse MPD
   mpd, err := parseMPD(mpdFile)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD: %v\n", err)
      os.Exit(1)
   }

   // Extract segments
   result, err := extractSegments(mpd, mpdFile)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error extracting segments: %v\n", err)
      os.Exit(1)
   }

   // Output JSON
   output, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(output))
}

func parseMPD(filePath string) (*MPD, error) {
   file, err := os.Open(filePath)
   if err != nil {
      return nil, fmt.Errorf("failed to open MPD file: %v", err)
   }
   defer file.Close()

   var mpd MPD
   decoder := xml.NewDecoder(file)
   if err := decoder.Decode(&mpd); err != nil {
      return nil, fmt.Errorf("failed to decode XML: %v", err)
   }

   return &mpd, nil
}

func extractSegments(mpd *MPD, mpdFile string) (map[string][]string, error) {
   result := make(map[string][]string)

   // Use base URL from specification
   baseURL := "http://test.test/test.mpd"

   for _, period := range mpd.Periods {
      for _, adaptationSet := range period.AdaptationSets {
         for _, representation := range adaptationSet.Representations {
            urls, err := extractRepresentationSegments(mpd, &period, &adaptationSet, &representation, baseURL)
            if err != nil {
               return nil, fmt.Errorf("error extracting segments for representation %s: %v", representation.ID, err)
            }
            if len(urls) > 0 {
               result[representation.ID] = urls
            }
         }
      }
   }

   return result, nil
}

func extractRepresentationSegments(mpd *MPD, period *Period, adaptationSet *AdaptationSet, representation *Representation, baseURL string) ([]string, error) {
   var urls []string

   // Check for SegmentList first
   segmentList := representation.SegmentList
   if segmentList == nil {
      segmentList = adaptationSet.SegmentList
   }

   if segmentList != nil {
      // Resolve base URLs hierarchically for SegmentList processing
      resolvedBaseURL := resolveBaseURL(baseURL, mpd.BaseURL, period.BaseURL, adaptationSet.BaseURL, representation.BaseURL)
      return extractFromSegmentList(segmentList, resolvedBaseURL)
   }

   // Check for SegmentTemplate
   segmentTemplate := representation.SegmentTemplate
   if segmentTemplate == nil {
      segmentTemplate = adaptationSet.SegmentTemplate
   }

   if segmentTemplate != nil {
      // Resolve base URLs hierarchically for SegmentTemplate processing
      resolvedBaseURL := resolveBaseURL(baseURL, mpd.BaseURL, period.BaseURL, adaptationSet.BaseURL, representation.BaseURL)
      return extractFromSegmentTemplate(segmentTemplate, representation.ID, resolvedBaseURL)
   }

   // If no SegmentList or SegmentTemplate, check if Representation has only BaseURL
   if representation.BaseURL != "" {
      // Resolve BaseURL hierarchically but don't include representation.BaseURL twice
      parentBaseURL := resolveBaseURL(baseURL, mpd.BaseURL, period.BaseURL, adaptationSet.BaseURL)
      segmentURL, err := resolveURL(parentBaseURL, representation.BaseURL)
      if err != nil {
         return nil, err
      }
      urls = append(urls, segmentURL)
   }

   return urls, nil
}

func extractFromSegmentList(segmentList *SegmentList, baseURL string) ([]string, error) {
   var urls []string

   // Add initialization URL if present
   if segmentList.Initialization != nil && segmentList.Initialization.SourceURL != "" {
      initURL, err := resolveURL(baseURL, segmentList.Initialization.SourceURL)
      if err != nil {
         return nil, err
      }
      urls = append(urls, initURL)
   }

   // Add segment URLs
   for _, segmentURL := range segmentList.SegmentURLs {
      segURL, err := resolveURL(baseURL, segmentURL.Media)
      if err != nil {
         return nil, err
      }
      urls = append(urls, segURL)
   }

   return urls, nil
}

func extractFromSegmentTemplate(template *SegmentTemplate, representationID, baseURL string) ([]string, error) {
   var urls []string

   // Add initialization URL if present
   if template.Initialization != "" {
      initTemplate := substituteTemplateVariables(template.Initialization, representationID, 0, 0)
      initURL, err := resolveURL(baseURL, initTemplate)
      if err != nil {
         return nil, err
      }
      urls = append(urls, initURL)
   }

   // Generate segment URLs
   if template.SegmentTimeline != nil {
      // Use SegmentTimeline
      segmentURLs, err := generateSegmentURLsFromTimeline(template, representationID, baseURL)
      if err != nil {
         return nil, err
      }
      urls = append(urls, segmentURLs...)
   } else if template.Duration > 0 {
      // Use duration-based approach
      segmentURLs, err := generateSegmentURLsFromDuration(template, representationID, baseURL)
      if err != nil {
         return nil, err
      }
      urls = append(urls, segmentURLs...)
   }

   return urls, nil
}

func generateSegmentURLsFromTimeline(template *SegmentTemplate, representationID, baseURL string) ([]string, error) {
   var urls []string
   var currentTime uint64 = 0
   segmentNumber := template.StartNumber
   if segmentNumber == 0 {
      segmentNumber = 1
   }

   for _, s := range template.SegmentTimeline.S {
      // Use t attribute if present, otherwise continue from current time
      if s.T != nil {
         currentTime = *s.T
      }

      repeatCount := 1
      if s.R != nil {
         repeatCount = *s.R + 1
      }

      for i := 0; i < repeatCount; i++ {
         // Check endNumber limit
         if template.EndNumber > 0 && segmentNumber > template.EndNumber {
            break
         }

         mediaTemplate := substituteTemplateVariables(template.Media, representationID, segmentNumber, currentTime)
         segmentURL, err := resolveURL(baseURL, mediaTemplate)
         if err != nil {
            return nil, err
         }
         urls = append(urls, segmentURL)

         currentTime += s.D
         segmentNumber++
      }

      if template.EndNumber > 0 && segmentNumber > template.EndNumber {
         break
      }
   }

   return urls, nil
}

func generateSegmentURLsFromDuration(template *SegmentTemplate, representationID, baseURL string) ([]string, error) {
   var urls []string
   segmentNumber := template.StartNumber
   if segmentNumber == 0 {
      segmentNumber = 1
   }

   timescale := template.Timescale
   if timescale == 0 {
      timescale = 1
   }

   currentTime := uint64(0)

   // Generate segments until endNumber (if specified)
   for {
      if template.EndNumber > 0 && segmentNumber > template.EndNumber {
         break
      }

      mediaTemplate := substituteTemplateVariables(template.Media, representationID, segmentNumber, currentTime)
      segmentURL, err := resolveURL(baseURL, mediaTemplate)
      if err != nil {
         return nil, err
      }
      urls = append(urls, segmentURL)

      currentTime += uint64(template.Duration)
      segmentNumber++

      // If no endNumber specified, we need some limit - use a reasonable default
      if template.EndNumber == 0 && segmentNumber > 1000 {
         break
      }
   }

   return urls, nil
}

func substituteTemplateVariables(template, representationID string, number int, time uint64) string {
   result := template
   result = strings.ReplaceAll(result, "$RepresentationID$", representationID)
   result = strings.ReplaceAll(result, "$Number$", strconv.Itoa(number))
   result = strings.ReplaceAll(result, "$Time$", strconv.FormatUint(time, 10))

   // Handle formatted number patterns like $Number%05d$
   if strings.Contains(result, "$Number%") {
      // Find and replace formatted number patterns
      for {
         start := strings.Index(result, "$Number%")
         if start == -1 {
            break
         }
         end := strings.Index(result[start:], "$")
         if end == -1 {
            break
         }
         end += start + 1

         formatStr := result[start+8 : end-1] // Extract format between % and $
         formatted := fmt.Sprintf("%"+formatStr, number)
         result = result[:start] + formatted + result[end:]
      }
   }

   return result
}

func resolveBaseURL(base string, baseURLs ...string) string {
   result := base

   for _, baseURL := range baseURLs {
      if baseURL != "" {
         var err error
         result, err = resolveURL(result, baseURL)
         if err != nil {
            // If resolution fails, just append
            if !strings.HasSuffix(result, "/") && !strings.HasPrefix(baseURL, "/") {
               result += "/"
            }
            result += baseURL
         }
      }
   }

   return result
}

func resolveURL(base, relative string) (string, error) {
   if relative == "" {
      return base, nil
   }

   // If relative is absolute URL, return it
   if strings.HasPrefix(relative, "http://") || strings.HasPrefix(relative, "https://") {
      return relative, nil
   }

   baseURL, err := url.Parse(base)
   if err != nil {
      return "", err
   }

   relativeURL, err := url.Parse(relative)
   if err != nil {
      return "", err
   }

   resolvedURL := baseURL.ResolveReference(relativeURL)
   return resolvedURL.String(), nil
}
