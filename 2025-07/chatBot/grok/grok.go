package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "net/url"
   "os"
   "regexp"
   "strconv"
   "strings"
)

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   Periods []Period `xml:"Period"`
   BaseURL string   `xml:"BaseURL"`
}

type Period struct {
   Duration       string          `xml:"duration,attr"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
   BaseURL        string          `xml:"BaseURL"`
}

type AdaptationSet struct {
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
   BaseURL         string           `xml:"BaseURL"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   BaseURL         string           `xml:"BaseURL"`
}

type SegmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   Timescale       string           `xml:"timescale,attr"`
   Duration        string           `xml:"duration,attr"`
   StartNumber     *string          `xml:"startNumber,attr"`
   EndNumber       *string          `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   Segments []Segment `xml:"S"`
}

type Segment struct {
   T *int64 `xml:"t,attr"` // Nullable start time
   D int64  `xml:"d,attr"` // Duration
   R *int64 `xml:"r,attr"` // Repeat count (optional, default 0)
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }

   mpdPath := os.Args[1]
   data, err := os.ReadFile(mpdPath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD XML: %v\n", err)
      os.Exit(1)
   }

   defaultBaseURL, err := url.Parse("http://test.test/test.mpd")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing default base URL: %v\n", err)
      os.Exit(1)
   }

   result := make(map[string][]string)
   for _, period := range mpd.Periods {
      for _, adaptSet := range period.AdaptationSets {
         for _, rep := range adaptSet.Representations {
            var segmentTemplate *SegmentTemplate
            if rep.SegmentTemplate != nil {
               segmentTemplate = rep.SegmentTemplate
            } else if adaptSet.SegmentTemplate != nil {
               segmentTemplate = adaptSet.SegmentTemplate
            } else {
               fmt.Fprintf(os.Stderr, "No SegmentTemplate found for representation %s\n", rep.ID)
               os.Exit(1)
            }

            // Resolve BaseURL for this Representation
            baseURL, err := resolveBaseURL(defaultBaseURL, mpd.BaseURL, period.BaseURL, adaptSet.BaseURL, rep.BaseURL)
            if err != nil {
               fmt.Fprintf(os.Stderr, "Error resolving BaseURL for representation %s: %v\n", rep.ID, err)
               os.Exit(1)
            }

            urls, err := expandSegmentTemplate(baseURL, segmentTemplate, rep.ID, period.Duration)
            if err != nil {
               fmt.Fprintf(os.Stderr, "Error expanding segment template for representation %s: %v\n", rep.ID, err)
               os.Exit(1)
            }
            result[rep.ID] = urls
         }
      }
   }

   output, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(output))
}

// resolveBaseURL resolves the BaseURL hierarchy: Representation > AdaptationSet > Period > MPD > default
func resolveBaseURL(defaultBaseURL *url.URL, mpdBaseURL, periodBaseURL, adaptSetBaseURL, repBaseURL string) (*url.URL, error) {
   currentBaseURL := defaultBaseURL

   // Process BaseURLs from MPD to Representation, with lower levels overriding
   for i, baseURLStr := range []string{mpdBaseURL, periodBaseURL, adaptSetBaseURL, repBaseURL} {
      if baseURLStr == "" {
         continue
      }
      parsedURL, err := url.Parse(baseURLStr)
      if err != nil {
         return nil, fmt.Errorf("error parsing BaseURL at level %d: %v", i, err)
      }
      // If the BaseURL is absolute, use it directly
      if parsedURL.IsAbs() {
         currentBaseURL = parsedURL
      } else {
         // Resolve relative BaseURL against the current BaseURL
         currentBaseURL = currentBaseURL.ResolveReference(parsedURL)
      }
   }

   return currentBaseURL, nil
}

func expandSegmentTemplate(baseURL *url.URL, template *SegmentTemplate, repID string, periodDuration string) ([]string, error) {
   var urls []string

   // Parse format specifiers for $Number$ and $Time$ in media
   numberFormat, err := parseFormatSpecifier(template.Media, "Number")
   if err != nil {
      return nil, fmt.Errorf("error parsing Number format in media: %v", err)
   }
   timeFormat, err := parseFormatSpecifier(template.Media, "Time")
   if err != nil {
      return nil, fmt.Errorf("error parsing Time format in media: %v", err)
   }

   // Handle initialization segment
   if template.Initialization != "" {
      if strings.Contains(template.Initialization, "$Number") || strings.Contains(template.Initialization, "$Time") {
         return nil, fmt.Errorf("initialization attribute must not contain $Number$ or $Time$ placeholders")
      }
      initURL := strings.ReplaceAll(template.Initialization, "$RepresentationID$", repID)
      absURL, err := resolveURL(baseURL, initURL)
      if err != nil {
         return nil, fmt.Errorf("error resolving initialization URL: %v", err)
      }
      urls = append(urls, absURL)
   }

   // Parse startNumber (default to 1 if absent)
   startNumber := 1
   if template.StartNumber != nil {
      var err error
      startNumber, err = strconv.Atoi(*template.StartNumber)
      if err != nil {
         return nil, fmt.Errorf("error parsing startNumber: %v", err)
      }
   }

   // Check for SegmentTimeline mode
   if template.SegmentTimeline != nil {
      if template.EndNumber != nil {
         return nil, fmt.Errorf("SegmentTimeline and endNumber are mutually exclusive")
      }
      return expandSegmentTimeline(baseURL, template, repID, startNumber, numberFormat, timeFormat)
   }

   // Parse timescale and duration for simple mode
   timescale := 1
   if template.Timescale != "" {
      var err error
      timescale, err = strconv.Atoi(template.Timescale)
      if err != nil {
         return nil, fmt.Errorf("error parsing timescale: %v", err)
      }
      if timescale <= 0 {
         return nil, fmt.Errorf("invalid timescale: %d", timescale)
      }
   }

   segmentDuration := 0
   if template.Duration != "" {
      var err error
      segmentDuration, err = strconv.Atoi(template.Duration)
      if err != nil {
         return nil, fmt.Errorf("error parsing segment duration: %v", err)
      }
      if segmentDuration <= 0 {
         return nil, fmt.Errorf("invalid segment duration: %d", segmentDuration)
      }
   } else if template.EndNumber == nil && !strings.Contains(template.Media, "$Time$") {
      return nil, fmt.Errorf("segment duration is required but absent when endNumber is not provided")
   }

   // Validate Time placeholder usage
   if strings.Contains(template.Media, "$Time$") && segmentDuration == 0 {
      return nil, fmt.Errorf("Time placeholder requires segment duration in simple mode")
   }

   // Handle endNumber mode
   if template.EndNumber != nil {
      endNumber, err := strconv.Atoi(*template.EndNumber)
      if err != nil {
         return nil, fmt.Errorf("error parsing endNumber: %v", err)
      }
      if endNumber < startNumber {
         return nil, fmt.Errorf("endNumber (%d) must be greater than or equal to startNumber (%d)", endNumber, startNumber)
      }
      for i := startNumber; i <= endNumber; i++ {
         mediaURL := strings.ReplaceAll(template.Media, "$RepresentationID$", repID)
         // Format Number with padding
         numberStr := fmt.Sprintf("%d", i)
         if numberFormat != "" {
            numberStr = fmt.Sprintf(numberFormat, i)
         }
         mediaURL = strings.ReplaceAll(mediaURL, "$Number"+numberFormat+"$", numberStr)
         // Format Time with padding
         if strings.Contains(template.Media, "$Time") {
            time := int64((i - startNumber) * segmentDuration)
            timeStr := fmt.Sprintf("%d", time)
            if timeFormat != "" {
               timeStr = fmt.Sprintf(timeFormat, time)
            }
            mediaURL = strings.ReplaceAll(mediaURL, "$Time"+timeFormat+"$", timeStr)
         }
         absURL, err := resolveURL(baseURL, mediaURL)
         if err != nil {
            return nil, fmt.Errorf("error resolving media URL for segment %d: %v", i, err)
         }
         urls = append(urls, absURL)
      }
      return urls, nil
   }

   // Fallback to duration-based segment count calculation
   segmentDurationSeconds := float64(segmentDuration) / float64(timescale)
   periodDurationSeconds := 3600.0 // Default 1 hour
   if periodDuration != "" {
      var err error
      periodDurationSeconds, err = parseISODuration(periodDuration)
      if err != nil {
         return nil, fmt.Errorf("error parsing period duration: %v", err)
      }
   }

   segmentCount := int(periodDurationSeconds / segmentDurationSeconds)
   if segmentCount <= 0 {
      segmentCount = 1 // Ensure at least one segment
   }

   for i := 0; i < segmentCount; i++ {
      segmentNumber := startNumber + i
      mediaURL := strings.ReplaceAll(template.Media, "$RepresentationID$", repID)
      // Format Number with padding
      numberStr := fmt.Sprintf("%d", segmentNumber)
      if numberFormat != "" {
         numberStr = fmt.Sprintf(numberFormat, segmentNumber)
      }
      mediaURL = strings.ReplaceAll(mediaURL, "$Number"+numberFormat+"$", numberStr)
      // Format Time with padding
      if strings.Contains(template.Media, "$Time") {
         time := int64(i * segmentDuration)
         timeStr := fmt.Sprintf("%d", time)
         if timeFormat != "" {
            timeStr = fmt.Sprintf(timeFormat, time)
         }
         mediaURL = strings.ReplaceAll(mediaURL, "$Time"+timeFormat+"$", timeStr)
      }
      absURL, err := resolveURL(baseURL, mediaURL)
      if err != nil {
         return nil, fmt.Errorf("error resolving media URL for segment %d: %v", segmentNumber, err)
      }
      urls = append(urls, absURL)
   }

   return urls, nil
}

func expandSegmentTimeline(baseURL *url.URL, template *SegmentTemplate, repID string, startNumber int, numberFormat, timeFormat string) ([]string, error) {
   var urls []string
   currentNumber := startNumber
   currentTime := int64(0) // Initialize timeline

   for i, segment := range template.SegmentTimeline.Segments {
      if segment.D <= 0 {
         return nil, fmt.Errorf("invalid segment duration: %d", segment.D)
      }
      repeat := int64(0)
      if segment.R != nil {
         if *segment.R < 0 {
            return nil, fmt.Errorf("invalid repeat count: %d", *segment.R)
         }
         repeat = *segment.R
      }

      // Determine start time for this <S> element
      segmentStartTime := currentTime
      if segment.T != nil {
         if *segment.T < 0 {
            return nil, fmt.Errorf("invalid start time: %d", *segment.T)
         }
         segmentStartTime = *segment.T
         // For subsequent elements, validate @t consistency
         if i > 0 && *segment.T < currentTime {
            return nil, fmt.Errorf("start time %d is less than previous end time %d", *segment.T, currentTime)
         }
      }

      // Generate segments for this <S> element
      for j := int64(0); j <= repeat; j++ {
         mediaURL := strings.ReplaceAll(template.Media, "$RepresentationID$", repID)
         // Format Number with padding
         numberStr := fmt.Sprintf("%d", currentNumber)
         if numberFormat != "" {
            numberStr = fmt.Sprintf(numberFormat, currentNumber)
         }
         mediaURL = strings.ReplaceAll(mediaURL, "$Number"+numberFormat+"$", numberStr)
         // Format Time with padding
         time := segmentStartTime + j*segment.D
         timeStr := fmt.Sprintf("%d", time)
         if timeFormat != "" {
            timeStr = fmt.Sprintf(timeFormat, time)
         }
         mediaURL = strings.ReplaceAll(mediaURL, "$Time"+timeFormat+"$", timeStr)
         absURL, err := resolveURL(baseURL, mediaURL)
         if err != nil {
            return nil, fmt.Errorf("error resolving media URL for segment %d: %v", currentNumber, err)
         }
         urls = append(urls, absURL)
         currentNumber++
      }

      // Update currentTime for the next <S> element
      currentTime = segmentStartTime + (1+repeat)*segment.D
   }

   return urls, nil
}

func resolveURL(baseURL *url.URL, relative string) (string, error) {
   relURL, err := url.Parse(relative)
   if err != nil {
      return "", fmt.Errorf("error parsing relative URL: %v", err)
   }
   absURL := baseURL.ResolveReference(relURL)
   return absURL.String(), nil
}

// parseISODuration parses a simple ISO 8601 duration (e.g., PTnS) into seconds
func parseISODuration(duration string) (float64, error) {
   if !strings.HasPrefix(duration, "PT") {
      return 0, fmt.Errorf("invalid duration format: %s", duration)
   }
   duration = strings.TrimPrefix(duration, "PT")
   if strings.HasSuffix(duration, "S") {
      duration = strings.TrimSuffix(duration, "S")
      seconds, err := strconv.ParseFloat(duration, 64)
      if err != nil {
         return 0, fmt.Errorf("error parsing duration value: %v", err)
      }
      return seconds, nil
   }
   return 0, fmt.Errorf("unsupported duration format: %s", duration)
}

// parseFormatSpecifier extracts the %0xd format from $Number%0xd$ or $Time%0xd$
func parseFormatSpecifier(template, placeholder string) (string, error) {
   if template == "" {
      return "", nil
   }
   re := regexp.MustCompile(`\$` + placeholder + `(%0[0-9]+d)\$`)
   matches := re.FindStringSubmatch(template)
   if len(matches) > 1 {
      widthStr := matches[1][2 : len(matches[1])-1] // Extract number from %0xd
      width, err := strconv.Atoi(widthStr)
      if err != nil {
         return "", fmt.Errorf("invalid %s format specifier: %v", placeholder, err)
      }
      if width <= 0 {
         return "", fmt.Errorf("invalid %s format width: %d", placeholder, width)
      }
      return "%0" + widthStr + "d", nil
   }
   if strings.Contains(template, "$"+placeholder+"$") {
      return "", nil // Plain placeholder without format
   }
   return "", nil // No placeholder
}
