package main

import (
   "encoding/json"
   "encoding/xml"
   "fmt"
   "log"
   "net/url"
   "os"
   "path/filepath"
   "regexp"
   "strconv"
   "strings"
)

// Initial absolute URL per specification
const startMPDURL = "http://test.test/test.mpd"

/* --------------------------------------------------------------------------
   MPEG‑DASH XML structures (single <BaseURL> per level)
   --------------------------------------------------------------------------*/

type SegmentURL struct {
   Media string `xml:"media,attr"`
}

type SegmentList struct {
   SegmentURLs []SegmentURL `xml:"SegmentURL"`
}

type S struct {
   T int `xml:"t,attr"`
   D int `xml:"d,attr"`
   R int `xml:"r,attr"`
}

type SegmentTimeline struct {
   Ss []S `xml:"S"`
}

type SegmentTemplate struct {
   Media           string           `xml:"media,attr"`
   Initialization  string           `xml:"initialization,attr"`
   StartNumberStr  string           `xml:"startNumber,attr"`
   EndNumberStr    string           `xml:"endNumber,attr"`
   DurationStr     string           `xml:"duration,attr"`
   TimescaleStr    string           `xml:"timescale,attr"`
   SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

// Helper getters
func (st *SegmentTemplate) startNumber() int {
   if st.StartNumberStr == "" {
      return 1
   }
   n, _ := strconv.Atoi(st.StartNumberStr)
   if n < 1 {
      n = 1
   }
   return n
}

func (st *SegmentTemplate) endNumber() (int, bool) {
   if st.EndNumberStr == "" {
      return 0, false
   }
   n, err := strconv.Atoi(st.EndNumberStr)
   if err != nil || n < 1 {
      return 0, false
   }
   return n, true
}

/* ---- hierarchy ---- */

type Representation struct {
   ID              string           `xml:"id,attr"`
   BaseURL         string           `xml:"BaseURL"`
   SegmentList     *SegmentList     `xml:"SegmentList"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type AdaptationSet struct {
   BaseURL         string           `xml:"BaseURL"`
   SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
   Representations []Representation `xml:"Representation"`
}

type Period struct {
   BaseURL        string          `xml:"BaseURL"`
   AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type MPD struct {
   XMLName xml.Name `xml:"MPD"`
   BaseURL string   `xml:"BaseURL"`
   Periods []Period `xml:"Period"`
}

/* --------------------------------------------------------------------------
   URL helpers
   --------------------------------------------------------------------------*/

func applyBaseURL(base *url.URL, rawBase string) *url.URL {
   rawBase = strings.TrimSpace(rawBase)
   if rawBase == "" {
      return base
   }
   ref, err := url.Parse(rawBase)
   if err != nil {
      log.Fatalf("invalid BaseURL %q: %v", rawBase, err)
   }
   return base.ResolveReference(ref)
}

/* --------------------------------------------------------------------------
   Token expansion ($RepresentationID$, $Number$, $Time$ with optional %0Nd)
   --------------------------------------------------------------------------*/

var tokenRE = regexp.MustCompile(`\$([A-Za-z]+)(%0(\d+)d)?\$`)

func zeroPad(val, width int) string { return fmt.Sprintf("%0*d", width, val) }

func expandTemplate(tpl, repID string, number, time int) string {
   return tokenRE.ReplaceAllStringFunc(tpl, func(tok string) string {
      m := tokenRE.FindStringSubmatch(tok)
      name := m[1]
      width := 0
      if m[3] != "" {
         width, _ = strconv.Atoi(m[3])
      }
      switch name {
      case "RepresentationID":
         return repID
      case "Number":
         if width > 0 {
            return zeroPad(number, width)
         }
         return strconv.Itoa(number)
      case "Time":
         if width > 0 {
            return zeroPad(time, width)
         }
         return strconv.Itoa(time)
      default:
         return tok
      }
   })
}

/* --------------------------------------------------------------------------
   Segment generators
   --------------------------------------------------------------------------*/

func generateFromSegmentList(base *url.URL, sl *SegmentList, repID string) []string {
   var out []string
   for idx, su := range sl.SegmentURLs {
      expanded := expandTemplate(strings.TrimSpace(su.Media), repID, idx+1, 0)
      ref, err := url.Parse(expanded)
      if err != nil {
         log.Fatalf("bad SegmentURL %q: %v", su.Media, err)
      }
      out = append(out, base.ResolveReference(ref).String())
   }
   return out
}

func generateFromSegmentTemplate(base *url.URL, tmpl *SegmentTemplate, repID string) ([]string, error) {
   if tmpl == nil {
      return nil, nil
   }

   var urls []string
   startNum := tmpl.startNumber()

   // ---------------- Initialization ----------------
   if tmpl.Initialization != "" {
      initStr := expandTemplate(tmpl.Initialization, repID, startNum, 0)
      ref, err := url.Parse(initStr)
      if err != nil {
         return nil, fmt.Errorf("SegmentTemplate initialization parse: %w", err)
      }
      urls = append(urls, base.ResolveReference(ref).String())
   }

   // If no media field we return just initialization
   if tmpl.Media == "" {
      return urls, nil
   }

   // -------- Option 1: SegmentTimeline --------
   if tl := tmpl.SegmentTimeline; tl != nil && len(tl.Ss) > 0 {
      num, curTime := startNum, 0
      for _, entry := range tl.Ss {
         repeat := entry.R
         if repeat < 0 {
            repeat = 0
         }
         if entry.T != 0 {
            curTime = entry.T
         }
         for i := 0; i <= repeat; i++ {
            mediaStr := expandTemplate(tmpl.Media, repID, num, curTime)
            ref, err := url.Parse(mediaStr)
            if err != nil {
               return nil, fmt.Errorf("SegmentTemplate media parse: %w", err)
            }
            urls = append(urls, base.ResolveReference(ref).String())
            num++
            curTime += entry.D
         }
      }
      return urls, nil
   }

   // -------- Option 2: startNumber / endNumber pair --------
   if endNum, ok := tmpl.endNumber(); ok {
      if endNum < startNum {
         return nil, fmt.Errorf("endNumber (%d) < startNumber (%d)", endNum, startNum)
      }
      for num := startNum; num <= endNum; num++ {
         mediaStr := expandTemplate(tmpl.Media, repID, num, 0)
         ref, err := url.Parse(mediaStr)
         if err != nil {
            return nil, fmt.Errorf("SegmentTemplate media parse: %w", err)
         }
         urls = append(urls, base.ResolveReference(ref).String())
      }
      return urls, nil
   }

   // Fallback – single segment using startNumber
   mediaStr := expandTemplate(tmpl.Media, repID, startNum, 0)
   ref, err := url.Parse(mediaStr)
   if err != nil {
      return nil, fmt.Errorf("SegmentTemplate media parse: %w", err)
   }
   urls = append(urls, base.ResolveReference(ref).String())
   return urls, nil
}

/* --------------------------------------------------------------------------
   Traversal – gather segments
   --------------------------------------------------------------------------*/

func collectSegments(mpd *MPD) (map[string][]string, error) {
   start, _ := url.Parse(startMPDURL)
   mpdBase := applyBaseURL(start, mpd.BaseURL)
   out := make(map[string][]string)

   for _, period := range mpd.Periods {
      periodBase := applyBaseURL(mpdBase, period.BaseURL)
      for _, aset := range period.AdaptationSets {
         asetBase := applyBaseURL(periodBase, aset.BaseURL)

         for _, rep := range aset.Representations {
            repBase := applyBaseURL(asetBase, rep.BaseURL)

            // Priority: SegmentList > Representation.Template > AdaptationSet.Template
            if rep.SegmentList != nil {
               out[rep.ID] = generateFromSegmentList(repBase, rep.SegmentList, rep.ID)
               continue
            }

            tmpl := rep.SegmentTemplate
            if tmpl == nil {
               tmpl = aset.SegmentTemplate
            }
            if tmpl != nil {
               segs, err := generateFromSegmentTemplate(repBase, tmpl, rep.ID)
               if err != nil {
                  return nil, fmt.Errorf("representation %s: %v", rep.ID, err)
               }
               out[rep.ID] = segs
            }
         }
      }
   }
   return out, nil
}

/* --------------------------------------------------------------------------
   main
   --------------------------------------------------------------------------*/

func main() {
   if len(os.Args) != 2 {
      fmt.Fprintf(os.Stderr, "Usage: %s <mpd_file_path>\n", filepath.Base(os.Args[0]))
      os.Exit(1)
   }

   data, err := os.ReadFile(os.Args[1])
   if err != nil {
      log.Fatalf("read MPD: %v", err)
   }

   var mpd MPD
   if err := xml.Unmarshal(data, &mpd); err != nil {
      log.Fatalf("parse MPD XML: %v", err)
   }

   result, err := collectSegments(&mpd)
   if err != nil {
      log.Fatalf("processing MPD: %v", err)
   }

   enc, err := json.MarshalIndent(result, "", "  ")
   if err != nil {
      log.Fatalf("marshal JSON: %v", err)
   }

   fmt.Println(string(enc))
}
