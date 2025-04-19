param (
	[Parameter(Mandatory = $true)][string]$path,
	[int]$streamIndex = 0,
	[int]$loops = 2
)

vgmstream-cli -I -m "$path" -s "$streamIndex" -l "$loops"
