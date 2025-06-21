package crypto

var Modules = []struct {
   ecb    bool
   go_sum int
   note   string
   pad    bool
   url    string
}{
   {
      ecb:    true,
      go_sum: 4,
      note:   "export CMAC",
      pad:    true,
      url:    "github.com/emmansun/gmsm/issues/332",
   },
   {
      ecb:    true,
      go_sum: 4,
      note:   "tag version",
      pad:    true,
      url:    "github.com/go-webdl/crypto/issues/2",
   },
   {
      go_sum: 8,
      url:    "github.com/tink-crypto/tink-go",
   },
   {
      go_sum: 10,
      url:    "github.com/jacobsa/crypto",
   },
   {
      note: "export types",
      url:  "github.com/Colduction/aes/issues/1",
   },
   {
      note: "pad function",
      url:  "github.com/RyuaNerin/go-krypto/issues/6",
   },
   {
      note: "pubkey/elgamalecc: Encrypt should accept point input",
      url:  "github.com/deatil/go-cryptobin/issues/37",
   },
   {
      note: "ECB mode",
      url:  "github.com/enceve/crypto/issues/20",
   },
   {
      note: `module is stupid
      github.com/pedroalbanese/gogost/blob/master/cmd/cmac/main.go`,
      url: "github.com/pedroalbanese/gogost",
   },
}
