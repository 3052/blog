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

// MPD represents the root element of the MPD XML file
type MPD struct {
   XMLName         xml.Name         `xml:"MPD"`
   BaseURL         string           `xml:"BaseURL"`
   Periods         []Period         `xml:"Period"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

// Period represents a Period element in the MPD
type Period struct {
   XMLName         xml.Name         `xml:"Period"`
   BaseURL         string           `xml:"BaseURL"`
   AdaptationSets  []AdaptationSet  `xml:"AdaptationSet"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Duration        string           `xml:"duration,attr"`
}

// AdaptationSet represents an AdaptationSet element
type AdaptationSet struct {
   XMLName         xml.Name         `xml:"AdaptationSet"`
   BaseURL         string           `xml:"BaseURL"`
   Representations []Representation `xml:"Representation"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

// Representation represents a Representation element
type Representation struct {
   XMLName         xml.Name         `xml:"Representation"`
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

// SegmentTemplate represents a SegmentTemplate element
type SegmentTemplate struct {
   XMLName         xml.Name         `xml:"SegmentTemplate"`
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"`
   Duration        int              `xml:"duration,attr"`
   Timescale       int              `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// SegmentTimeline represents a SegmentTimeline element
type SegmentTimeline struct {
   XMLName xml.Name `xml:"SegmentTimeline"`
   S       []S      `xml:"S"`
}

// S represents an S element in the SegmentTimeline
type S struct {
   XMLName xml.Name `xml:"S"`
   T       int      `xml:"t,attr"`
   D       int      `xml:"d,attr"`
   R       int      `xml:"r,attr"`
}

// SegmentList represents a SegmentList element
type SegmentList struct {
   XMLName        xml.Name        `xml:"SegmentList"`
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

// Initialization represents the Initialization element for SegmentList
type Initialization struct {
   XMLName   xml.Name `xml:"Initialization"`
   SourceURL string   `xml:"sourceURL,attr"`
}

// SegmentURL represents a SegmentURL element
type SegmentURL struct {
   XMLName xml.Name `xml:"SegmentURL"`
   Media   string   `xml:"media,attr"`
}

// Function to resolve a relative URL against a base URL
func resolveURL(base, relative string) (string, error) {
   baseURL, err := url.Parse(base)
   if err != nil {
      return "", err
   }
   relativeURL, err := url.Parse(relative)
   if err != nil {
      return "", err
   }
   return baseURL.ResolveReference(relativeURL).String(), nil
}

// parseDuration converts an ISO 8601 duration string (e.g., "PT1H53M46.040S") to seconds
func parseDuration(isoDuration string) (float64, error) {
   re := regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?`)
   matches := re.FindStringSubmatch(isoDuration)
   if matches == nil {
      return 0, fmt.Errorf("invalid ISO 8601 duration format: %s", isoDuration)
   }

   var seconds float64

   if matches[1] != "" {
      h, err := strconv.ParseFloat(matches[1], 64)
      if err != nil {
         return 0, err
      }
      seconds += h * 3600
   }

   if matches[2] != "" {
      m, err := strconv.ParseFloat(matches[2], 64)
      if err != nil {
         return 0, err
      }
      seconds += m * 60
   }

   if matches[3] != "" {
      s, err := strconv.ParseFloat(matches[3], 64)
      if err != nil {
         return 0, err
      }
      seconds += s
   }

   return seconds, nil
}

