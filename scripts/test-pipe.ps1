param (
	[Parameter(Mandatory = $true)][string]$path,
	[int]$streamIndex = 0,
	[int]$loops = 2
)

# Findings: This is very likely to break as vgmstream outputs an invalid data frame if the
# WAV exceeds 4GB. It doesn't support e.g. https://en.wikipedia.org/wiki/RF64.
#   https://github.com/vgmstream/vgmstream/blob/9355795734ec58cb09d83c84794125e7e83f0048/cli/vgmstream_cli.c#L366-L388
vgmstream-cli "$path" -s "$streamIndex" -l "$loops" -p |
ffmpeg -y `
	-err_detect ignore_err `
	-i - `
	-af "pan=stereo|c0<c0+c2+c4+c6+c8+c10|c1<c1+c3+c5+c7+c9+c11" `
	output.wav
