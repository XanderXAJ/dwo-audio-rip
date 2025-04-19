param (
	[Parameter(Mandatory = $true)][string]$path,
	[int]$loops = 2
)

vgmstream-cli "$path" -l "$loops" -p |
ffmpeg -y `
	-i - `
	output-txtp.wav
