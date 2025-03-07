# 2025-03-07
# go.dev/doc/go1.13
$env:GOPROXY = 'direct'
$env:GOSUMDB = 'off'

# 2025-03-06
$env:path = 'D:\zig'

# 2025-01-19
$env:path += ';C:\Program Files\Python311'
$env:path += ';C:\Program Files\Python311\Scripts'
$env:path += ';c:\users\steven\appdata\roaming\python\python311\scripts'

# 2025-01-18
$env:path += ';C:\Program Files\Mullvad VPN\resources'

# 2025-01-17
$env:path += ';C:\Users\Steven\go\bin'

# 2025-01-14
$env:path += ';D:\MinGit\mingw64\bin'

# 2025-01-13
$env:path += ';D:\bin'

# 2025-01-12
$env:path += ';D:\go\bin'

# 2025-01-11
$env:path += ';D:\vim'

## 2024-9-21

Set-PSReadLineOption -AddToHistoryHandler $null
$env:RIPGREP_CONFIG_PATH = 'C:\Users\Steven\ripgrep.txt'
$MaximumHistoryCount = 9999

# git diff unicode
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()

# git commit -v
$env:EDITOR = 'gvim'

Set-PSReadLineKeyHandler Ctrl+UpArrow {
   Set-Location ..
   [Microsoft.PowerShell.PSConsoleReadLine]::InvokePrompt()
}

# 2024-9-10
Get-Alias | Remove-Alias -Force

# 2024-9-10
Set-PSReadLineOption -PredictionSource None
