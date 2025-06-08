package crypto

type module struct {
   go_sum int
   url    string
}

var modules = []module{
   {
      go_sum: 2,
      url:    "github.com/pedroalbanese/gogost",
   },
   {
      go_sum: 4,
      url:    "github.com/enceve/crypto",
   },
   {
      url:    "github.com/RyuaNerin/go-krypto",
      go_sum: 4,
   },
   {
      go_sum: 7,
      url:    "github.com/deatil/go-cryptobin",
   },
   {
      go_sum: 8,
      url:    "github.com/tink-crypto/tink-go",
   },
   {
      url:    "github.com/jacobsa/crypto",
      go_sum: 10,
   },
}
