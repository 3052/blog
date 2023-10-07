package main

import (
   "154.pages.dev/api/musicbrainz"
   "154.pages.dev/media/youtube"
   "flag"
   "os"
   "strconv"
   "strings"
   "time"
)

func main() {
   var f struct {
      group string
      r youtube.Request
      release string
   }
   flag.StringVar(&f.group, "g", "", "MusicBrainz release group ID")
   flag.StringVar(&f.release, "r", "", "MusicBrainz release ID")
   flag.StringVar(&f.r.Video_ID, "y", "", "YouTube video ID")
   flag.Parse()
   switch {
   case f.group != "":
      group, err := musicbrainz.New_Release_Group(f.group)
      if err != nil {
         panic(err)
      }
      group.Sort()
      if err := view_musicbrainz(group.Releases[0]); err != nil {
         panic(err)
      }
   case f.release != "":
      rel, err := musicbrainz.New_Release(f.release)
      if err != nil {
         panic(err)
      }
      if err := view_musicbrainz(rel); err != nil {
         panic(err)
      }
   case f.r.Video_ID != "":
      f.r.Mobile_Web()
      play, err := f.r.Player(nil)
      if err != nil {
         panic(err)
      }
      if _, err := since_hours(play); err != nil {
         panic(err)
      }
   default:
      flag.Usage()
   }
}

func since_hours(play *youtube.Player) (int, error) {
   date, err := play.Time()
   if err != nil {
      return 0, err
   }
   years := time.Since(date).Hours() / 24 / 365
   per_year := float64(play.Video_Details.View_Count) / years
   var b []byte
   if per_year >= 10_000_000 {
      b = append(b, red...)
      b = append(b, " FAIL "...)
   } else {
      b = append(b, green...)
      b = append(b, " PASS "...)
   }
   b = append(b, reset...)
   b = append(b, "   "...)
   b = strconv.AppendFloat(b, per_year, 'f', 0, 64)
   b = append(b, "   "...)
   b = append(b, play.Video_Details.Video_ID...)
   b = append(b, '\n')
   return os.Stdout.Write(b)
}

func view_musicbrainz(rel *musicbrainz.Release) error {
   var artists strings.Builder
   for _, artist := range rel.Artist_Credit {
      artists.WriteString(artist.Name)
      artists.WriteByte(' ')
   }
   var web youtube.Request
   web.Mobile_Web()
   for _, media := range rel.Media {
      for _, track := range media.Tracks {
         search, err := web.Search(artists.String() + track.Title)
         if err != nil {
            return err
         }
         for _, item := range search.Items() {
            video := item.Video_With_Context_Renderer
            if video != nil {
               web.Video_ID = video.Video_ID
               play, err := web.Player(nil)
               if err != nil {
                  return err
               }
               if _, err := since_hours(play); err != nil {
                  return err
               }
            }
            break
         }
         time.Sleep(99 * time.Millisecond)
      }
   }
   return nil
}

const (
   reset = "\x1b[m"
   green = "\x1b[30;102m"
   red = "\x1b[30;101m"
)

