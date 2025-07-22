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
    "path/filepath"
    "regexp"
    "strconv"
    "strings"
)

type MPD struct {
    XMLName xml.Name `xml:"MPD"`
    BaseURL string   `xml:"BaseURL"`
    Periods []Period `xml:"Period"`
}

type Period struct {
    Duration       string          `xml:"duration,attr"`
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
}

type SegmentTemplate struct {
    Timescale       int              `xml:"timescale,attr"`
    StartNumber     *int             `xml:"startNumber,attr"` // Spec #8: pointer to detect missing
    Duration        int              `xml:"duration,attr"`
    Media           string           `xml:"media,attr"`
    EndNumber       int              `xml:"endNumber,attr"`
    SegmentTimeline *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentTimeline struct {
    S []S `xml:"S"`
}

type S struct {
    D int `xml:"d,attr"`
    R int `xml:"r,attr"`
    T int `xml:"t,attr"`
}

func parseDuration(dur string) float64 {
    dur = strings.TrimPrefix(dur, "PT")
    dur = strings.TrimSuffix(dur, "S")
    if sec, err := strconv.ParseFloat(dur, 64); err == nil {
        return sec
    }
    return 0
}

func resolveURL(base string, layers ...string) string {
    u, _ := url.Parse(base)
    for _, l := range layers {
        if l != "" {
            parsed, _ := url.Parse(l)
            u = u.ResolveReference(parsed)
        }
    }
    return u.String()
}

func replaceNumberFormat(media string, number int) string {
    re := regexp.MustCompile(`\$Number%0(\d+)d\$`)
    if match := re.FindStringSubmatch(media); len(match) == 2 {
        width, _ := strconv.Atoi(match[1])
        formatted := fmt.Sprintf("%0*d", width, number)
        media = re.ReplaceAllString(media, formatted)
    }
    media = strings.ReplaceAll(media, "$Number$", fmt.Sprintf("%d", number))
    return media
}

func replaceVars(media string, id string, number int, time int) string {
    media = strings.ReplaceAll(media, "$RepresentationID$", id)
    media = replaceNumberFormat(media, number)
    media = strings.ReplaceAll(media, "$Time$", fmt.Sprintf("%d", time))
    return media
}

func main() {
    if len(os.Args) != 2 {
        log.Fatal("Usage: go run main.go <path-to-mpd>")
    }
    path := os.Args[1]

    data, err := ioutil.ReadFile(filepath.Clean(path))
    if err != nil {
        log.Fatal(err)
    }

    var mpd MPD
    if err := xml.Unmarshal(data, &mpd); err != nil {
        log.Fatal(err)
    }

    result := make(map[string][]string)
    seen := make(map[string]bool)

    for _, period := range mpd.Periods {
        periodDur := parseDuration(period.Duration)

        for _, set := range period.AdaptationSets {
            for _, rep := range set.Representations {
                id := rep.ID

                // Spec #6: Include Representation@BaseURL
                if rep.BaseURL != "" {
                    u := resolveURL("http://test.test/test.mpd", mpd.BaseURL, period.BaseURL, set.BaseURL, rep.BaseURL)
                    if !seen[u] {
                        result[id] = append(result[id], u)
                        seen[u] = true
                    }
                }

                var tmpl *SegmentTemplate
                if rep.SegmentTemplate != nil {
                    tmpl = rep.SegmentTemplate
                } else {
                    tmpl = set.SegmentTemplate
                }
                if tmpl == nil || tmpl.Media == "" {
                    continue
                }

                timescale := tmpl.Timescale
                if timescale == 0 {
                    timescale = 1
                }

                start := 1 // Spec #8: default to 1
                if tmpl.StartNumber != nil {
                    start = *tmpl.StartNumber
                }

                var urls []string
                if tmpl.SegmentTimeline != nil {
                    number := start
                    time := 0
                    for _, s := range tmpl.SegmentTimeline.S {
                        repeat := s.R
                        if repeat < 0 {
                            repeat = 0
                        }
                        for i := 0; i <= repeat; i++ {
                            media := replaceVars(tmpl.Media, id, number, time)
                            u := resolveURL("http://test.test/test.mpd", mpd.BaseURL, period.BaseURL, set.BaseURL, media)
                            if !seen[u] {
                                urls = append(urls, u)
                                seen[u] = true
                            }
                            number++
                            time += s.D
                        }
                    }
                } else {
                    segCount := tmpl.EndNumber
                    if segCount == 0 {
                        segCount = int(math.Ceil(periodDur * float64(timescale) / float64(tmpl.Duration)))
                    }
                    for i := 0; i < segCount; i++ {
                        number := start + i
                        media := replaceVars(tmpl.Media, id, number, 0)
                        u := resolveURL("http://test.test/test.mpd", mpd.BaseURL, period.BaseURL, set.BaseURL, media)
                        if !seen[u] {
                            urls = append(urls, u)
                            seen[u] = true
                        }
                    }
                }

                result[id] = append(result[id], urls...)
            }
        }
    }

    out, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(string(out))
}
