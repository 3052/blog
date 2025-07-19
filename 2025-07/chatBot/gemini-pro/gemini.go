package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "net/url"
   "os"
   "strconv"
   "strings"
)

// MPD represents the root element of a DASH manifest.
type MPD struct {
   XMLName    xml.Name `xml:"MPD"`
   BaseURL    string   `xml:"BaseURL"`
   Periods    []Period `xml:"Period"`
   Type       string   `xml:"type,attr"`
   MediaPresentationDuration string `xml:"mediaPresentationDuration,attr"`
}

// Period represents a Period element in the MPD.
type Period struct {
   XMLName        xml.Name        `xml:"Period"`
   BaseURL        string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

// AdaptationSet represents an AdaptationSet element in the MPD.
type AdaptationSet struct {
   XMLName          xml.Name         `xml:"AdaptationSet"`
   Representations  []Representation `xml:"Representation"`
   SegmentTemplate  *SegmentTemplate `xml:"SegmentTemplate"`
   ContentType      string           `xml:"contentType,attr"`
}

// Representation represents a Representation element in the MPD.
type Representation struct {
   XMLName         xml.Name         `xml:"Representation"`
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentBase     *SegmentBase     `xml:"SegmentBase"`
   Bandwidth       string           `xml:"bandwidth,attr"`
}

// SegmentBase represents a SegmentBase element in the MPD.
type SegmentBase struct {
   XMLName xml.Name `xml:"SegmentBase"`
}

// SegmentTemplate represents a SegmentTemplate element in the MPD.
type SegmentTemplate struct {
   XMLName         xml.Name         `xml:"SegmentTemplate"`
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     string           `xml:"startNumber,attr"`
   EndNumber       string           `xml:"endNumber,attr"`
   Duration        string           `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline represents a SegmentTimeline element in the MPD.
type SegmentTimeline struct {
   XMLName xml.Name `xml:"SegmentTimeline"`
   Segments []S `xml:"S"`
}

// S represents an S element within a SegmentTimeline.
type S struct {
   XMLName xml.Name `xml:"S"`
   T       *int64   `xml:"t,attr"`
   D       int64    `xml:"d,attr"`
   R       *int     `xml:"r,attr"`
}

func main() {
   if len(os.Args) < 2 {
      fmt.Println("Usage: go run <script_name>.go <path_to_mpd_file>")
      os.Exit(1)
   }
   filePath := os.Args[1]

   xmlFile, err := os.Open(filePath)
   if err != nil {
      fmt.Printf("Error opening file: %v\n", err)
      os.Exit(1)
   }
   defer xmlFile.Close()

   byteValue, _ := ioutil.ReadAll(xmlFile)

   var mpd MPD
   err = xml.Unmarshal(byteValue, &mpd)
   if err != nil {
      fmt.Printf("Error unmarshalling XML: %v\n", err)
      os.Exit(1)
   }

   mpdURL, _ := url.Parse("http://test.test/test.mpd")
   output := make(map[string][]string)

   var periodBaseURL *url.URL
   if mpd.BaseURL != "" {
       baseURL, _ := url.Parse(mpd.BaseURL)
       periodBaseURL = mpdURL.ResolveReference(baseURL)
   } else {
      periodBaseURL = mpdURL
   }
   

   for _, period := range mpd.Periods {
      currentPeriodBaseURL := periodBaseURL
      if period.BaseURL != "" {
         baseURL, _ := url.Parse(period.BaseURL)
         currentPeriodBaseURL = periodBaseURL.ResolveReference(baseURL)
      }

      for _, adaptationSet := range period.AdaptationSets {
         for _, representation := range adaptationSet.Representations {
            var segmentURLs []string
            
            var representationBaseURL *url.URL
            if representation.BaseURL != "" {
               baseURL, _ := url.Parse(representation.BaseURL)
               representationBaseURL = currentPeriodBaseURL.ResolveReference(baseURL)
            } else {
               representationBaseURL = currentPeriodBaseURL
            }
            
            segmentTemplate := representation.SegmentTemplate
            if segmentTemplate == nil {
               segmentTemplate = adaptationSet.SegmentTemplate
            }
            
            if representation.SegmentBase != nil {
               segmentURLs = append(segmentURLs, representationBaseURL.String())
            } else if segmentTemplate != nil {
               if segmentTemplate.SegmentTimeline != nil {
                  segmentCount := 0
                  for _, s := range segmentTemplate.SegmentTimeline.Segments {
                     segmentCount++
                     if s.R != nil {
                        segmentCount += *s.R
                     }
                  }
                  startNumber := 1
                  if segmentTemplate.StartNumber != "" {
                     startNumber, _ = strconv.Atoi(segmentTemplate.StartNumber)
                  }

                  for i := 0; i < segmentCount; i++ {
                     segmentNumber := startNumber + i
                     mediaURL := strings.Replace(segmentTemplate.Media, "$Number$", strconv.Itoa(segmentNumber), -1)
                     mediaURL = strings.Replace(mediaURL, "$Bandwidth$", representation.Bandwidth, -1)
                     mediaURL = strings.Replace(mediaURL, "$RepresentationID$", representation.ID, -1)
                     
                     finalURL := representationBaseURL.ResolveReference(&url.URL{Path: mediaURL})
                     segmentURLs = append(segmentURLs, finalURL.String())
                  }

               } else if segmentTemplate.StartNumber != "" && (segmentTemplate.EndNumber != "" || segmentTemplate.Duration != "") {
                  start, _ := strconv.Atoi(segmentTemplate.StartNumber)
                  var end int

                  if segmentTemplate.EndNumber != "" {
                     end, _ = strconv.Atoi(segmentTemplate.EndNumber)
                  }

                  for i := start; end == 0 || i <= end; i++ {
                     mediaURL := strings.Replace(segmentTemplate.Media, "$RepresentationID$", representation.ID, -1)
                     mediaURL = strings.Replace(mediaURL, "$Number$", strconv.Itoa(i), -1)
                     mediaURL = strings.Replace(mediaURL, "$Bandwidth$", representation.Bandwidth, -1)
                     
                     finalURL := representationBaseURL.ResolveReference(&url.URL{Path: mediaURL})
                     segmentURLs = append(segmentURLs, finalURL.String())
                     if end == 0 {
                        break
                     }
                  }
               }
            } else if representation.BaseURL != "" {
               segmentURLs = append(segmentURLs, representationBaseURL.String())
            }
            output[representation.ID] = segmentURLs
         }
      }
   }

   jsonData, err := json.MarshalIndent(output, "", "  ")
   if err != nil {
      fmt.Printf("Error marshalling JSON: %v\n", err)
      os.Exit(1)
   }
   fmt.Println(string(jsonData))
}
