package musicbrainz

import (
   "encoding/json"
   "net/http"
   "net/url"
   "sort"
   "strconv"
   "strings"
   "time"
)

const root = "http://musicbrainz.org/ws/2"

var status = map[string]int{"Official": 0, "Bootleg": 1}

type Release_Group struct {
   Release_Count int `json:"release-count"`
   Releases []*Release
}

func From_Artist(artist_ID string, offset int) (*Release_Group, error) {
   req, err := http.NewRequest("GET", root + "/release", nil)
   if err != nil {
      return nil, err
   }
   val := url.Values{
      "artist": {artist_ID},
      "fmt": {"json"},
      "inc": {"release-groups"},
      "limit": {"100"},
      "status": {"official"},
      "type": {"album"},
   }
   if offset >= 1 {
      val.Set("offset", strconv.Itoa(offset))
   }
   req.URL.RawQuery = val.Encode()
   res, err := new(http.Transport).RoundTrip(req)
   if err != nil {
      return nil, err
   }
   defer res.Body.Close()
   group := new(Release_Group)
   if err := json.NewDecoder(res.Body).Decode(group); err != nil {
      return nil, err
   }
   return group, nil
}

func New_Release_Group(group_ID string) (*Release_Group, error) {
   req, err := http.NewRequest("GET", root + "/release", nil)
   if err != nil {
      return nil, err
   }
   req.URL.RawQuery = url.Values{
      "fmt": {"json"},
      "inc": {"artist-credits recordings"},
      "release-group": {group_ID},
   }.Encode()
   res, err := new(http.Transport).RoundTrip(req)
   if err != nil {
      return nil, err
   }
   defer res.Body.Close()
   group := new(Release_Group)
   if err := json.NewDecoder(res.Body).Decode(group); err != nil {
      return nil, err
   }
   return group, nil
}

func (r Release_Group) Sort() {
   sort.Slice(r.Releases, func(a, b int) bool {
      ra, rb := r.Releases[a], r.Releases[b]
      com := status[ra.Status] - status[rb.Status]
      if com != 0 {
         return com <= -1
      }
      com = strings.Compare(ra.date(4), rb.date(4))
      if com != 0 {
         return com == -1
      }
      com = ra.track_len() - rb.track_len()
      if com != 0 {
         return com <= -1
      }
      return ra.date(10) < rb.date(10)
   })
}

type Release struct {
   Artist_Credit []struct {
      Name string
      Artist struct {
         ID string
      }
   } `json:"artist-credit"`
   Date string
   Media []struct {
      Track_Count int `json:"track-count"`
      Tracks []Track
   }
   Release_Group struct {
      First_Release_Date string `json:"first-release-date"`
      ID string
      Secondary_Types []string `json:"secondary-types"`
      Title string
   } `json:"release-group"`
   Status string
   Title string
}

func New_Release(release_ID string) (*Release, error) {
   req, err := http.NewRequest("GET", root + "/release/" + release_ID, nil)
   if err != nil {
      return nil, err
   }
   req.URL.RawQuery = url.Values{
      "fmt": {"json"},
      "inc": {"artist-credits recordings"},
   }.Encode()
   res, err := new(http.Transport).RoundTrip(req)
   if err != nil {
      return nil, err
   }
   defer res.Body.Close()
   rel := new(Release)
   if err := json.NewDecoder(res.Body).Decode(rel); err != nil {
      return nil, err
   }
   return rel, nil
}

func (r Release) date(width int) string {
   start := len(r.Date)
   right := "9999-12-31"[start:]
   return (r.Date + right)[:width]
}

func (r Release) track_len() int {
   var count int
   for _, media := range r.Media {
      count += media.Track_Count
   }
   return count
}

type Track struct {
   Length float64
   Title string
}

func (r Track) Duration() time.Duration {
   return time.Duration(r.Length) * time.Millisecond
}
