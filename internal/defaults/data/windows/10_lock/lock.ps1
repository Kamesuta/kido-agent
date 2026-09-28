# 画面をロックする(Windows キー + L と同じ)。
# 引数を引用符で囲むのは、PowerShell が「a,b」を配列として空白でつないで渡してしまうため。
rundll32.exe 'user32.dll,LockWorkStation'
