# physical device

the current version works:

~~~
> play -i com.bluegate.app
details[8] = 0 USD
details[13][1][4] = 1.5.082
details[13][1][12] = http://www.pal-es.com
details[13][1][16] = Mar 21, 2024
details[13][1][17] = APK APK APK APK APK
details[13][1][82][1][1] = 6.0 and up
downloads = 1.67 million
name = PalGate
size = 38.51 megabyte
version code = 344
~~~

and previous version:

~~~
play -i com.bluegate.app -s -c 342
~~~

older versions fail:

~~~
play -i com.bluegate.app -s -c 338
~~~

> Unable to register: Security check not passed!
