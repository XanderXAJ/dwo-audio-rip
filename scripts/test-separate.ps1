param (
	[Parameter(Mandatory = $true)][string]$path,
	[int]$streamIndex = 0,
	[int]$loops = 2
)

vgmstream-cli "$path" -s "$streamIndex" -l "$loops" -o output_orig.wav
ffmpeg -y `
	-err_detect ignore_err `
	-i output_orig.wav `
	-af "pan=stereo|c0<c0+c2+c4+c6+c8+c10|c1<c1+c3+c5+c7+c9+c11" `
	output.wav

#ffmpeg -i input1.wav -i input2.wav -filter_complex "[0:a][1:a]amerge=inputs=2, pan=stereo | c0<c0+c2 | c1<c1+c3[a]" -map "[a]" output.mp3
