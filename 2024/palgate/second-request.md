# second request

even with current version, even without proxy, you get:

~~~
POST https://api1.pal-es.com/v1/bt/un/verify/start/+12345678901?countryCode=us&os=android HTTP/2.0
content-type: application/json; charset=UTF-8
user-agent: okhttp/4.9.3

{
  "v": 1,
  "s": "cbcaffb0-2d70-4dab-bc99-b475cb5028f0",
  "b": "F883B121D238F41D930F281C7A742A1ED75ACD96079AB58B3961CA17B86E993C87B6...",
  "type": "sms",
  "k": "EC7FF1016262879E2075220E428216F3"
}

HTTP/2.0 400 
date: Sun, 22 Sep 2024 14:59:35 GMT
content-type: application/json; charset=utf-8
content-length: 74
access-control-allow-origin: *
access-control-allow-methods: GET,PUT,POST,DELETE,OPTIONS
access-control-max-age: 600
strict-transport-security: max-age=15724800; includeSubDomains

{"status":"failed","msg":"Unable to register: Security check not passed!"}
~~~

lets download oldest working version:

~~~
play -i com.bluegate.app -c 342
~~~

extract:

~~~
jadx com.bluegate.app-342.apk
~~~

start here:

~~~
sources\com\bluegate\shared\BlueGateAPI.java
@o("un/verify/start/{id}")
e<OneTimeTokenResponse> sVerifyStart(@s("id") String str, @t("countryCode") String str2, @t("os") String str3, @a Map<String, Object> map);
~~~

then:

~~~
sources\com\bluegate\shared\ConnectionManager.java
public void sVerifyStart(String str, String str2, String str3, Map<String, Object> map, Response response) {
~~~

then:

~~~
sources\com\bluegate\app\fragments\VerifyPhoneFragment.java
395:ConnectionManager.getInstance().sVerifyStart(r3, VerifyPhoneFragment.this.mCountryCode, Constants.OS_ANDROID, hashMap, new AnonymousClass1());
1036:ConnectionManager.getInstance().sVerifyStart(r3, VerifyPhoneFragment.this.mCountryCode, Constants.OS_ANDROID, hashMap, new AnonymousClass1());
~~~

then:

~~~
hashMap.put("v", 1);
hashMap.put("type", r4);
hashMap.put("s", VerifyPhoneFragment.this.mSessionId);
hashMap.put("k", SharedUtils.intToHexString(t22));
if (FaceDetectNative.getInstance() != null && pk != null) {
   FaceDetectNative.getInstance().setPk(pk);
   FaceDetectNative.getInstance().setUser(r3);
   FaceDetectNative.getInstance().startBox();
   hashMap.put("b", FaceDetectNative.getInstance().getBox());
}
~~~

then:

~~~java
this.mSessionId = UUID.randomUUID().toString();
~~~

so `s` is good, what about `k`:

~~~java
int[] t22 = r2.getT2(hexStringToByteArray);
~~~

then:

~~~java
byte[] hexStringToByteArray = Utils.hexStringToByteArray(VerifyPhoneFragment.this.mT1_S);
~~~

then:

~~~java
VerifyPhoneFragment.this.mT1_S = oneTimeTokenResponse.getK();
~~~

then:

~~~
sources\com\bluegate\shared\FaceDetectNative.java
static {
   System.loadLibrary("native-lib");
}
public native int[] getT2(byte[] bArr);
~~~

which is here:

~~~
com.bluegate.app-config.x86-342.apk\lib\x86\libnative-lib.so
~~~

how do we call it?

https://mas.owasp.org/MASTG/techniques/android/MASTG-TECH-0018
