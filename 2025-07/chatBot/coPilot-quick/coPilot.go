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
    "strconv"
    "strings"
)

type SegmentMap map[string][]string

type MPD struct {
    XMLName xml.Name `xml:"MPD"`
    BaseURL string   `xml:"BaseURL"`
    Periods []Period `xml:"Period"`
}

type Period struct {
    XMLName  xml.Name     `xml:"Period"`
    Duration string       `xml:"duration,attr"`
    BaseURL  string       `xml:"BaseURL"`
    AdaptSet []Adaptation `xml:"AdaptationSet"`
}

type Adaptation struct {
    XMLName         xml.Name         `xml:"AdaptationSet"`
    BaseURL         string           `xml:"BaseURL"`
    SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
    Representations []Representation `xml:"Representation"`
}

type Representation struct {
    XMLName         xml.Name         `xml:"Representation"`
    ID              string           `xml:"id,attr"`
    BaseURL         string           `xml:"BaseURL"`
    SegmentList     *SegmentList     `xml:"SegmentList"`
    SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentList struct {
    SegmentURLs []struct {
        Media string `xml:"media,attr"`
    } `xml:"SegmentURL"`
}

type SegmentTemplate struct {
    Timescale     int    `xml:"timescale,attr"`
    Duration      int    `xml:"duration,attr"`
    StartNumber   int    `xml:"startNumber,attr"`
    EndNumber     int    `xml:"endNumber,attr"`
    Initialization string `xml:"initialization,attr"`
    Media         string  `xml:"media,attr"`
    Timeline      []S     `xml:"SegmentTimeline>S"`
}

type S struct {
    T int `xml:"t,attr"`
    D int `xml:"d,attr"`
    R int `xml:"r,attr"`
}

func parseDuration(dur string) float64 {
    dur = strings.TrimPrefix(dur, "PT")
    if strings.HasSuffix(dur, "S") {
        dur = strings.TrimSuffix(dur, "S")
        f, _ := strconv.ParseFloat(dur, 64)
        return f
    }
    return 0
}

func resolveURL(base, ref string) string {
    u, err := url.Parse(ref)
    if err != nil || u.IsAbs() {
        return ref
    }
    b, err := url.Parse(base)
    if err != nil {
        return ref
    }
    return b.ResolveReference(u).String()
}

func layeredResolveURL(relative, mpdBase, periodBase string) string {
    base := "http://test.test/test.mpd"
    if mpdBase != "" {
        base = resolveURL(base, mpdBase)
    }
    if periodBase != "" {
        base = resolveURL(base, periodBase)
    }
    return resolveURL(base, relative)
}

func main() {
    if len(os.Args) < 2 {
        log.Fatal("Usage: go run main.go <path-to-mpd>")
    }

    data, err := ioutil.ReadFile(os.Args[1])
    if err != nil {
        log.Fatalf("Failed to read MPD file: %v", err)
    }

    var mpd MPD
    if err := xml.Unmarshal(data, &mpd); err != nil {
        log.Fatalf("Failed to parse MPD XML: %v", err)
    }

    segmentMap := make(SegmentMap)

    for _, period := range mpd.Periods {
        for _, ad := range period.AdaptSet {
            for _, rep := range ad.Representations {
                id := rep.ID
                var urls []string

                // Pick SegmentTemplate if available
                template := rep.SegmentTemplate
                if template == nil {
                    template = ad.SegmentTemplate
                }

                // Layered base URL resolution
                rBase := rep.BaseURL
                aBase := ad.BaseURL
                pBase := period.BaseURL
                mBase := mpd.BaseURL

                base := layeredResolveURL("", mBase, pBase)
                if aBase != "" {
                    base = resolveURL(base, aBase)
                }
                if rBase != "" {
                    base = resolveURL(base, rBase)
                }

                if rep.SegmentList != nil {
                    for _, seg := range rep.SegmentList.SegmentURLs {
                        urls = append(urls, layeredResolveURL(seg.Media, mBase, pBase))
                    }
                } else if template != nil {
                    timescale := template.Timescale
                    if timescale == 0 {
                        timescale = 1
                    }
                    duration := template.Duration
                    startNum := template.StartNumber
                    if startNum == 0 {
                        startNum = 1
                    }
                    endNum := template.EndNumber
                    media := template.Media

                    if len(template.Timeline) > 0 {
                        num := startNum
                        timelineTime := 0
                        for _, s := range template.Timeline {
                            repeat := s.R
                            if repeat < 0 {
                                repeat = 0
                            }
                            if s.T != 0 {
                                timelineTime = s.T
                            }
                            for i := 0; i <= repeat; i++ {
                                url := strings.ReplaceAll(media, "$RepresentationID$", id)
                                url = strings.ReplaceAll(url, "$Number$", fmt.Sprintf("%d", num))
                                url = strings.ReplaceAll(url, "$Time$", fmt.Sprintf("%d", timelineTime))
                                urls = append(urls, resolveURL(base, url))
                                timelineTime += s.D
                                num++
                            }
                        }
                    } else if endNum > 0 {
                        for i := startNum; i <= endNum; i++ {
                            url := strings.ReplaceAll(media, "$RepresentationID$", id)
                            url = strings.ReplaceAll(url, "$Number$", fmt.Sprintf("%d", i))
                            urls = append(urls, resolveURL(base, url))
                        }
                    } else {
                        periodDur := parseDuration(period.Duration)
                        count := int(math.Ceil(periodDur * float64(timescale) / float64(duration)))
                        for i := 0; i < count; i++ {
                            url := strings.ReplaceAll(media, "$RepresentationID$", id)
                            url = strings.ReplaceAll(url, "$Number$", fmt.Sprintf("%d", startNum+i))
                            urls = append(urls, resolveURL(base, url))
                        }
                    }
                } else {
                    urls = append(urls, base)
                }

                segmentMap[id] = append(segmentMap[id], urls...)
            }
        }
    }

    output, err := json.MarshalIndent(segmentMap, "", "  ")
    if err != nil {
        log.Fatalf("Failed to marshal JSON: %v", err)
    }
    fmt.Println(string(output))
}
