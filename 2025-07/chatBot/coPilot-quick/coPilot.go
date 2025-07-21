package main

import (
    "encoding/xml"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "math"
    "net/url"
    "os"
    "strings"
   "strconv"
)

// Structs for MPD XML

type MPD struct {
    XMLName xml.Name `xml:"MPD"`
    BaseURL string   `xml:"BaseURL"`
    Periods []Period `xml:"Period"`
}

type Period struct {
    BaseURL        string           `xml:"BaseURL"`
    Duration       string           `xml:"duration,attr"`
    AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
    SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
    Representations []Representation `xml:"Representation"`
}

type Representation struct {
    ID              string            `xml:"id,attr"`
    BaseURL         string            `xml:"BaseURL"`
    SegmentTemplate *SegmentTemplate  `xml:"SegmentTemplate"`
    SegmentList     *SegmentList      `xml:"SegmentList"`
}

type SegmentList struct {
    // Placeholder if needed later
}

type SegmentTemplate struct {
    Media          string           `xml:"media,attr"`
    Timescale      int              `xml:"timescale,attr"`
    Duration       int              `xml:"duration,attr"`
    StartNumber    int              `xml:"startNumber,attr"`
    EndNumber      int              `xml:"endNumber,attr"`
    SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
    Segments []S `xml:"S"`
}

type S struct {
    T int `xml:"t,attr"`
    D int `xml:"d,attr"`
    R int `xml:"r,attr"`
}

func main() {
    if len(os.Args) < 2 {
        log.Fatalf("Usage: %s path/to/mpdfile.mpd\n", os.Args[0])
    }
    mpdPath := os.Args[1]
    data, err := ioutil.ReadFile(mpdPath)
    if err != nil {
        log.Fatal(err)
    }

    var mpd MPD
    if err := xml.Unmarshal(data, &mpd); err != nil {
        log.Fatal(err)
    }

    base, _ := url.Parse("http://test.test/test.mpd")
    output := make(map[string][]string)

    for _, period := range mpd.Periods {
        for _, as := range period.AdaptationSets {
            for _, rep := range as.Representations {
                template := rep.SegmentTemplate
                if template == nil && as.SegmentTemplate != nil {
                    template = as.SegmentTemplate
                }

                var segments []string

                if template != nil {
                    timescale := template.Timescale
                    if timescale == 0 {
                        timescale = 1
                    }
                    start := template.StartNumber
                    if start == 0 {
                        start = 1
                    }
                    var count int
                    if template.EndNumber > 0 {
                        count = template.EndNumber - start + 1
                    } else if template.SegmentTimeline != nil {
                        for _, s := range template.SegmentTimeline.Segments {
                            r := s.R
                            if r < 0 {
                                r = 0
                            }
                            for i := 0; i <= r; i++ {
                                num := start + len(segments)
                                urlStr := strings.ReplaceAll(template.Media, "$RepresentationID$", rep.ID)
                                urlStr = strings.ReplaceAll(urlStr, "$Number$", fmt.Sprintf("%d", num))
                                segments = append(segments, resolveURL(base, mpd.BaseURL, period.BaseURL, rep.BaseURL, urlStr))
                            }
                        }
                    } else if template.Duration > 0 && period.Duration != "" {
                        // rough estimation, assuming duration is in PT[n]S
                        dur := parseDuration(period.Duration)
                        count = int(math.Ceil(dur * float64(timescale) / float64(template.Duration)))
                    }

                    for i := 0; i < count; i++ {
                        num := start + i
                        urlStr := strings.ReplaceAll(template.Media, "$RepresentationID$", rep.ID)
                        urlStr = strings.ReplaceAll(urlStr, "$Number$", fmt.Sprintf("%d", num))
                        segments = append(segments, resolveURL(base, mpd.BaseURL, period.BaseURL, rep.BaseURL, urlStr))
                    }
                } else if rep.SegmentList == nil {
                    segments = append(segments, resolveURL(base, mpd.BaseURL, period.BaseURL, rep.BaseURL, ""))
                }

                output[rep.ID] = append(output[rep.ID], segments...)
            }
        }
    }

    b, err := json.MarshalIndent(output, "", "  ")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(b))
}


func resolveURL(base *url.URL, mpdBase, periodBase, repBase, seg string) string {
    u := base
    if mpdBase != "" {
        ref, _ := url.Parse(mpdBase)
        u = u.ResolveReference(ref)
    }
    if periodBase != "" {
        ref, _ := url.Parse(periodBase)
        u = u.ResolveReference(ref)
    }
    if repBase != "" {
        ref, _ := url.Parse(repBase)
        u = u.ResolveReference(ref)
    }
    if seg != "" {
        ref, _ := url.Parse(seg)
        u = u.ResolveReference(ref)
    }
    return u.String()
}


func parseDuration(d string) float64 {
    // very basic PT[n]S parser
    if strings.HasPrefix(d, "PT") && strings.HasSuffix(d, "S") {
        num := strings.TrimSuffix(strings.TrimPrefix(d, "PT"), "S")
        val, _ := strconv.ParseFloat(num, 64)
        return val
    }
    return 0.0
}
