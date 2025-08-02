package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "os"
   "strconv"
   "strings"
)

const mpdBaseURL = "http://test.test/test.mpd"

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   Periods []Period `xml:"Period"`
}

type Period struct {
   BaseURL        string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
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
   EndNumber       int              `xml:"endNumber,attr"`
   Timescale       int              `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   Segments []Segment `xml:"S"`
}

type Segment struct {
   T int64 `xml:"t,attr"`
   D int64 `xml:"d,attr"`
   R int   `xml:"r,attr"`
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
}

func joinURL(base, rel string) string {
   if strings.HasPrefix(rel, "http://") || strings.HasPrefix(rel, "https://") {
      return rel
   }
   return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(rel, "/")
}

func resolveTemplate(base, template, repID string, number int, time int64) string {
   url := template
   url = strings.ReplaceAll(url, "$RepresentationID$", repID)
   url = strings.ReplaceAll(url, "$Number$", strconv.Itoa(number))
   url = strings.ReplaceAll(url, "$Time$", strconv.FormatInt(time, 10))
   return joinURL(base, url)
}

func generateTemplateURLs(base string, st *SegmentTemplate, repID string) []string {
   var urls []string

   if st.Initialization != "" {
      urls = append(urls, resolveTemplate(base, st.Initialization, repID, 0, 0))
   }

   if st.SegmentTimeline != nil {
      var currentTime int64
      for _, seg := range st.SegmentTimeline.Segments {
         repeat := seg.R
         if repeat < 0 {
            repeat = 0
         }
         t := seg.T
         if t != 0 {
            currentTime = t
         }
         for i := 0; i <= repeat; i++ {
            urls = append(urls, resolveTemplate(base, st.Media, repID, 0, currentTime))
            currentTime += seg.D
         }
      }
      return urls
   }

   start := 1
   if st.StartNumber > 0 {
      start = st.StartNumber
   }
   end := start + 10
   if st.EndNumber >= start {
      end = st.EndNumber + 1
   }
   for i := start; i < end; i++ {
      urls = append(urls, resolveTemplate(base, st.Media, repID, i, 0))
   }
   return urls
}

func generateSegmentListURLs(base string, sl *SegmentList) []string {
   var urls []string
   if sl.Initialization != nil && sl.Initialization.SourceURL != "" {
      urls = append(urls, joinURL(base, sl.Initialization.SourceURL))
   }
   for _, seg := range sl.SegmentURLs {
      urls = append(urls, joinURL(base, seg.Media))
   }
   return urls
}

func main() {
   if len(os.Args) != 2 {
      fmt.Println("Usage: go run main.go <mpd_file_path>")
      return
   }

   data, err := ioutil.ReadFile(os.Args[1])
   if err != nil {
      fmt.Println("Error reading MPD:", err)
      return
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Println("Error parsing MPD:", err)
      return
   }

   baseDir := mpdBaseURL[:strings.LastIndex(mpdBaseURL, "/")+1]
   result := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodBase := baseDir
      if period.BaseURL != "" {
         periodBase = joinURL(baseDir, period.BaseURL)
      }

      for _, aset := range period.AdaptationSets {
         for _, rep := range aset.Representations {
            repID := rep.ID

            // Representation base
            repBase := periodBase
            if rep.BaseURL != "" {
               repBase = joinURL(periodBase, rep.BaseURL)
            }

            // SegmentTemplate
            tmpl := rep.SegmentTemplate
            if tmpl == nil {
               tmpl = aset.SegmentTemplate
            }
            if tmpl != nil {
               result[repID] = generateTemplateURLs(repBase, tmpl, repID)
               continue
            }

            // SegmentList
            slist := rep.SegmentList
            if slist == nil {
               slist = aset.SegmentList
            }
            if slist != nil {
               result[repID] = generateSegmentListURLs(repBase, slist)
               continue
            }

            // Fallback: single BaseURL
            if rep.BaseURL != "" {
               result[repID] = []string{repBase}
            }
         }
      }
   }

   jsonOut, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Println("Error marshaling JSON:", err)
      return
   }
   fmt.Println(string(jsonOut))
}
