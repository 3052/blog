package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ---------- MPD structs ----------
type MPD struct {
	XMLName  xml.Name  `xml:"MPD"`
	Type     string    `xml:"type,attr"`
	MediaPresentationDuration IsoDuration `xml:"mediaPresentationDuration,attr"`
	BaseURL  string    `xml:"BaseURL"`
	Periods  []Period  `xml:"Period"`
}

type Period struct {
	Duration IsoDuration `xml:"duration,attr"`
	BaseURL  string      `xml:"BaseURL"`
	AS       []AdaptationSet `xml:"AdaptationSet"`
}

type AdaptationSet struct {
	BaseURL         string `xml:"BaseURL"`
	Representations []Representation `xml:"Representation"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
	SegmentList     *SegmentList     `xml:"SegmentList"`
}

type Representation struct {
	ID      string `xml:"id,attr"`
	BaseURL string `xml:"BaseURL"`
	SegmentTemplate *SegmentTemplate `xml:"SegmentTemplate"`
	SegmentList     *SegmentList     `xml:"SegmentList"`
}

// SegmentTemplate / SegmentList pointers for optional attributes
type SegmentTemplate struct {
	Media       string           `xml:"media,attr"`
	Timescale   int              `xml:"timescale,attr"`
	Duration    int              `xml:"duration,attr"`
	StartNumber *int             `xml:"startNumber,attr"`
	Timeline    *SegmentTimeline `xml:"SegmentTimeline"`
}

type SegmentList struct {
	SegmentURLs []SegmentURL       `xml:"SegmentURL"`
	Timeline    *SegmentTimeline   `xml:"SegmentTimeline"`
	Duration    int                `xml:"duration,attr"`
	Timescale   int                `xml:"timescale,attr"`
	StartNumber *int               `xml:"startNumber,attr"`
}

type SegmentURL struct {
	Media string `xml:"media,attr"`
}

type SegmentTimeline struct {
	S []S `xml:"S"`
}

type S struct {
	T int `xml:"t,attr"`
	D int `xml:"d,attr"`
	R int `xml:"r,attr"`
}

// ---------- ISO8601 duration ----------
type IsoDuration time.Duration

func (d *IsoDuration) UnmarshalXMLAttr(attr xml.Attr) error {
	v, err := parseISO8601Duration(attr.Value)
	if err != nil {
		return err
	}
	*d = IsoDuration(v)
	return nil
}

func parseISO8601Duration(s string) (time.Duration, error) {
	if !strings.HasPrefix(s, "PT") {
		return 0, errors.New("invalid duration")
	}
	s = strings.TrimPrefix(s, "PT")

	var h, m, sec float64
	if idx := strings.IndexByte(s, 'H'); idx != -1 {
		h, _ = strconv.ParseFloat(s[:idx], 64)
		s = s[idx+1:]
	}
	if idx := strings.IndexByte(s, 'M'); idx != -1 {
		m, _ = strconv.ParseFloat(s[:idx], 64)
		s = s[idx+1:]
	}
	if idx := strings.IndexByte(s, 'S'); idx != -1 {
		sec, _ = strconv.ParseFloat(s[:idx], 64)
	}
	total := (h*3600 + m*60 + sec) * float64(time.Second)
	return time.Duration(total), nil
}

// ---------- main ----------
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <local.mpd>\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	mpdPath := os.Args[1]

	data, err := os.ReadFile(mpdPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var mpd MPD
	if err := xml.Unmarshal(data, &mpd); err != nil {
		fmt.Fprintf(os.Stderr, "xml parse error: %v\n", err)
		os.Exit(1)
	}

	originalMPDURLStr := "http://test.test/test.mpd"
	if v := os.Getenv("MPDEXPAND_ORIGINAL_URL"); v != "" {
		originalMPDURLStr = v
	}
	originalMPDURL, err := url.Parse(originalMPDURLStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid original MPD URL: %v\n", err)
		os.Exit(1)
	}

	out := make(map[string][]string)

	for _, p := range mpd.Periods {
		periodDur := time.Duration(p.Duration)
		if periodDur == 0 {
			periodDur = time.Duration(mpd.MediaPresentationDuration)
		}
		periodBase := resolveBase(originalMPDURL, p.BaseURL)

		for _, as := range p.AS {
			asBase := resolveBase(periodBase, as.BaseURL)

			for _, rep := range as.Representations {
				repBase := resolveBase(asBase, rep.BaseURL)
				repID := rep.ID

				var stmpl *SegmentTemplate
				var slist *SegmentList
				switch {
				case rep.SegmentTemplate != nil:
					stmpl = rep.SegmentTemplate
				case rep.SegmentList != nil:
					slist = rep.SegmentList
				case as.SegmentTemplate != nil:
					stmpl = as.SegmentTemplate
				case as.SegmentList != nil:
					slist = as.SegmentList
				}

				var urls []string
				switch {
				case stmpl != nil:
					urls, err = expandTemplate(stmpl, repID, periodDur, repBase)
				case slist != nil:
					urls, err = expandSegmentList(slist, repID, periodDur, repBase)
				default:
					urls = []string{repBase.String()}
				}
				if err != nil {
					fmt.Fprintf(os.Stderr, "error expanding rep %s: %v\n", repID, err)
					continue
				}
				out[repID] = unique(append(out[repID], urls...))
			}
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(os.Stderr, "json encode: %v\n", err)
		os.Exit(1)
	}
}

// ---------- helpers ----------
func resolveBase(parent *url.URL, base string) *url.URL {
	if base == "" {
		return parent
	}
	u, err := url.Parse(base)
	if err != nil {
		return parent
	}
	return parent.ResolveReference(u)
}

func effectiveStartNumber(p *int) int {
	if p == nil {
		return 1
	}
	return *p
}

func expandTemplate(st *SegmentTemplate, repID string, periodDur time.Duration, base *url.URL) ([]string, error) {
	media := st.Media
	if media == "" {
		return []string{base.String()}, nil
	}

	timescale := st.Timescale
	if timescale == 0 {
		timescale = 1
	}
	startNum := effectiveStartNumber(st.StartNumber)

	if st.Timeline != nil {
		return expandTimeline(st.Timeline, media, repID, startNum, base)
	}

	duration := st.Duration
	if duration <= 0 {
		return nil, errors.New("SegmentTemplate missing duration")
	}

	periodTicks := int64(periodDur.Seconds() * float64(timescale))
	lastNum := startNum + int((periodTicks+int64(duration)-1)/int64(duration)) - 1

	var urls []string
	for n := startNum; n <= lastNum; n++ {
		u := fillTemplate(media, repID, n, 0)
		abs := base.ResolveReference(mustParseURL(u))
		urls = append(urls, abs.String())
	}
	return urls, nil
}

func expandSegmentList(sl *SegmentList, repID string, periodDur time.Duration, base *url.URL) ([]string, error) {
	if sl.Timeline != nil {
		return expandTimeline(sl.Timeline, "", repID, effectiveStartNumber(sl.StartNumber), base)
	}
	var urls []string
	for _, su := range sl.SegmentURLs {
		if su.Media == "" {
			continue
		}
		abs := base.ResolveReference(mustParseURL(su.Media))
		urls = append(urls, abs.String())
	}
	return urls, nil
}

func expandTimeline(tl *SegmentTimeline, mediaTpl string, repID string, startNum int, base *url.URL) ([]string, error) {
	var urls []string
	num := startNum
	time := int64(0)

	for _, s := range tl.S {
		count := s.R + 1
		for i := 0; i < count; i++ {
			if i == 0 && s.T != 0 {
				time = int64(s.T)
			}
			if mediaTpl != "" {
				u := fillTemplate(mediaTpl, repID, num, time)
				abs := base.ResolveReference(mustParseURL(u))
				urls = append(urls, abs.String())
			}
			time += int64(s.D)
			num++
		}
	}
	return urls, nil
}

// ---------- template substitution ----------
var reToken = regexp.MustCompile(`\$([^$]+)\$`)

func fillTemplate(tpl string, repID string, number int, time int64) string {
	return reToken.ReplaceAllStringFunc(tpl, func(tok string) string {
		inner := tok[1 : len(tok)-1]
		field, widthStr, _ := strings.Cut(inner, "%")

		width := 0
		if widthStr != "" {
			if n, err := strconv.Atoi(strings.TrimSuffix(widthStr, "d")); err == nil {
				width = n
			}
		}

		switch strings.TrimSpace(field) {
		case "RepresentationID":
			return repID
		case "Number":
			return fmt.Sprintf("%0*d", width, number)
		case "Time":
			return fmt.Sprintf("%0*d", width, time)
		default:
			return tok
		}
	})
}

// ---------- utilities ----------
func unique(in []string) []string {
	seen := make(map[string]struct{})
	out := in[:0]
	for _, s := range in {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

func mustParseURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}
