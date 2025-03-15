package main

import (
   "41.neocities.org/google/play"
   "41.neocities.org/x/stringer"
   "fmt"
   "os"
   "slices"
   "time"
)

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
   for _, app := range apps {
      detail, err := auth.Details(checkin, app.id, false)
      if err != nil {
         panic(err)
      }
      app.installs = detail.Downloads()
      app.name = detail.Name()
      fmt.Println(app.id, stringer.Cardinal(app.installs))
      time.Sleep(99 * time.Millisecond)
   }
   slices.SortFunc(apps, func(a, b *application) int {
      return int(b.installs - a.installs)
   })
   for i, app := range apps {
      fmt.Printf("%v. %v\n", i+1, app.name)
   }
}

type application struct {
   id string
   name string
   installs uint64
}

var apps = []*application{
   {id: "air.ITVMobilePlayer"},
   {id: "au.com.streamotion.ares"},
   {id: "be.rtbf.auvio"},
   {id: "ca.ctv.ctvgo"},
   {id: "com.amcplus.amcfullepisodes"},
   {id: "com.cbs.app"},
   {id: "com.criterionchannel"},
   {id: "com.draken.android"},
   {id: "com.hulu.plus"},
   {id: "com.kanopy"},
   {id: "com.mubi"},
   {id: "com.nbcuni.nbc"},
   {id: "com.plexapp.android"},
   {id: "com.roku.web.trc"},
   {id: "com.tubitv"},
   {id: "com.wbd.stream"},
   {id: "nl.peoplesplayground.audienceplayer.cinemember"},
   {id: "tv.pluto.android"},
   {id: "tv.wuaki"},
}

