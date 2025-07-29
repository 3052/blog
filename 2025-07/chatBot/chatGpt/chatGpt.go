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

const baseMPDURL = "http://test.test/test.mpd"

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL        string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
}

type SegmentTemplate struct {
   Initialization  string           `xml:"initialization,attr"`
   Media           string           `xml:"media,attr"`
   StartNumber     int              `xml:"startNumber,attr"`
   EndNumber       int              `xml:"endNumber,attr"` // custom extension
   Timescale       int              `xml:"timescale,attr"`
   Duration        int              `xml:"duration,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T int `xml:"t,attr"`
   D int `xml:"d,attr"`
   R int `xml:"r,attr"`
}

type SegmentList struct {
   Initialization *URL  `xml:"Initialization"`
   SegmentURLs    []URL `xml:"SegmentURL"`
}

type URL struct {
   SourceURL string `xml:"sourceURL,attr"`
}

func resolveURL(base, ref string) string {
   baseURL, _ := url.Parse(base)
   refURL, _ := url.Parse(ref)
   return baseURL.ResolveReference(refURL).String()
}

func extractSegments(rep Representation, parentBase string, inheritedTemplate *SegmentTemplate) []string {
   finalBase := parentBase
   if rep.BaseURL != "" {
      finalBase = resolveURL(parentBase, rep.BaseURL)
   }

   var segments []string

   tpl := rep.SegmentTemplate
   if tpl == nil {
      tpl = inheritedTemplate
   }

   if tpl != nil {
      if tpl.Initialization != "" {
         initURL := strings.ReplaceAll(tpl.Initialization, "$RepresentationID$", rep.ID)
         segments = append(segments, resolveURL(finalBase, initURL))
      }

      if tpl.SegmentTimeline != nil {
         currentTime := 0
         first := true
         for _, s := range tpl.SegmentTimeline.S {
            if first && s.T != 0 {
               currentTime = s.T
            }
            first = false

            count := 1
            if s.R > 0 {
               count += s.R
            }
            for i := 0; i < count; i++ {
               mediaURL := strings.ReplaceAll(tpl.Media, "$RepresentationID$", rep.ID)
               mediaURL = strings.ReplaceAll(mediaURL, "$Time$", strconv.Itoa(currentTime))
               segments = append(segments, resolveURL(finalBase, mediaURL))
               currentTime += s.D
            }
         }
      } else {
         start := tpl.StartNumber
         if start == 0 {
            start = 1
         }
         end := tpl.EndNumber
         if end < start {
            end = start + 4 // default: generate 5 segments
         }
         for num := start; num <= end; num++ {
            mediaURL := strings.ReplaceAll(tpl.Media, "$RepresentationID$", rep.ID)
            mediaURL = strings.ReplaceAll(mediaURL, "$Number$", strconv.Itoa(num))
            segments = append(segments, resolveURL(finalBase, mediaURL))
         }
      }
   } else if rep.SegmentList != nil {
      if rep.SegmentList.Initialization != nil {
         segments = append(segments, resolveURL(finalBase, rep.SegmentList.Initialization.SourceURL))
      }
      for _, seg := range rep.SegmentList.SegmentURLs {
         segments = append(segments, resolveURL(finalBase, seg.SourceURL))
      }
   } else if rep.BaseURL != "" {
      segments = append(segments, finalBase)
   }

   return segments
}

func main() {
   if len(os.Args) != 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      return
   }

   filePath := os.Args[1]
   data, err := ioutil.ReadFile(filePath)
   if err != nil {
      fmt.Printf("Failed to read MPD file: %v\n", err)
      return
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Printf("Failed to parse MPD XML: %v\n", err)
      return
   }

   base := baseMPDURL
   if mpd.BaseURL != "" {
      base = resolveURL(base, mpd.BaseURL)
   }

   output := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodBase := base
      if period.BaseURL != "" {
         periodBase = resolveURL(base, period.BaseURL)
      }
      for _, aset := range period.AdaptationSets {
         asetBase := periodBase
         if aset.BaseURL != "" {
            asetBase = resolveURL(periodBase, aset.BaseURL)
         }
         for _, rep := range aset.Representations {
            segments := extractSegments(rep, asetBase, aset.SegmentTemplate)
            output[rep.ID] = segments
         }
      }
   }

   jsonData, _ := json.MarshalIndent(output, "", "  ")
   fmt.Println(string(jsonData))
}
