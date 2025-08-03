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

// The initial MPD URL used to resolve all relative BaseURLs.
const initialMPDURL = "http://test.test/test.mpd"

// Fallback for live streams (where segment count is not fixed)
const maxSegmentsForLive = 10

// Regex to find placeholders like $Number$, $Time%05d$, or $RepresentationID$.
var placeholderRegex = regexp.MustCompile(`\$([A-Za-z0-9_]+)(%.+?)?\$`)

// ## XML Data Structures ##

type MPD struct {
   XMLName                   xml.Name `xml:"MPD"`
   Type                      string   `xml:"type,attr"`
   MediaPresentationDuration string   `xml:"mediaPresentationDuration,attr"`
   BaseURL                   *BaseURL `xml:"BaseURL"`
   Periods                   []Period `xml:"Period"`
}

type Period struct {
   XMLName        xml.Name        `xml:"Period"`
   Duration       string          `xml:"duration,attr"`
   BaseURL        *BaseURL        `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   XMLName         xml.Name         `xml:"AdaptationSet"`
   Timescale       *int             `xml:"timescale,attr"`
   BaseURL         *BaseURL         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   XMLName         xml.Name         `xml:"Representation"`
   ID              string           `xml:"id,attr"`
   Timescale       *int             `xml:"timescale,attr"`
   BaseURL         *BaseURL         `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type BaseURL struct {
   Value string `xml:",chardata"`
}

type SegmentTemplate struct {
   XMLName         xml.Name         `xml:"SegmentTemplate"`
   Initialization  *string          `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   Timescale       *int             `xml:"timescale,attr"`
   Duration        *uint64          `xml:"duration,attr"`
   StartNumber     *int             `xml:"startNumber,attr"`
   EndNumber       *int             `xml:"endNumber,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   XMLName  xml.Name `xml:"SegmentTimeline"`
   Segments []S      `xml:"S"`
}

type S struct {
   XMLName  xml.Name `xml:"S"`
   Time     *uint64  `xml:"t,attr"`
   Duration uint64   `xml:"d,attr"`
   Repeat   *int     `xml:"r,attr"`
}

type SegmentList struct {
   XMLName        xml.Name        `xml:"SegmentList"`
   Initialization *Initialization `xml:"Initialization"`
   SegmentURLs    []SegmentURL    `xml:"SegmentURL"`
}

type Initialization struct {
   SourceURL string `xml:"sourceURL,attr"`
}

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

var durationRegex = regexp.MustCompile(`^PT(?:(\d+)H)?(?:(\d+)M)?(?:([\d.]+)S)?$`)

func parseISODuration(durationStr string) (float64, error) {
   if !strings.HasPrefix(durationStr, "PT") {
      return 0, fmt.Errorf("invalid duration format: %s", durationStr)
   }
   matches := durationRegex.FindStringSubmatch(durationStr)
   if matches == nil {
      return 0, fmt.Errorf("could not parse duration: %s", durationStr)
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

func main() {
   if len(os.Args) < 2 {
      log.Fatalf("Usage: go run main.go <mpd_file_path>")
   }
   mpdFilePath := os.Args[1]
   xmlFile, err := ioutil.ReadFile(mpdFilePath)
   if err != nil {
      log.Fatalf("Failed to read MPD file '%s': %v", mpdFilePath, err)
   }
   var mpd MPD
   if err := xml.Unmarshal(xmlFile, &mpd); err != nil {
      log.Fatalf("Failed to parse MPD XML: %v", err)
   }
   segmentMap := processMPD(&mpd)
   jsonOutput, err := json.MarshalIndent(segmentMap, "", "  ")
   if err != nil {
      log.Fatalf("Failed to generate JSON output: %v", err)
   }
   fmt.Println(string(jsonOutput))
}

/*
An example MPD with endNumber (`example-end-number.mpd`):

<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="static">

   <Period>
       <AdaptationSet>
           <Representation id="video_endNumber">
               <SegmentTemplate startNumber="10" endNumber="12" media="$Number$.m4s"/>
           </Representation>
       </AdaptationSet>
   </Period>

</MPD>
*/
func processMPD(mpd *MPD) map[string][]string {
   results := make(map[string][]string)
   baseURL, err := url.Parse(initialMPDURL)
   if err != nil {
      log.Fatalf("Internal error: initialMPDURL is invalid: %v", err)
   }
   mpdBase := resolveNextBase(baseURL, mpd.BaseURL)
   mpdDurationSec, err := parseISODuration(mpd.MediaPresentationDuration)
   if err != nil {
      log.Printf("Warning: Could not parse MPD mediaPresentationDuration: %v", err)
      mpdDurationSec = 0
   }
   for i := range mpd.Periods {
      period := &mpd.Periods[i]
      periodBase := resolveNextBase(mpdBase, period.BaseURL)
      effectiveDurationSec := mpdDurationSec
      if period.Duration != "" {
         periodDuration, err := parseISODuration(period.Duration)
         if err == nil {
            effectiveDurationSec = periodDuration
         } else {
            log.Printf("Warning: Could not parse Period duration '%s': %v", period.Duration, err)
         }
      }
      for j := range period.AdaptationSets {
         as := &period.AdaptationSets[j]
         asBase := resolveNextBase(periodBase, as.BaseURL)
         for k := range as.Representations {
            rep := &as.Representations[k]
            repBase := resolveNextBase(asBase, rep.BaseURL)
            template := rep.SegmentTemplate
            if template == nil {
               template = as.SegmentTemplate
            }
            list := rep.SegmentList
            if list == nil {
               list = as.SegmentList
            }
            var newURLs []string
            if template != nil {
               newURLs = generateTemplateURLs(rep, as, repBase, template, effectiveDurationSec, mpd.Type)
            } else if list != nil {
               newURLs = generateListURLs(rep.ID, repBase, list)
            } else if rep.BaseURL != nil && rep.BaseURL.Value != "" {
               newURLs = []string{repBase.String()}
            }
            if len(newURLs) > 0 {
               if existingURLs, ok := results[rep.ID]; ok {
                  results[rep.ID] = append(existingURLs, newURLs...)
               } else {
                  results[rep.ID] = newURLs
               }
            }
         }
      }
   }
   return results
}

func replacePlaceholders(template string, values map[string]interface{}) string {
   return placeholderRegex.ReplaceAllStringFunc(template, func(match string) string {
      parts := placeholderRegex.FindStringSubmatch(match)
      if len(parts) < 2 {
         return match
      }
      identifier := parts[1]
      format := "%v"
      if len(parts) > 2 && parts[2] != "" {
         format = parts[2]
      }
      if value, ok := values[identifier]; ok {
         return fmt.Sprintf(format, value)
      }
      return match
   })
}

func resolveTimescale(rep *Representation, as *AdaptationSet, tmpl *SegmentTemplate) int {
   if tmpl != nil && tmpl.Timescale != nil {
      return *tmpl.Timescale
   }
   if rep != nil && rep.Timescale != nil {
      return *rep.Timescale
   }
   if as != nil && as.Timescale != nil {
      return *as.Timescale
   }
   return 1
}

func generateListURLs(repID string, base *url.URL, list *SegmentList) []string {
   var allURLs []string
   if list.Initialization != nil && list.Initialization.SourceURL != "" {
      if resolvedURL, err := base.Parse(list.Initialization.SourceURL); err == nil {
         allURLs = append(allURLs, resolvedURL.String())
      }
   }
   for _, segmentURL := range list.SegmentURLs {
      if segmentURL.Media != "" {
         if resolvedURL, err := base.Parse(segmentURL.Media); err == nil {
            allURLs = append(allURLs, resolvedURL.String())
         }
      }
   }
   return allURLs
}

func generateTemplateURLs(rep *Representation, as *AdaptationSet, base *url.URL, template *SegmentTemplate, totalDuration float64, mpdType string) []string {
   var allURLs []string
   baseValues := map[string]interface{}{"RepresentationID": rep.ID}
   if template.Initialization != nil {
      initPath := replacePlaceholders(*template.Initialization, baseValues)
      if resolvedURL, err := base.Parse(initPath); err == nil {
         allURLs = append(allURLs, resolvedURL.String())
      }
   }
   startNumber := 1
   if template.StartNumber != nil {
      startNumber = *template.StartNumber
   }
   if template.SegmentTimeline != nil {
      segmentNumberCounter := startNumber
      var currentTime uint64 = 0
      for _, s := range template.SegmentTimeline.Segments {
         if s.Time != nil {
            currentTime = *s.Time
         }
         repeatCount := 0
         if s.Repeat != nil {
            repeatCount = *s.Repeat
            if repeatCount == -1 {
               repeatCount = maxSegmentsForLive - len(allURLs)
            }
         }
         for i := 0; i <= repeatCount; i++ {
            values := map[string]interface{}{
               "RepresentationID": rep.ID,
               "Time":             currentTime,
               "Number":           segmentNumberCounter,
            }
            mediaPath := replacePlaceholders(template.Media, values)
            if resolvedURL, err := base.Parse(mediaPath); err == nil {
               allURLs = append(allURLs, resolvedURL.String())
            }
            currentTime += s.Duration
            segmentNumberCounter++
         }
      }
   } else {
      numSegments := maxSegmentsForLive
      // Priority 1: Use endNumber to calculate segment count.
      if template.EndNumber != nil {
         numSegments = *template.EndNumber - startNumber + 1
      } else if mpdType == "static" && template.Duration != nil && totalDuration > 0 {
         // Priority 2: Use duration to calculate segment count.
         timescale := resolveTimescale(rep, as, template)
         segmentDuration := float64(*template.Duration) / float64(timescale)
         if segmentDuration > 0 {
            numSegments = int(math.Ceil(totalDuration / segmentDuration))
         }
      }
      if numSegments < 0 {
         numSegments = 0
      }
      for i := 0; i < numSegments; i++ {
         values := map[string]interface{}{
            "RepresentationID": rep.ID,
            "Number":           startNumber + i,
         }
         mediaPath := replacePlaceholders(template.Media, values)
         if resolvedURL, err := base.Parse(mediaPath); err == nil {
            allURLs = append(allURLs, resolvedURL.String())
         }
      }
   }
   return allURLs
}

func resolveNextBase(parentBase *url.URL, childBaseURL *BaseURL) *url.URL {
   if childBaseURL == nil || childBaseURL.Value == "" {
      return parentBase
   }
   resolvedURL, err := parentBase.Parse(childBaseURL.Value)
   if err != nil {
      log.Printf("Warning: Could not resolve BaseURL '%s'. Using parent base. Error: %v", childBaseURL.Value, err)
      return parentBase
   }
   return resolvedURL
}
