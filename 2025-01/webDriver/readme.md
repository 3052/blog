# webDriver

https://github.com/mozilla/geckodriver

## new session

request:

~~~
curl -d '
{
   "capabilities": {
      "alwaysMatch": {
         "proxy": {
            "proxyType": "manual",
            "sslProxy": "res.proxy-seller.com:10000"
         }
      }
   }
}
' -H content-type:application/json 127.0.0.1:4444/session

curl -d '
{
   "capabilities": {
      "alwaysMatch": {
         "proxy": {"proxyType": "system"}
      }
   }
}
' -H content-type:application/json 127.0.0.1:4444/session

curl -d '
{
   "capabilities": {}
}
' -H content-type:application/json 127.0.0.1:4444/session
~~~

https://github.com/mozilla/geckodriver/issues/1872

~~~json
{
  "value": {
    "sessionId": "222423a0-4da3-413c-a3fe-be1725c4c142"
  }
}
~~~

- https://w3c.github.io/webdriver#dfn-new-sessions
- https://w3c.github.io/webdriver#proxy

## navigate to

~~~
curl -H content-type:application/json `
-d '{"url": "https://www.sky.ch"}' `
127.0.0.1:4444/session/222423a0-4da3-413c-a3fe-be1725c4c142/url
~~~

https://w3c.github.io/webdriver#navigate-to

## get all cookies

~~~
curl 127.0.0.1:4444/session/73a1041c-0bf6-4d40-a3f0-1784f8bde792/cookie
~~~

https://w3c.github.io/webdriver#cookies
