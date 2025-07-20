package main

import (
    "encoding/json"
    "encoding/xml"
    "fmt"
    "io/ioutil"
    "net/url"
    "os"
    "strings"
)

type MPD struct {
    XMLName         xml.Name         `xml:"MPD"`
    Period          Period           `xml:"Period"`
    XMLBase         string           `xml:"BaseURL"`
}

type Period struct {
    AdaptationSets []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
    Representations []Representation `xml:"Representation"`
    SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
    BaseURL         string           `xml:"BaseURL"`
}

type Representation struct {
    ID              string           `xml:"id,attr"`
    BaseURL         string           `xml:"BaseURL"`
    SegmentList     *SegmentList     `xml:"SegmentList"`
    SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentTemplate struct {
    Media          string         `xml:"media,attr"`
    Initialization string         `xml:"initialization,attr"`
    Timescale      int            `xml:"timescale,attr"`
    SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
    StartNumber    int            `xml:"startNumber,attr"`
    Duration       int            `xml:"duration,attr"`
}

type SegmentTimeline struct {
    Segments []S `xml:"S"`
}

type S struct {
    T int `xml:"t,attr"`
    D int `xml:"d,attr"`
    R int `xml:"r,attr"`
}

type SegmentList struct {
    SegmentURLs []SegmentURL `xml:"SegmentURL"`
}

type SegmentURL struct {
    Media string `xml:"media,attr"`
}

func resolveURL(base string, ref string) string {
    baseURL, err := url.Parse(base)
    if err != nil {
        return ref
    }
    refURL, err := url.Parse(ref)
    if err != nil {
        return ref
    }
    return baseURL.ResolveReference(refURL).String()
}

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: go run main.go <mpd file path>")
        return
    }

    mpdPath := os.Args[1]
    mpdBytes, err := ioutil.ReadFile(mpdPath)
    if err != nil {
        panic(err)
    }

    var mpd MPD
    if err := xml.Unmarshal(mpdBytes, &mpd); err != nil {
        panic(err)
    }

    baseMPDURL := "http://test.test/test.mpd"
    output := make(map[string][]string)

    for _, as := range mpd.Period.AdaptationSets {
        for _, rep := range as.Representations {
            id := rep.ID
            var urls []string

            var template *SegmentTemplate
            if rep.SegmentTemplate != nil {
                template = rep.SegmentTemplate
            } else if as.SegmentTemplate != nil {
                template = as.SegmentTemplate
            }

            if template != nil && template.SegmentTimeline != nil {
                num := template.StartNumber
                if num == 0 {
                    num = 1
                }
                for _, s := range template.SegmentTimeline.Segments {
                    count := s.R + 1
                    for j := 0; j < count; j++ {
                        repl := strings.Replace(template.Media, "$Number$", fmt.Sprintf("%d", num), -1)
                        urls = append(urls, resolveURL(baseMPDURL, repl))
                        num++
                    }
                }
            } else if rep.SegmentList != nil {
                for _, seg := range rep.SegmentList.SegmentURLs {
                    urls = append(urls, resolveURL(baseMPDURL, seg.Media))
                }
            } else if rep.BaseURL != "" {
                urls = append(urls, resolveURL(baseMPDURL, rep.BaseURL))
            }

            if id != "" && len(urls) > 0 {
                output[id] = urls
            }
        }
    }

    outJSON, _ := json.MarshalIndent(output, "", "  ")
    fmt.Println(string(outJSON))
}

