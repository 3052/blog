package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "io/ioutil"
   "net/url"
   "os"
   "regexp"
   "strconv"
)

// Regex for placeholder substitution: $RepresentationID[%fmt]$, $Number[%fmt]$, $Time[%fmt]$
var placeholderRe = regexp.MustCompile(`\$(RepresentationID|Number|Time)(%[^$]+)?\$`)

func substitute(str, repID string, number, timestamp uint64) string {
   return placeholderRe.ReplaceAllStringFunc(str, func(m string) string {
      sub := placeholderRe.FindStringSubmatch(m)
      name := sub[1]
      fmtSpec := sub[2]
      if fmtSpec == "" {
         switch name {
         case "RepresentationID":
            return repID
         case "Number":
            return strconv.FormatUint(number, 10)
         case "Time":
            return strconv.FormatUint(timestamp, 10)
         }
      } else {
         // fmtSpec includes leading %
         switch name {
         case "RepresentationID":
            return fmt.Sprintf(fmtSpec, repID)
         case "Number":
            return fmt.Sprintf(fmtSpec, number)
         case "Time":
            return fmt.Sprintf(fmtSpec, timestamp)
         }
      }
      return m
   })
}

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Period  Period   `xml:"Period"`
}

type Period struct {
   BaseURL        string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
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

// Added EndNumber for numeric-only templates
type SegmentTemplate struct {
   Timescale       uint64          `xml:"timescale,attr"`
   Initialization  string          `xml:"initialization,attr"`
   Media           string          `xml:"media,attr"`
   StartNumber     uint64          `xml:"startNumber,attr"`
   EndNumber       uint64          `xml:"endNumber,attr"`
   SegmentTimeline SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
   S []S `xml:"S"`
}

type S struct {
   T uint64 `xml:"t,attr"`
   D uint64 `xml:"d,attr"`
   R uint64 `xml:"r,attr"` // repeat count
}

func resolveBase(parent *url.URL, refStr string) *url.URL {
   if refStr == "" {
      return parent
   }
   if ref, err := url.Parse(refStr); err == nil {
      return parent.ResolveReference(ref)
   }
   return parent
}

// applyTemplate builds URLs using SegmentTemplate; respects EndNumber if set.
func applyTemplate(tmpl *SegmentTemplate, base *url.URL, repID string) []string {
   var urls []string
   if tmpl == nil {
      return urls
   }
   // Initialization segment
   if tmpl.Initialization != "" {
      initStr := substitute(tmpl.Initialization, repID, 0, 0)
      u := resolveBase(base, initStr)
      urls = append(urls, u.String())
   }
   // If no timeline but numeric range
   if len(tmpl.SegmentTimeline.S) == 0 && tmpl.EndNumber > 0 {
      number := tmpl.StartNumber
      for number <= tmpl.EndNumber {
         mediaStr := substitute(tmpl.Media, repID, number, 0)
         u := resolveBase(base, mediaStr)
         urls = append(urls, u.String())
         number++
      }
      return urls
   }
   // Timeline-based
   number := tmpl.StartNumber
   var currentTime uint64
   first := true
   for _, seg := range tmpl.SegmentTimeline.S {
      if first {
         currentTime = seg.T
         first = false
      } else if seg.T != 0 {
         currentTime = seg.T
      }
      repeat := int(seg.R)
      for i := 0; i <= repeat; i++ {
         if tmpl.EndNumber > 0 && number > tmpl.EndNumber {
            return urls
         }
         mediaStr := substitute(tmpl.Media, repID, number, currentTime)
         u := resolveBase(base, mediaStr)
         urls = append(urls, u.String())
         number++
         currentTime += seg.D
      }
   }
   return urls
}

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", os.Args[0])
      os.Exit(1)
   }
   mpdPath := os.Args[1]

   data, err := ioutil.ReadFile(mpdPath)
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error reading MPD: %v\n", err)
      os.Exit(1)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing MPD: %v\n", err)
      os.Exit(1)
   }

   // Root base URL
   rootBase, err := url.Parse("http://test.test/test.mpd")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Invalid base URL: %v\n", err)
      os.Exit(1)
   }
   rootBase = resolveBase(rootBase, mpd.BaseURL)

   result := make(map[string][]string)
   periodBase := resolveBase(rootBase, mpd.Period.BaseURL)

   for _, aset := range mpd.Period.AdaptationSets {
      asetBase := resolveBase(periodBase, aset.BaseURL)
      for _, rep := range aset.Representations {
         repBase := resolveBase(asetBase, rep.BaseURL)
         var urls []string
         if rep.SegmentList != nil {
            if rep.SegmentList.Initialization != nil {
               u := resolveBase(repBase, rep.SegmentList.Initialization.SourceURL)
               urls = append(urls, u.String())
            }
            for _, seg := range rep.SegmentList.SegmentURLs {
               u := resolveBase(repBase, seg.Media)
               urls = append(urls, u.String())
            }
         } else {
            tmpl := rep.SegmentTemplate
            if tmpl == nil {
               tmpl = aset.SegmentTemplate
            }
            urls = append(urls, applyTemplate(tmpl, repBase, rep.ID)...)
         }
         result[rep.ID] = urls
      }
   }

   out, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
      os.Exit(1)
   }
   fmt.Println(string(out))
}
