package tuta

type test struct {
   connected bool
   country   string
   response  string
   service   string
}

const sorry = `Sorry, you are currently not allowed to send or receive emails
because your account was marked for approval. This process is necessary to
offer a privacy-friendly registration and prevent mass registrations at the
same time. Your account will normally be automatically approved after 48 hours.`

var Tests = []test{
   {
      connected: false,
      country:   "United States",
      response:  sorry,
   },
   {
      connected: true,
      country:   "Germany",
      response:  sorry,
      service:   "mullvad.net",
   },
   {
      connected: true,
      country:   "Germany",
      response:  sorry,
      service:   "proxy-seller.com",
   },
}
