param (
	[Parameter(Mandatory = $true)][string]$path,
	[int]$streamIndex = 0
)

vgmstream-cli "$path" -s "$streamIndex" -k -2 -l 1 -f 0 -o "?s_loop.wav"
