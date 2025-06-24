package crypto

var Modules = []struct {
   cmac    bool
   ecb     bool
   elGamal bool
   go_sum  int
   note    string
   pad     bool
   url     string
}{
   {
      go_sum: 8,
      url:    "github.com/tink-crypto/tink-go",
   },
   {
      go_sum: 10,
      url:    "github.com/jacobsa/crypto",
   },
   {
      note: "pad function",
      url:  "github.com/RyuaNerin/go-krypto/issues/6",
   },
   {
      go_sum: 7,
      note: "pubkey/elgamalecc: Encrypt should accept point input",
      url:  "github.com/deatil/go-cryptobin/issues/37",
   },
   {
      note: "ECB mode",
      url:  "github.com/enceve/crypto/issues/20",
   },
   {
      ecb:  true,
      pad:  true,
      cmac: false,
      note: "CMAC",
      url:  "github.com/Colduction/aes/issues/2",
   },
   {
      note: `module is stupid
      github.com/pedroalbanese/gogost/blob/master/cmd/cmac/main.go`,
      url: "github.com/pedroalbanese/gogost",
   },
   //////////////////////////////////////////////////////////////////////////////
   {
      ecb:     true,
      pad:     true,
      cmac:    true,
      elGamal: false,
      go_sum:  4,
      note:    "p256: Encrypt should accept point input",
      url:     "github.com/go-webdl/crypto/issues/3",
   },
   {
      ecb:     true,
      pad:     true,
      cmac:    true,
      elGamal: false,
      go_sum:  4,
      note:    "ElGamal ECC",
      url:     "github.com/emmansun/gmsm/issues/338",
   },
}
