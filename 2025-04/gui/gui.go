package gui

type test struct {
   api_url   string
   issue     string
   issue_url string
   pushed_at int
   size_exe  string
   size_go   int
   url       string
}

var Tests = []test{
   {
      pushed_at: 2021,
      url:       "github.com/tadvi/winc",
      size_exe:  "1.62 MB",
   },
   {
      pushed_at: 2023,
      size_exe:  "1.69 MB",
      url:       "github.com/rodrigocfd/windigo",
   },
   {
      api_url:   "api.github.com/repos/aarzilli/nucular",
      issue_url: "github.com/aarzilli/nucular/issues/89",
      size_exe:  "2.75 MB",
      size_go:   13_293,
   },
   {
      issue:     "button example",
      issue_url: "github.com/yottahmd/furex/issues/73",
      size_exe:  "8.44 MB",
   },
   {
      issue:     "example without ebiten",
      issue_url: "github.com/demouth/ebitenlg/issues/1",
      size_exe:  "8.93 MB",
   },
   {
      issue:     "executable size",
      issue_url: "github.com/hajimehoshi/guigui/issues/27",
      size_exe:  "10.4 MB",
   },
   {
      url:       "github.com/zeozeozeo/ebitengine-microui-go",
      issue_url: "github.com/zeozeozeo/microui-go/issues/4",
      size_exe:  "10.6 MB",
   },
   {
      issue_url: "github.com/zeozeozeo/microui-go/issues/4",
      size_exe:  "10.6 MB",
      url:       "github.com/zeozeozeo/microui-go",
   },
   {
      size_exe:  "15.7 MB",
      issue_url: "codeberg.org/tslocum/etk/issues/11",
   },
   {
      api_url:   "api.github.com/repos/cogentcore/core",
      size_go:   113_943,
      issue_url: "github.com/cogentcore/core/issues/1497",
      issue:      "unknown revision f26f1ae0a7c4",
   },
   {
      api_url:   "api.github.com/repos/richardwilkes/unison",
      issue:     "build without C",
      issue_url: "github.com/richardwilkes/unison/issues/64",
      size_go:   161_209,
   },
   {
      api_url:   "api.github.com/repos/fyne-io/fyne",
      issue:     "build without C",
      issue_url: "github.com/fyne-io/fyne/issues/5651",
      size_go:   113_691,
   },
   {
      issue:     "make it clear that this is not pure Go",
      issue_url: "github.com/AllenDang/giu/issues/965",
   },
   {
      issue:     "undefined: syscall.Mkfifo",
      issue_url: "github.com/codeation/impress/issues/3",
   },
   {
      issue:     "static library",
      issue_url: "github.com/twgh/xcgui/issues/41",
   },
   {
      issue:     "BindWidgets error GetDlgItem",
      issue_url: "github.com/whtiehack/wingui/issues/18",
   },
   {
      issue_url: "github.com/ying32/govcl/issues/226",
      issue:     "static Liblcl",
   },
   {
      issue_url: "github.com/AllenDang/gform/issues/18",
      issue:     "cannot use flag (type uint32)",
   },
   {
      issue_url: "github.com/torlangballe/zui/issues/4",
      issue:     "found packages zwindow (Window!js.go) and zui (Window_windows.go)",
   },
}
