package main

import (
   "encoding/json"
   "log"
   "math"
   "os/exec"
   "testing"
)

const initialization = 1

var tests = []struct {
   name           string
   representation []representationA
}{
   {
      name: "../testdata/canal.mpd",
      representation: []representationA{
         {
            content_type: type_video,
            id:           "video=3399914",
            length:       initialization + 1 + 1332 + 1,
            url:          prefix + "dash/appletvcz_A007300100102_2464C3BF9652075492E7CF48A400F243_HD-video=3399914-4798800.dash?serviceid=298f95e1bf91361258c44a2b1f4a2425",
         },
         {
            content_type: type_image,
            id:           "thumbnail",
            length:       80,
            url:          prefix + "dash/thumbnail/tile_80.jpeg?serviceid=298f95e1bf91361258c44a2b1f4a2425",
         },
      },
   },
   {
      name: "../testdata/criterion.mpd",
      representation: []representationA{
         {
            content_type: type_video,
            id:           "video-31e3417b-d163-45ba-9d7d-c356ef7aca57",
            length:       initialization + 1084,
            url:          prefix + "drm/cenc,derived,1039101898,70c108dd0a77b48f8738a37ed402d1e3/range/prot/aW5pdF9yYW5nZT0wLTgwNCZyYW5nZT0xODYzNTk5MTk1LTE4NjM4MTU0Mzk/avf/31e3417b-d163-45ba-9d7d-c356ef7aca57.mp4?init_range=0-804&pathsig=8c953e4f~H4likJuUcpljeDjz1r1ky7d3kH28faTdejTPfwh_5DY&r=dXMtZWFzdDE%3D&range=1863599195-1863815439",
         },
         {
            content_type: type_text,
            id:           "subs-202936084",
            length:       1,
            url:          prefix + "texttrack/sub/202936084.vtt?pathsig=8c953e4f~VmhEmE4Ge9n9S9R4gdPqJo02oUhoM8-o1o0iBVypx3k&r=dXMtY2VudHJhbDE%3D",
         },
      },
   },
   {
      name: "../testdata/max.mpd",
      representation: []representationA{
         {
            content_type: type_video,
            id:           "v31",
            length:       8, // allow duplicates
            url:          prefix + "v/1_584f52/v31.mp4",
         },
         {
            content_type: type_text,
            id:           "t2",
            length:       8,
            url:          prefix + "t/0_d2d294/t2/8.vtt",
         },
         {
            content_type: type_image,
            id:           "images_1",
            length: func() int { // allow duplicates
               media := math.Ceil(1100.34925 / 5)
               media += math.Ceil(1122.2044166666667 / 5)
               media += math.Ceil(1104.1864166666664 / 5)
               media += math.Ceil(1017.5582083333338 / 5)
               media += math.Ceil(1058.9328749999995 / 5)
               media += math.Ceil(1011.9692916666672 / 5)
               media += math.Ceil(994.4934999999996 / 5)
               media += math.Ceil(1914.2873749999999 / 5)
               return int(media)
            }(),
            url: prefix + "i/1_6c0f17/images_1_00001863.jpg",
         },
      },
   },
   {
      name: "../testdata/molotov.mpd",
      representation: []representationA{
         {
            content_type: type_video,
            id:           "video=4800000",
            length:       initialization + 3555,
            url:          prefix + "dash/32e3c47902de4911dca77b0ad73e9ac34965a1d8-video=4800000-3555.m4s",
         },
         {
            content_type: type_text,
            id:           "3=1000",
            length:       initialization + 3339,
            url:          prefix + "dash/32e3c47902de4911dca77b0ad73e9ac34965a1d8-3=1000-3339.m4s",
         },
      },
   },
   {
      name: "../testdata/paramount.mpd",
      representation: []representationA{
         {
            content_type: type_video,
            id:           "5",
            url:          prefix + "TPIR_0722_100824_2997DF_1920x1080_178_2CH_PRORESHQ_2CH_2939373_4500/seg_571.m4s",
            length:       9*initialization + 539 + 1 + 1 + 29 + 1,
         },
         {
            content_type: type_image,
            id:           "thumb_320x180",
            length:       11,
            url:          prefix + "thumb_320x180/tile_11.jpg",
         },
         {
            content_type: type_text,
            id:           "8",
            length:       9*initialization + 540 + 1 + 22,
            url:          prefix + "TPIR_0722_2997_2CH_DF_1728406422/seg_563.m4s",
         },
      },
   },
   {
      name: "../testdata/rakuten.mpd",
      representation: []representationA{
         {
            content_type: type_video,
            id:           "video-avc1-6",
            length:       1,
            url:          "https://prod-avod-pmd-cdn77.cdn.rakuten.tv/3/1/8/318f7ece69afcfe3e96de31be6b77272-mc-0-164-0-0_DS2BB/video-avc1-6.ismv?streaming_id=630ed6ed-1137-473c-8858-23ba59d12675&st_country=CZ,AT,DE,PL,SK&st_valid=1752245624&secure=PcDPRLtVJ_-tDsEPMD5Hzg==,1752267224",
         },
      },
   },
}

func Test(t *testing.T) {
   log.SetFlags(log.Ltime)
   for _, testVar := range tests {
      data, err := output("go", "run", ".", testVar.name)
      if err != nil {
         t.Fatal(string(data))
      }
      var representsB map[string][]string
      err = json.Unmarshal(data, &representsB)
      if err != nil {
         t.Fatal(err)
      }
      for _, representA := range testVar.representation {
         representB := representsB[representA.id]
         if len(representB) != representA.length {
            t.Fatal(
               representA.id,
               "pass", representA.length,
               "fail", len(representB),
            )
         }
         if representB[len(representB)-1] != representA.url {
            t.Fatal(
               "\npass", representA.url,
               "\nfail", representB[len(representB)-1],
            )
         }
      }
   }
}

func output(name string, arg ...string) ([]byte, error) {
   command := exec.Command(name, arg...)
   log.Print(command.Args)
   return command.Output()
}

type content_type int

const (
   type_image content_type = iota
   type_text
   type_video
)

type representationA struct {
   id           string
   length       int
   url          string
   content_type content_type
}

const prefix = "http://test.test/"
