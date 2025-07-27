package main

import (
   "encoding/json"
   "log"
   "encoding/xml"
   "errors"
   "fmt"
   "math"
   "net/url"
   "os"
   "strconv"
   "strings"
)

type MPD struct {
   XMLName      xml.Name     `xml:"MPD"`
   BaseURL      string       `xml:"BaseURL"`
   Periods      []Period     `xml:"Period"`
   MediaPresDur string       `xml:"mediaPresentationDuration,attr"`
   Type         string       `xml:"type,attr"`
}

type Period struct {
   BaseURL         string          `xml:"BaseURL"`
   AdaptationSets  []AdaptationSet `xml:"AdaptationSet"`
   Duration        string          `xml:"duration,attr"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   Representations []Representation `xml:"Representation"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type Representation struct {
   ID             string           `xml:"id,attr"`
   Bandwidth      int              `xml:"bandwidth,attr"`
   BaseURL        string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   BaseURL        string           `xml:"BaseURL"`
   Initialization string           `xml:"initialization,attr"`
   Media          string           `xml:"media,attr"`
   StartNumber    *int             `xml:"startNumber,attr"` // Pointer to distinguish absent from 0
   Timescale      int              `xml:"timescale,attr"`
   Duration       int              `xml:"duration,attr"`
   EndNumber      *int             `xml:"endNumber,attr"` // Pointer for consistency
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   Segments []SegmentTimelineEntry `xml:"S"`
}

type SegmentTimelineEntry struct {
   Duration int `xml:"d,attr"`
   Repeat   int `xml:"r,attr"`
   Time     int `xml:"t,attr"`
}

type SegmentList struct {
   BaseURL     string       `xml:"BaseURL"`
   SegmentURLs []SegmentURL `xml:"SegmentURL"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

func parseDuration(durationStr string) (float64, error) {
   if !strings.HasPrefix(durationStr, "PT") {
      return 0, errors.New("invalid duration format")
   }

   durationStr = strings.ToLower(durationStr[2:])
   var hours, minutes, seconds float64
   var err error

   if strings.Contains(durationStr, "h") {
      parts := strings.Split(durationStr, "h")
      hours, err = strconv.ParseFloat(parts[0], 64)
      if err != nil {
         return 0, err
      }
      durationStr = parts[1]
   }

   if strings.Contains(durationStr, "m") {
      parts := strings.Split(durationStr, "m")
      minutes, err = strconv.ParseFloat(parts[0], 64)
      if err != nil {
         return 0, err
      }
      durationStr = parts[1]
   }

   if strings.Contains(durationStr, "s") {
      parts := strings.Split(durationStr, "s")
      seconds, err = strconv.ParseFloat(parts[0], 64)
      if err != nil {
         return 0, err
      }
   }

   return hours*3600 + minutes*60 + seconds, nil
}

func resolveURL(base, rel string) (string, error) {
   if base == "" {
      return rel, nil
   }
   if rel == "" {
      return base, nil
   }

   baseURL, err := url.Parse(base)
   if err != nil {
      return "", fmt.Errorf("invalid base URL: %w", err)
   }

   relURL, err := url.Parse(rel)
   if err != nil {
      return "", fmt.Errorf("invalid relative URL: %w", err)
   }

   return baseURL.ResolveReference(relURL).String(), nil
}

func getEffectiveBaseURL(mpdBaseURL, periodBaseURL, adaptationSetBaseURL, representationBaseURL, segmentBaseURL string) (string, error) {
   base := mpdBaseURL
   
   var err error
   if periodBaseURL != "" {
      base, err = resolveURL(base, periodBaseURL)
      if err != nil {
         return "", fmt.Errorf("period resolution failed: %w", err)
      }
   }
   
   if adaptationSetBaseURL != "" {
      base, err = resolveURL(base, adaptationSetBaseURL)
      if err != nil {
         return "", fmt.Errorf("adaptation set resolution failed: %w", err)
      }
   }
   
   if representationBaseURL != "" {
      base, err = resolveURL(base, representationBaseURL)
      if err != nil {
         return "", fmt.Errorf("representation resolution failed: %w", err)
      }
   }
   
   if segmentBaseURL != "" {
      base, err = resolveURL(base, segmentBaseURL)
      if err != nil {
         return "", fmt.Errorf("segment resolution failed: %w", err)
      }
   }
   
   return base, nil
}

func expandTemplate(template, representationID string, number, timeValue int) string {
   var result strings.Builder
   i := 0

   for i < len(template) {
      if i+len("$RepresentationID$") <= len(template) && 
         template[i:i+len("$RepresentationID$")] == "$RepresentationID$" {
         result.WriteString(representationID)
         i += len("$RepresentationID$")
         continue
      }

      if i+len("$Number") <= len(template) && template[i:i+len("$Number")] == "$Number" {
         i = handleNumberTemplate(template, i, number, &result)
         continue
      }

      if i+len("$Time") <= len(template) && template[i:i+len("$Time")] == "$Time" {
         i = handleTimeTemplate(template, i, timeValue, &result)
         continue
      }

      result.WriteByte(template[i])
      i++
   }

   return result.String()
}

func handleNumberTemplate(template string, i int, number int, result *strings.Builder) int {
   start := i
   i += len("$Number")

   if i < len(template) && template[i] == '%' {
      i++
      padStart := i

      for i < len(template) && template[i] != 'd' {
         i++
      }
      if i >= len(template) || template[i] != 'd' {
         result.WriteString(template[start:i])
         return i
      }

      padSpec := template[padStart:i]
      i++

      if i >= len(template) || template[i] != '$' {
         result.WriteString(template[start:i])
         return i
      }
      i++

      padLen, err := strconv.Atoi(padSpec)
      if err != nil {
         padLen = 0
      }
      result.WriteString(fmt.Sprintf("%0*d", padLen, number))
      return i
   }

   if i < len(template) && template[i] == '$' {
      i++
      result.WriteString(strconv.Itoa(number))
      return i
   }

   result.WriteString(template[start:i])
   return i
}

func handleTimeTemplate(template string, i int, timeValue int, result *strings.Builder) int {
   start := i
   i += len("$Time")

   if i < len(template) && template[i] == '%' {
      i++
      padStart := i

      for i < len(template) && template[i] != 'd' {
         i++
      }
      if i >= len(template) || template[i] != 'd' {
         result.WriteString(template[start:i])
         return i
      }

      padSpec := template[padStart:i]
      i++

      if i >= len(template) || template[i] != '$' {
         result.WriteString(template[start:i])
         return i
      }
      i++

      padLen, err := strconv.Atoi(padSpec)
      if err != nil {
         padLen = 0
      }
      result.WriteString(fmt.Sprintf("%0*d", padLen, timeValue))
      return i
   }

   if i < len(template) && template[i] == '$' {
      i++
      result.WriteString(strconv.Itoa(timeValue))
      return i
   }

   result.WriteString(template[start:i])
   return i
}

func processSegmentTimeline(template *SegmentTemplate, representationID, baseURL string) ([]string, error) {
   var segments []string

   if template.SegmentTimeline == nil || len(template.SegmentTimeline.Segments) == 0 {
      return nil, errors.New("no segment timeline entries")
   }

   currentTime := 0
   if len(template.SegmentTimeline.Segments) > 0 && template.SegmentTimeline.Segments[0].Time != 0 {
      currentTime = template.SegmentTimeline.Segments[0].Time
   }

   // Handle startNumber - absent (default 1) vs explicit 0
   segmentNumber := 1
   if template.StartNumber != nil {
      segmentNumber = *template.StartNumber
   }

   for _, entry := range template.SegmentTimeline.Segments {
      duration := entry.Duration
      repeat := entry.Repeat

      mediaURL := expandTemplate(template.Media, representationID, segmentNumber, currentTime)
      absoluteURL, err := resolveURL(baseURL, mediaURL)
      if err != nil {
         return nil, err
      }
      segments = append(segments, absoluteURL)
      segmentNumber++
      currentTime += duration

      for i := 0; i < repeat; i++ {
         mediaURL := expandTemplate(template.Media, representationID, segmentNumber, currentTime)
         absoluteURL, err := resolveURL(baseURL, mediaURL)
         if err != nil {
            return nil, err
         }
         segments = append(segments, absoluteURL)
         segmentNumber++
         currentTime += duration
      }
   }

   return segments, nil
}

func processNumberBasedSegments(template *SegmentTemplate, representationID, baseURL string, periodDuration float64) ([]string, error) {
   var segments []string

   timescale := template.Timescale
   if timescale == 0 {
      timescale = 1
   }

   if template.Duration == 0 {
      return nil, errors.New("invalid duration")
   }

   // Handle startNumber - absent (default 1) vs explicit 0
   start := 1
   if template.StartNumber != nil {
      start = *template.StartNumber
   }

   // Handle endNumber - absent vs explicit
   var end int
   if template.EndNumber != nil {
      end = *template.EndNumber
   } else {
      if periodDuration > 0 {
         segmentCount := math.Ceil(periodDuration * float64(timescale) / float64(template.Duration))
         if segmentCount < 1 {
            segmentCount = 1
         }
         end = start + int(segmentCount) - 1
      } else {
         if template.Initialization != "" {
            initURL := expandTemplate(template.Initialization, representationID, 0, 0)
            absoluteURL, err := resolveURL(baseURL, initURL)
            if err != nil {
               return nil, err
            }
            return []string{absoluteURL}, nil
         }
         return nil, errors.New("cannot determine segment count")
      }
   }

   for i := start; i <= end; i++ {
      mediaURL := expandTemplate(template.Media, representationID, i, 0)
      absoluteURL, err := resolveURL(baseURL, mediaURL)
      if err != nil {
         return nil, err
      }
      segments = append(segments, absoluteURL)
   }

   return segments, nil
}

func processSegmentList(segmentList *SegmentList, representationID, baseURL string) ([]string, error) {
   var segments []string

   for _, seg := range segmentList.SegmentURLs {
      if seg.Media == "" {
         continue
      }
      absoluteURL, err := resolveURL(baseURL, seg.Media)
      if err != nil {
         return nil, err
      }
      segments = append(segments, absoluteURL)
   }

   return segments, nil
}

func processRepresentation(adaptationSet *AdaptationSet, representation *Representation, mpdBaseURL, periodBaseURL, adaptationSetBaseURL string, periodDuration float64) ([]string, error) {
   template := representation.SegmentTemplate
   if template == nil {
      template = adaptationSet.SegmentTemplate
   }

   list := representation.SegmentList
   if list == nil {
      list = adaptationSet.SegmentList
   }

   baseURL, err := getEffectiveBaseURL(
      mpdBaseURL,
      periodBaseURL,
      adaptationSetBaseURL,
      representation.BaseURL,
      "",
   )
   if err != nil {
      return nil, err
   }

   if template != nil {
      templateBaseURL, err := getEffectiveBaseURL(
         mpdBaseURL,
         periodBaseURL,
         adaptationSetBaseURL,
         representation.BaseURL,
         template.BaseURL,
      )
      if err != nil {
         return nil, err
      }

      if template.SegmentTimeline != nil {
         return processSegmentTimeline(template, representation.ID, templateBaseURL)
      } else if template.Media != "" && template.Duration != 0 {
         return processNumberBasedSegments(template, representation.ID, templateBaseURL, periodDuration)
      }
   }

   if list != nil {
      listBaseURL, err := getEffectiveBaseURL(
         mpdBaseURL,
         periodBaseURL,
         adaptationSetBaseURL,
         representation.BaseURL,
         list.BaseURL,
      )
      if err != nil {
         return nil, err
      }
      return processSegmentList(list, representation.ID, listBaseURL)
   }

   if baseURL != "" {
      return []string{baseURL}, nil
   }

   return nil, errors.New("no segment information found")
}

func processMPD(mpd *MPD, originalMPDURL string) (map[string][]string, error) {
   result := make(map[string][]string)

   mpdBaseURL := originalMPDURL
   if mpd.BaseURL != "" {
      resolved, err := resolveURL(originalMPDURL, mpd.BaseURL)
      if err != nil {
         return nil, err
      }
      mpdBaseURL = resolved
   }

   for _, period := range mpd.Periods {
      periodDuration, err := parseDuration(period.Duration)
      if err != nil {
         return nil, fmt.Errorf("invalid period duration: %w", err)
      }

      periodBaseURL := mpdBaseURL
      if period.BaseURL != "" {
         resolved, err := resolveURL(mpdBaseURL, period.BaseURL)
         if err != nil {
            return nil, err
         }
         periodBaseURL = resolved
      }

      for _, adaptationSet := range period.AdaptationSets {
         adaptationSetBaseURL := periodBaseURL
         if adaptationSet.BaseURL != "" {
            resolved, err := resolveURL(periodBaseURL, adaptationSet.BaseURL)
            if err != nil {
               return nil, err
            }
            adaptationSetBaseURL = resolved
         }

         for _, representation := range adaptationSet.Representations {
            segments, err := processRepresentation(
               &adaptationSet,
               &representation,
               mpdBaseURL,
               periodBaseURL,
               adaptationSetBaseURL,
               periodDuration,
            )
            if err != nil {
               return nil, fmt.Errorf("representation %s: %w", representation.ID, err)
            }

            if existing, ok := result[representation.ID]; ok {
               result[representation.ID] = append(existing, segments...)
            } else {
               result[representation.ID] = segments
            }
         }
      }
   }

   return result, nil
}

func main() {
   if len(os.Args) < 2 {
      fmt.Fprintln(os.Stderr, "Usage: mpd-expander <path-to-mpd-file>")
      os.Exit(1)
   }

   mpdPath := os.Args[1]
   mpdData, err := os.ReadFile(mpdPath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(mpdData, &mpd); err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD: %v\n", err)
      os.Exit(1)
   }

   originalMPDURL := "http://test.test/test.mpd"
   segments, err := processMPD(&mpd, originalMPDURL)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error processing MPD: %v\n", err)
      os.Exit(1)
   }

   jsonOutput, err := json.MarshalIndent(segments, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error generating JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonOutput))
}
