package crypto

type module struct {
   go_sum int
   note   string
   url    []string
}

var modules = []module{
   {
      go_sum: 2,
      note: `module is stupid
      github.com/pedroalbanese/gogost/blob/master/cmd/cmac/main.go`,
      url: []string{"github.com/pedroalbanese/gogost"},
   },
   {
      go_sum: 4,
      url: []string{
         "api.github.com/repos/RyuaNerin/go-krypto",
         "github.com/RyuaNerin/go-krypto/issues/6",
      },
   },
   {
      go_sum: 7,
      url:    []string{"github.com/deatil/go-cryptobin"},
   },
   {
      go_sum: 8,
      url:    []string{"github.com/tink-crypto/tink-go"},
   },
   {
      go_sum: 10,
      url:    []string{"github.com/jacobsa/crypto"},
   },
   {
      go_sum: 4,
      url: []string{
         "api.github.com/repos/enceve/crypto",
         "github.com/enceve/crypto/issues/20",
      },
   },
}
