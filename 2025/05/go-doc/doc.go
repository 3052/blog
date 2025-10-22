package doc

type option struct {
   issue string
   url   string
}

var options = []option{
   // yes
   {
      issue: "opt-in vanity import tags",
      url:   "github.com/abhinav/doc2go/issues/74",
   },
   // maybe
   // no
   {
      issue: "do not document tests",
      url:   "github.com/goradd/moddoc/issues/2",
   },
   {
      issue: "field types should be clickable",
      url:   "github.com/Vanilla-OS/Pallas/issues/10",
   },
   {
      issue: "remove wget requirement",
      url:   "github.com/viamrobotics/govanity/issues/6",
   },
   {
      issue: "document single module",
      url:   "github.com/dsnet/godoc/issues/3",
   },
   {
      issue: "tool incorrectly strips module prefix",
      url:   "codeberg.org/pfad.fr/vanitydoc/issues/24",
   },
}
