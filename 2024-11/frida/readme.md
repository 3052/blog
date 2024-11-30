# Frida

- https://github.com/wkunzhi/FridaHookSysAPI
- https://httptoolkit.tech/blog/frida-certificate-pinning

install Frida:

~~~
pip install frida-tools
~~~

download and extract server:

https://github.com/frida/frida/releases

for example:

~~~
frida-server-16.0.0-android-x86.xz
~~~

then push:

~~~
adb root
adb push frida-server-16.2.2-android-x86 /data/app/frida-server
adb shell chmod +x /data/app/frida-server
adb shell /data/app/frida-server
~~~

then start Frida:

~~~
frida -U `
-l android/android-certificate-unpinning.js `
-l config.js `
-f com.audioteka
~~~

or:

~~~
frida -U `
-l config.js `
-l android/android-certificate-unpinning.js `
-l android/android-certificate-unpinning-fallback.js `
-f com.audioteka
~~~

https://github.com/httptoolkit/frida-interception-and-unpinning/issues/51
