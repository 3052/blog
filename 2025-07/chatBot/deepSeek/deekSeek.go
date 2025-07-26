package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "net/url"
   "os"
   "strconv"
   "strings"
   "time"
)

const originalMPDURL = "http://test.test/test.mpd"

type MPD struct {
   XMLName    xml.Name `xml:"MPD"`
   Periods    []Period `xml:"Period"`
   BaseURL    string   `xml:"BaseURL"`
   Type       string   `xml:"type,attr"`
   MediaPDur  string   `xml:"mediaPresentationDuration,attr"`
   MinBufTime string   `xml:"minBufferTime,attr"`
}

type Period struct {
   XMLName       xml.Name     `xml:"Period"`
   AdaptationSet []Adaptation `xml:"AdaptationSet"`
   BaseURL       string       `xml:"BaseURL"`
}

type Adaptation struct {
   XMLName        xml.Name         `xml:"AdaptationSet"`
   Representation []Representation `xml:"Representation"`
   BaseURL        string           `xml:"BaseURL"`
   Segment        *Segment         `xml:"SegmentTemplate"`
}

type Representation struct {
   XMLName   xml.Name `xml:"Representation"`
   ID        string   `xml:"id,attr"`
   Bandwidth string   `xml:"bandwidth,attr"`
   BaseURL   string   `xml:"BaseURL"`
   Segment   *Segment `xml:"SegmentTemplate"`
}

type Segment struct {
   XMLName     xml.Name         `xml:"SegmentTemplate"`
   Initial     string           `xml:"initialization,attr"`
   Media       string           `xml:"media,attr"`
   StartNumber string           `xml:"startNumber,attr"`
   Timescale   string           `xml:"timescale,attr"`
   Duration    string           `xml:"duration,attr"`
   Timeline    *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   Segments []TimelineSegment `xml:"S"`
}

type TimelineSegment struct {
   T string `xml:"t,attr"` // start time
   D string `xml:"d,attr"` // duration
   R string `xml:"r,attr"` // repeat count
}

