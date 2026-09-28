# スリープにする。
# rundll32 powrprof.dll,SetSuspendState は、休止状態が有効な PC では
# スリープではなく休止状態に入ってしまうので、.NET の SetSuspendState を使う。
Add-Type -AssemblyName System.Windows.Forms
[System.Windows.Forms.Application]::SetSuspendState('Suspend', $false, $false) | Out-Null
