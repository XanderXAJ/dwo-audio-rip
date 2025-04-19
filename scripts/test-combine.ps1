param (
	[Parameter(Mandatory = $true)][string]$directory
)

# Combine all individual stems of a song
# Skip stem 00 as it appears to be digitally silent
ffmpeg -y `
	-channel_layout stereo -i "$directory/0xeda82cc0.srsa#23_bgm021-02.wav" `
	-channel_layout stereo -i "$directory/0xeda82cc0.srsa#23_bgm021-04.wav" `
	-channel_layout stereo -i "$directory/0xeda82cc0.srsa#23_bgm021-06.wav" `
	-channel_layout stereo -i "$directory/0xeda82cc0.srsa#23_bgm021-08.wav" `
	-channel_layout stereo -i "$directory/0xeda82cc0.srsa#23_bgm021-10.wav" `
	-filter_complex amix=inputs=5 -ac 2 `
	"$directory/0xeda82cc0.srsa#23_bgm021.flac"