func main() {
   if len(os.Args) < 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <path_to_mpd_file>\n", os.Args[0])
      os.Exit(1)
   }

   mpdFile := os.Args[1]
   data, err := os.ReadFile(mpdFile)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD XML: %v\n", err)
      os.Exit(1)
   }

   originalURL, err := url.Parse(originalMPDURL)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing original MPD URL: %v\n", err)
      os.Exit(1)
   }

   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      for _, adaptation := range period.AdaptationSet {
         currentBase := originalURL
         if mpd.BaseURL != "" {
            currentBase = resolveURL(currentBase, mpd.BaseURL)
         }
         if period.BaseURL != "" {
            currentBase = resolveURL(currentBase, period.BaseURL)
         }
         if adaptation.BaseURL != "" {
            currentBase = resolveURL(currentBase, adaptation.BaseURL)
         }

         for _, representation := range adaptation.Representation {
            segments := []string{}
            repBase := currentBase

            if representation.BaseURL != "" {
               repBase = resolveURL(repBase, representation.BaseURL)
            }

            segment := representation.Segment
            if segment == nil {
               segment = adaptation.Segment
            }

            if segment != nil {
               // Process initialization segment
               if segment.Initial != "" {
                  initialURL := resolveURL(repBase, expandTemplate(
                     segment.Initial,
                     representation.ID,
                     0, // number
                     0, // time
                  ))
                  segments = append(segments, initialURL.String())
               }

               // Process media segments
               if segment.Media != "" && segment.Timescale != "" {
                  timescale, err := strconv.Atoi(segment.Timescale)
                  if err != nil {
                     fmt.Fprintf(os.Stderr, "Error parsing timescale: %v\n", err)
                     os.Exit(1)
                  }

                  if segment.Timeline != nil {
                     // SegmentTimeline mode
                     segments = append(segments, processSegmentTimeline(
                        repBase,
                        segment.Media,
                        representation.ID,
                        segment.Timeline,
                        timescale,
                     )...)
                  } else if segment.Duration != "" {
                     // Simple startNumber/duration mode
                     startNumber := 1
                     if segment.StartNumber != "" {
                        startNumber, err = strconv.Atoi(segment.StartNumber)
                        if err != nil {
                           fmt.Fprintf(os.Stderr, "Error parsing startNumber: %v\n", err)
                           os.Exit(1)
                        }
                     }

                     segmentDuration, err := strconv.Atoi(segment.Duration)
                     if err != nil {
                        fmt.Fprintf(os.Stderr, "Error parsing segment duration: %v\n", err)
                        os.Exit(1)
                     }

                     // Estimate total segments from media duration
                     mediaDuration, err := parseMediaDuration(mpd.MediaPDur)
                     if err != nil {
                        fmt.Fprintf(os.Stderr, "Error parsing media duration: %v\n", err)
                        os.Exit(1)
                     }

                     segmentDurSec := float64(segmentDuration) / float64(timescale)
                     totalSegments := int(mediaDuration.Seconds() / segmentDurSec)
                     if float64(totalSegments)*segmentDurSec < mediaDuration.Seconds() {
                        totalSegments++
                     }

                     for i := startNumber; i < startNumber+totalSegments; i++ {
                        timeValue := (i - startNumber) * segmentDuration
                        mediaURL := resolveURL(repBase, expandTemplate(
                           segment.Media,
                           representation.ID,
                           i,
                           timeValue,
                        ))
                        segments = append(segments, mediaURL.String())
                     }
                  }
               }
            }

            if len(segments) > 0 {
               result[representation.ID] = segments
            }
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

func expandTemplate(template, repID string, number, time int) string {
   result := template

   // Handle $RepresentationID$
   result = strings.ReplaceAll(result, "$RepresentationID$", repID)

   // Handle $Number$ with padding
   result = expandTemplateVariable(result, "$Number", number)

   // Handle $Time$ with padding
   result = expandTemplateVariable(result, "$Time", time)

   return result
}

func expandTemplateVariable(template, prefix string, value int) string {
   if !strings.Contains(template, prefix) {
      return template
   }

   // Handle padded version (e.g., $Number%04d$)
   if idx := strings.Index(template, prefix+"%0"); idx != -1 {
      endIdx := strings.Index(template[idx:], "$") + idx
      if endIdx > idx {
         padSpec := template[idx+len(prefix)+1 : endIdx]
         padLen, err := strconv.Atoi(padSpec)
         if err == nil {
            format := fmt.Sprintf("%%0%dd", padLen)
            return strings.Replace(template, prefix+"%0"+padSpec+"$", fmt.Sprintf(format, value), 1)
         }
      }
   }

   // Handle simple version (e.g., $Number$)
   return strings.ReplaceAll(template, prefix+"$", strconv.Itoa(value))
}

func resolveURL(base *url.URL, relative string) *url.URL {
   relURL, err := url.Parse(relative)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing URL: %v\n", err)
      os.Exit(1)
   }
   return base.ResolveReference(relURL)
}

func parseMediaDuration(dur string) (time.Duration, error) {
   if !strings.HasPrefix(dur, "PT") {
      return 0, fmt.Errorf("duration must start with PT")
   }

   dur = strings.TrimPrefix(dur, "PT")
   var hours, minutes int
   var seconds float64
   var err error

   // Parse hours
   if idx := strings.Index(dur, "H"); idx != -1 {
      hours, err = strconv.Atoi(dur[:idx])
      if err != nil {
         return 0, fmt.Errorf("invalid hours format")
      }
      dur = dur[idx+1:]
   }

   // Parse minutes
   if idx := strings.Index(dur, "M"); idx != -1 {
      minutes, err = strconv.Atoi(dur[:idx])
      if err != nil {
         return 0, fmt.Errorf("invalid minutes format")
      }
      dur = dur[idx+1:]
   }

   // Parse seconds
   if dur != "" {
      if strings.HasSuffix(dur, "S") {
         dur = dur[:len(dur)-1]
      }
      seconds, err = strconv.ParseFloat(dur, 64)
      if err != nil {
         return 0, fmt.Errorf("invalid seconds format")
      }
   }

   return time.Duration(hours)*time.Hour +
      time.Duration(minutes)*time.Minute +
      time.Duration(seconds*float64(time.Second)), nil
}

func processSegmentTimeline(base *url.URL, mediaTemplate, repID string, timeline *SegmentTimeline, timescale int) []string {
   var segments []string
   segmentNumber := 1
   currentTime := 0

   for _, s := range timeline.Segments {
      duration, err := strconv.Atoi(s.D)
      if err != nil {
         fmt.Fprintf(os.Stderr, "Error parsing segment duration: %v\n", err)
         continue
      }

      // If @t is present, it overrides the current time
      if s.T != "" {
         t, err := strconv.Atoi(s.T)
         if err != nil {
            fmt.Fprintf(os.Stderr, "Error parsing segment start time: %v\n", err)
            continue
         }
         currentTime = t
      }

      repeat := 0
      if s.R != "" {
         repeat, err = strconv.Atoi(s.R)
         if err != nil {
            fmt.Fprintf(os.Stderr, "Error parsing segment repeat count: %v\n", err)
            continue
         }
      }

      // Generate segments for this timeline entry
      for i := 0; i <= repeat; i++ {
         mediaURL := resolveURL(base, expandTemplate(
            mediaTemplate,
            repID,
            segmentNumber,
            currentTime,
         ))
         segments = append(segments, mediaURL.String())
         segmentNumber++
         currentTime += duration
      }
   }

   return segments
}
