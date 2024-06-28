# play ready

- <https://security-explorations.com/samples/mspr_leak_screenshot3.png>
- <https://sendvsfeedback2-download.azurewebsites.net/api/fileBlob/file?name=B0cde770200a945109437927ba3fe4d67638537352993712632_ICE_REPRO.zip&tid=0cde770200a945109437927ba3fe4d67638537352993712632>
- https://files.catbox.moe/8iz2qk.pdb
- https://gofile.io/d/DwbPIU 
- https://reddit.com/r/ReverseEngineering/comments/1dnicyh
- https://seclists.org/fulldisclosure/2024/Jun/7
- https://security-explorations.com/microsoft-warbird-pmp.html

## how

1. visualstudio.microsoft.com/downloads
2. tools for visual studio
3. build tools for visual studio
4. continue
5. workloads
6. desktop development with C++
7. individual components
8. compilers, build tools, and runtimes
9. MSVC VS C++ ARM64/ARM64EC build tools
10. install
11. continue
12. close
13. start
14. visual studio
15. developer powershell for VS

~~~
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned
Launch-VsDevShell.ps1 -Arch arm64 -HostArch amd64
Get-Command link | Format-List
cd ice_repro\linker\linkrepro
$env:OBJECT_ROOT = '.'
link '@link.rsp'
~~~

result:

~~~
Generating code
Finished generating code

LINK : fatal error LNK1000: Internal error during IMAGE::EmitRelocations
~~~

this does not fix it:

https://support.microsoft.com/topic/19d26c90-5aeb-de46-ae0b-d864a94bb321

~~~
Date: Wed, 26 Jun 2024 16:41:27 -0500
Subject: how to build playReady DLL
To: "contact@security-explorations.com" <contact@security-explorations.com>

I tried this:

Set-ExecutionPolicy -ExecutionPolicy RemoteSigned
Launch-VsDevShell.ps1 -Arch arm64 -HostArch amd64
cd ice_repro\linker\linkrepro
$env:OBJECT_ROOT = '.'
link '@link.rsp'

but I get this result:

Generating code
Finished generating code
LINK : fatal error LNK1000: Internal error during IMAGE::EmitRelocations
~~~

then:

~~~
dumpbin /disasm windows.media.protection.playready.dll
~~~

## ghidra

1. file
2. new project
3. next
4. project name
5. finish
6. codeBrowser
7. file
8. import file
9. OK
10. analyze, yes
11. analyze

https://github.com/NationalSecurityAgency/ghidra

## search

- https://activation.playready.microsoft.com/PlayReady/ACT/Activation.asmx?WSDL
- https://prp.spserv.microsoft.com/PRProvisioning/AcquirePRCert/v1
- https://prp.spserv.microsoft.com/PRProvisioning/Bootstrap/v1

same:

- https://go.microsoft.com/fwlink/?LinkID=507346
- https://prod.provgateway.playready.microsoft.com/PRProvisioning/Bootstrap/v1

same:

- https://go.microsoft.com/fwlink/?LinkID=507347
- https://prod.provgateway.playready.microsoft.com/PRProvisioning/AcquirePRCert/v1

same:

- https://activation2.playready.microsoft.com/PlayReady/ACT/Activation.asmx?WSDL&Client=Win10&LinkId=613387
- https://go.microsoft.com/fwlink/?LinkID=613387
