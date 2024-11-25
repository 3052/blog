
# 2024-11-24
$env:PATH = @(
   'C:\Program Files\Mullvad VPN\resources'
   'C:\Users\Steven\go\bin'
   'C:\python'
   'C:\python\Scripts'
   'D:\MinGit\mingw64\bin'
   'D:\bin'
   'D:\go\bin'
   'D:\vim'
) -Join ';'

## 2024-9-21

Set-PSReadLineOption -AddToHistoryHandler $null
$env:RIPGREP_CONFIG_PATH = 'C:\Users\Steven\ripgrep.txt'
$MaximumHistoryCount = 9999
### `git diff` unicode
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()
### git commit -v
$env:EDITOR = 'gvim'

Set-PSReadLineKeyHandler Ctrl+UpArrow {
   Set-Location ..
   [Microsoft.PowerShell.PSConsoleReadLine]::InvokePrompt()
}

# 2024-9-10
Get-Alias | Remove-Alias -Force
Set-PSReadLineOption -PredictionSource None

# 2023-05-10
$env:LESS = -join @(
   # Quit if entire file fits on first screen.
   'F'
   # Output "raw" control characters.
   'R'
   # Don't use termcap init/deinit strings.
   'X'
   # Ignore case in searches that do not contain uppercase.
   'i'
)
