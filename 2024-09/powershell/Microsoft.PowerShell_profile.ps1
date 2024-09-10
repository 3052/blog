# 2024-9-10
Get-Alias | Remove-Alias -Force
Set-PSReadLineOption -PredictionSource None

# 2023-03-24
$env:PATH = @(
   'C:\Users\Steven\go\bin'
   'C:\python'
   'C:\python\Scripts'
   'D:\MinGit\mingw64\bin'
   'D:\bin'
   'D:\go\bin'
   'D:\php'
   'D:\vim'
) -Join ';'

# 2023-06-18
$env:RIPGREP_CONFIG_PATH = 'C:\Users\Steven\ripgrep.txt'

# 2023-06-06
$MaximumHistoryCount = 9999

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

# 2023-05-10
Set-PSReadLineKeyHandler Ctrl+UpArrow {
   Set-Location ..
   [Microsoft.PowerShell.PSConsoleReadLine]::InvokePrompt()
}

# 2023-05-10
# `git diff` unicode
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new()

# 2023-05-10
# git commit -v
$env:EDITOR = 'gvim'
