package main

import (
    "encoding/json"
    "encoding/xml"
    "fmt"
    "log"
    "net/url"
    "os"
    "strconv"
    "strings"
)

type MPD struct {
    XMLName  xml.Name `xml:"MPD"`
    BaseURLs []BaseURL   `xml:"BaseURL"`
    Periods  []Period    `xml:"Period"`
}

type BaseURL struct {
    Text string `xml:",chardata"`
}

type Period struct {
    BaseURLs []BaseURL       `xml:"BaseURL"`
    Adapt    []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
    BaseURLs        []BaseURL        `xml:"BaseURL"`
    SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
    Repr            []Representation `xml:"Representation"`
}

type Representation struct {
    ID              string           `xml:"id,attr"`
    BaseURLs        []BaseURL        `xml:"BaseURL"`
    SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
}

type SegmentTemplate struct {
    Timescale     uint64           `xml:"timescale,attr"`
    StartNumber   uint64           `xml:"startNumber,attr"`
    EndNumber     uint64           `xml:"endNumber,attr"`
    Media         string           `xml:"media,attr"`
    Initialization string          `xml:"initialization,attr"`
    Timeline      *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
    S []Segment `xml:"S"`
}

type Segment struct {
    T *uint64 `xml:"t,attr"`
    D uint64  `xml:"d,attr"`
    R *int    `xml:"r,attr"`
}

func main() {
    if len(os.Args) < 2 {
        log.Fatalln("Usage: dashsegments <mpd-file>")
    }
    mpdFile := os.Args[1]

    // Base MPD URL for resolution
    mpdURL, err := url.Parse("http://test.test/test.mpd")
    if err != nil {
        log.Fatalln(err)
    }

    f, err := os.Open(mpdFile)
    if err != nil {
        log.Fatalln(err)
    }
    defer f.Close()

    var mpd MPD
    if err := xml.NewDecoder(f).Decode(&mpd); err != nil {
        log.Fatalln(err)
    }

    // Resolve MPD-level BaseURL
    globalBase := mpdURL
    if len(mpd.BaseURLs) > 0 {
        ref, err := url.Parse(strings.TrimSpace(mpd.BaseURLs[0].Text))
        if err == nil {
            globalBase = mpdURL.ResolveReference(ref)
        }
    }

    // Result map: RepresentationID -> []segment URLs
    result := make(map[string][]string)

    for _, period := range mpd.Periods {
        // Resolve Period BaseURL
        periodBase := globalBase
        if len(period.BaseURLs) > 0 {
            ref, err := url.Parse(strings.TrimSpace(period.BaseURLs[0].Text))
            if err == nil {
                periodBase = periodBase.ResolveReference(ref)
            }
        }

        for _, aset := range period.Adapt {
            // Resolve AdaptationSet BaseURL
            adBase := periodBase
            if len(aset.BaseURLs) > 0 {
                ref, err := url.Parse(strings.TrimSpace(aset.BaseURLs[0].Text))
                if err == nil {
                    adBase = adBase.ResolveReference(ref)
                }
            }

            for _, rep := range aset.Repr {
                repID := rep.ID
                repBase := adBase
                // Resolve Representation BaseURL
                if len(rep.BaseURLs) > 0 {
                    ref, err := url.Parse(strings.TrimSpace(rep.BaseURLs[0].Text))
                    if err == nil {
                        repBase = repBase.ResolveReference(ref)
                    }
                }

                segURLs := []string{}

                // If Representation@BaseURL exists, it's a segment URL
                if len(rep.BaseURLs) > 0 {
                    segURLs = append(segURLs, repBase.String())
                }

                // Choose SegmentTemplate: rep-level overrides set-level
                st := rep.SegmentTemplate
                if st == nil {
                    st = aset.SegmentTemplate
                }

                if st != nil {
                    timescale := st.Timescale
                    if timescale == 0 {
                        timescale = 1
                    }
                    startNum := st.StartNumber
                    if startNum == 0 {
                        startNum = 1
                    }
                    mediaTpl := st.Media

                    // Use SegmentTimeline if present
                    if st.Timeline != nil && len(st.Timeline.S) > 0 {
                        var t uint64
                        hasT := false
                        idx := startNum
                        for _, s := range st.Timeline.S {
                            cnt := 1
                            if s.R != nil {
                                cnt = *s.R + 1
                            }
                            if s.T != nil {
                                t = *s.T
                                hasT = true
                            } else if !hasT {
                                t = 0
                                hasT = true
                            }
                            for i := 0; i < cnt; i++ {
                                urlStr := mediaTpl
                                urlStr = strings.ReplaceAll(urlStr, "$RepresentationID$", repID)
                                urlStr = strings.ReplaceAll(urlStr, "$Number$", strconv.FormatUint(idx, 10))
                                urlStr = strings.ReplaceAll(urlStr, "$Time$", strconv.FormatUint(t, 10))
                                ref, err := url.Parse(urlStr)
                                if err == nil {
                                    segURLs = append(segURLs, repBase.ResolveReference(ref).String())
                                }
                                idx++
                                t += s.D
                            }
                        }
                    } else if st.EndNumber != 0 {
                        // Use EndNumber
                        for num := startNum; num <= st.EndNumber; num++ {
                            urlStr := mediaTpl
                            urlStr = strings.ReplaceAll(urlStr, "$RepresentationID$", repID)
                            urlStr = strings.ReplaceAll(urlStr, "$Number$", strconv.FormatUint(num, 10))
                            urlStr = strings.ReplaceAll(urlStr, "$Time$", "")
                            ref, err := url.Parse(urlStr)
                            if err == nil {
                                segURLs = append(segURLs, repBase.ResolveReference(ref).String())
                            }
                        }
                    }
                }

                result[repID] = segURLs
            }
        }
    }

    // Output JSON
    out, err := json.Marshal(result)
    if err != nil {
        log.Fatalln(err)
    }
    fmt.Println(string(out))
}
