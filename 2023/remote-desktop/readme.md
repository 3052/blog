# Remote Desktop

OK I finally figured this out after playing with it at home. You have to set
your scaling client-side, then the remote session will match. those cant be
different it seems. then on top of that, if you are using Remote Desktop
Connection Manager, you HAVE to use FULL SCREEN with NO SCALING. any other
combination seems to ignore the scaling. this suck because then you likely have
the scroll bars because the remote session window will be too big. Alternately,
you can use Remote Desktop Connection instead in window mode, but then you lose
both the ability to auto scale to client area, and the ability to manage
multiple sessions in a single window.

- https://github.com/mRemoteNG/mRemoteNG/issues/1427
- https://techcommunity.microsoft.com/t5/security-compliance-and-identity/-/ba-p/248077
