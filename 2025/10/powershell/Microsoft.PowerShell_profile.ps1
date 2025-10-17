# 2025-10-16
$env:path = 'C:\Users\Steven\.cargo\bin'

# 2025-10-10

$env:RIPGREP_CONFIG_PATH = 'C:\Users\Steven\Documents\PowerShell\ripgrep.txt'

# disable auto complete
Set-PSReadLineOption -PredictionSource None

$MaximumHistoryCount = 9999

Set-PSReadLineOption -AddToHistoryHandler $null

# git diff unicode
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()

$env:path += ';C:\Users\Steven\AppData\Local\Programs\Python\Python311'
$env:path += ';C:\Users\Steven\AppData\Local\Programs\Python\Python311\Scripts'

Get-Alias | Remove-Alias -Force

Set-PSReadLineKeyHandler Ctrl+UpArrow {
   Set-Location ..
   [Microsoft.PowerShell.PSConsoleReadLine]::InvokePrompt()
}

# git commit -v
$env:EDITOR = 'gvim'

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

# 2025-03-07
# go.dev/doc/go1.13
$env:GOPROXY = 'direct'
$env:GOSUMDB = 'off'

