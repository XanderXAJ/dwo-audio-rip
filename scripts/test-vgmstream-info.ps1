param (
	[Parameter(Mandatory = $true)][string]$path,
	[int]$streamIndex = 0
)

vgmstream-cli -I -m "$path" -s "$streamIndex"
