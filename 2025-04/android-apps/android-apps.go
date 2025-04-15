package main

import (
   "41.neocities.org/google/play"
   "41.neocities.org/x/stringer"
   "fmt"
   "os"
   "slices"
   "time"
)

var apps = []application{
   {id: "air.ITVMobilePlayer"}, // ITVX
   {id: "be.rtbf.auvio"}, // RTBF Auvio : direct et replay
   {id: "ca.ctv.ctvgo"}, // CTV
   {id: "com.amcplus.amcfullepisodes"}, // AMC+
   {id: "com.canal.android.canal"}, // CANAL+, Live and catch-up TV
   {id: "com.cbs.app"}, // Paramount+
   {id: "com.criterionchannel"}, // The Criterion Channel
   {id: "com.draken.android"}, // Draken Film
   {id: "com.hulu.plus"}, // Hulu: Stream TV shows & movies
   {id: "com.kanopy"}, // Kanopy
   {id: "com.mubi"}, // MUBI: Curated Cinema
   {id: "com.nbcuni.nbc"}, // The NBC App - Stream TV Shows
   {id: "com.plexapp.android"}, // Plex: Stream Movies & TV
   {id: "com.roku.web.trc"}, // The Roku Channel
   {id: "com.tubitv"}, // Tubi: Free Movies & Live TV
   {id: "com.wbd.stream"}, // Max: Stream HBO, TV, & Movies
   {id: "nl.peoplesplayground.audienceplayer.cinemember"}, // CineMember
   {id: "tv.molotov.app"}, // Molotov - TV en direct, replay
   {id: "tv.pluto.android"}, // PlutoTV: Live TV & Free Movies
   {id: "tv.wuaki"}, // Rakuten TV -Movies & TV Series
}

type application struct {
   id       string
   name     string
   installs uint64
}

func main() {
   home, err := os.UserHomeDir()
   if err != nil {
      panic(err)
   }
   data, err := os.ReadFile(home + "/google/play/Token")
   if err != nil {
      panic(err)
   }
   var token play.Token
   err = token.Unmarshal(data)
   if err != nil {
      panic(err)
   }
   auth, err := token.Auth()
   if err != nil {
      panic(err)
   }
   data, err = os.ReadFile(home + "/google/play/x86")
   if err != nil {
      panic(err)
   }
   var checkin play.Checkin
   err = checkin.Unmarshal(data)
   if err != nil {
      panic(err)
   }
   for i, app := range apps {
      fmt.Println(app.id)
      detail, err := auth.Details(checkin, app.id, false)
      if err != nil {
         panic(err)
      }
      apps[i].installs = detail.Downloads()
      apps[i].name = detail.Name()
      time.Sleep(99 * time.Millisecond)
   }
   slices.SortFunc(apps, func(a, b application) int {
      return int(b.installs - a.installs)
   })
   for i, app := range apps {
      fmt.Printf("%v. %v\n", i+1, app.name)
      fmt.Printf("\t- %v\n", stringer.Cardinal(app.installs))
   }
}
