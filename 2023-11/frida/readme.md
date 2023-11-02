# Frida

- https://github.com/wkunzhi/FridaHookSysAPI
- https://httptoolkit.tech/blog/frida-certificate-pinning

install Frida:

~~~
pip install frida-tools
~~~

download and extra server:

https://github.com/frida/frida/releases

for example:

~~~
frida-server-16.0.0-android-x86.xz
~~~

then push:

~~~
adb root
adb push server /data/app/frida-server
adb shell chmod 755 /data/app/frida-server
adb shell /data/app/frida-server
~~~

then start Frida:

~~~
frida -U -l hello.js -f com.google.android.youtube
~~~

https://github.com/httptoolkit/frida-interception-and-unpinning/issues/51