func main() {
   log.SetOutput(os.Stderr)
   log.SetFlags(0)

   if len(os.Args) < 2 {
      log.Println("Usage: go run main.go <mpd_file_path>")
      os.Exit(1)
   }

   mpdFilePath := os.Args[1]
   mpdData, err := ioutil.ReadFile(mpdFilePath)
   if err != nil {
      log.Printf("Error reading MPD file: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   err = xml.Unmarshal(mpdData, &mpd)
   if err != nil {
      log.Printf("Error unmarshalling MPD XML: %v\n", err)
      os.Exit(1)
   }

   segmentURLs := make(map[string][]string)

   initialMPDURL := "http://test.test/test.mpd"
   baseMPDURL, err := url.Parse(initialMPDURL)
   if err != nil {
      log.Printf("Error parsing initial MPD URL: %v\n", err)
      os.Exit(1)
   }
   mpdBaseURL := baseMPDURL.String()

   if mpd.BaseURL != "" {
      mpdBaseURL, err = resolveURL(mpdBaseURL, mpd.BaseURL)
      if err != nil {
         log.Printf("Error resolving MPD BaseURL: %v\n", err)
         os.Exit(1)
      }
   }

   for _, period := range mpd.Periods {
      periodBaseURL := mpdBaseURL
      if period.BaseURL != "" {
         periodBaseURL, err = resolveURL(periodBaseURL, period.BaseURL)
         if err != nil {
            log.Printf("Error resolving Period BaseURL: %v\n", err)
            os.Exit(1)
         }
      }

      var periodDuration float64
      if period.Duration != "" {
         periodDuration, err = parseDuration(period.Duration)
         if err != nil {
            log.Printf("Error parsing Period duration: %v\n", err)
            periodDuration = 0
         }
      }

      for _, adaptationSet := range period.AdaptationSets {
         adaptationSetBaseURL := periodBaseURL
         if adaptationSet.BaseURL != "" {
            adaptationSetBaseURL, err = resolveURL(adaptationSetBaseURL, adaptationSet.BaseURL)
            if err != nil {
               log.Printf("Error resolving AdaptationSet BaseURL: %v\n", err)
               os.Exit(1)
            }
         }

         var asSegmentTemplate *SegmentTemplate
         if adaptationSet.SegmentTemplate != nil {
            asSegmentTemplate = adaptationSet.SegmentTemplate
         } else if period.SegmentTemplate != nil {
            asSegmentTemplate = period.SegmentTemplate
         } else {
            asSegmentTemplate = mpd.SegmentTemplate
         }

         var asSegmentList *SegmentList
         if adaptationSet.SegmentList != nil {
            asSegmentList = adaptationSet.SegmentList
         }

         for _, representation := range adaptationSet.Representations {
            representationBaseURL := adaptationSetBaseURL
            if representation.BaseURL != "" {
               representationBaseURL, err = resolveURL(representationBaseURL, representation.BaseURL)
               if err != nil {
                  log.Printf("Error resolving Representation BaseURL: %v\n", err)
                  continue
               }
            }

            var repSegments []string
            var st *SegmentTemplate
            var sl *SegmentList

            if representation.SegmentTemplate != nil {
               st = representation.SegmentTemplate
            } else {
               st = asSegmentTemplate
            }

            if representation.SegmentList != nil {
               sl = representation.SegmentList
            } else {
               sl = asSegmentList
            }

            if st != nil {
               if st.Timescale == 0 {
                  st.Timescale = 1
               }

               var startNumber int
               if st.StartNumber != nil {
                  startNumber = *st.StartNumber
               } else {
                  startNumber = 1 // Default to 1 if missing
               }

               if st.Initialization != "" {
                  initURL := replacePlaceholders(st.Initialization, representation.ID, 0)
                  resolvedInitURL, err := resolveURL(representationBaseURL, initURL)
                  if err != nil {
                     log.Printf("Error resolving initialization segment URL for representation %s: %v\n", representation.ID, err)
                     continue
                  }
                  repSegments = append(repSegments, resolvedInitURL)
               }
               if st.SegmentTimeline != nil {
                  time := 0
                  currentNumber := startNumber
                  for _, s := range st.SegmentTimeline.S {
                     if s.T > 0 {
                        time = s.T
                     }
                     for i := 0; i <= s.R; i++ {
                        segmentURL := replacePlaceholders(st.Media, representation.ID, currentNumber)
                        segmentURL = strings.Replace(segmentURL, "$Time$", strconv.Itoa(time), -1)
                        resolvedSegmentURL, err := resolveURL(representationBaseURL, segmentURL)
                        if err != nil {
                           log.Printf("Error resolving segment URL for representation %s: %v\n", representation.ID, err)
                           continue
                        }
                        repSegments = append(repSegments, resolvedSegmentURL)
                        time += s.D
                        currentNumber++
                     }
                  }
               } else {
                  endNumber := st.EndNumber
                  if endNumber == 0 && periodDuration > 0 && st.Duration > 0 {
                     periodDurationInTimescale := periodDuration * float64(st.Timescale)
                     segmentCount := math.Ceil(periodDurationInTimescale / float64(st.Duration))
                     endNumber = startNumber + int(segmentCount) - 1
                  } else if endNumber == 0 {
                     endNumber = startNumber + 10
                  }
                  for i := startNumber; i <= endNumber; i++ {
                     segmentURL := replacePlaceholders(st.Media, representation.ID, i)
                     resolvedSegmentURL, err := resolveURL(representationBaseURL, segmentURL)
                     if err != nil {
                        log.Printf("Error resolving segment URL for representation %s: %v\n", representation.ID, err)
                        continue
                     }
                     repSegments = append(repSegments, resolvedSegmentURL)
                  }
               }
            } else if sl != nil {
               if sl.Initialization != nil && sl.Initialization.SourceURL != "" {
                  resolvedInitURL, err := resolveURL(representationBaseURL, sl.Initialization.SourceURL)
                  if err != nil {
                     log.Printf("Error resolving SegmentList initialization URL for representation %s: %v\n", representation.ID, err)
                     continue
                  }
                  repSegments = append(repSegments, resolvedInitURL)
               }
               for _, sURL := range sl.SegmentURLs {
                  resolvedURL, err := resolveURL(representationBaseURL, sURL.Media)
                  if err != nil {
                     log.Printf("Error resolving SegmentList URL for representation %s: %v\n", representation.ID, err)
                     continue
                  }
                  repSegments = append(repSegments, resolvedURL)
               }
            } else if representation.BaseURL != "" {
               repSegments = append(repSegments, representationBaseURL)
            }

            if len(repSegments) > 0 {
               segmentURLs[representation.ID] = append(segmentURLs[representation.ID], repSegments...)
            }
         }
      }
   }

   jsonData, err := json.MarshalIndent(segmentURLs, "", "  ")
   if err != nil {
      log.Printf("Error marshalling JSON: %v\n", err)
      os.Exit(1)
   }

   fmt.Println(string(jsonData))
}

var numberFormatRegex = regexp.MustCompile(`\$Number(%[0-9]+d)\$`)

func replacePlaceholders(template string, repID string, number int) string {
   s := template
   s = strings.Replace(s, "$RepresentationID$", repID, -1)

   matches := numberFormatRegex.FindStringSubmatch(s)
   if len(matches) > 1 {
      format := matches[1]
      formattedNumber := fmt.Sprintf(format, number)
      s = strings.Replace(s, matches[0], formattedNumber, -1)
   } else {
      s = strings.Replace(s, "$Number$", strconv.Itoa(number), -1)
   }

   return s
}
