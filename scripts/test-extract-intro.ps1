param (
	[Parameter(Mandatory = $true)][string]$path,
	[int]$streamIndex = 0
)

vgmstream-cli "$path" -s "$streamIndex" -l "-1" -f 0 -o "?s_intro.wav"
