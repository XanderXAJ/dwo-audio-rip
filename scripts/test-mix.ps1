param (
	[Parameter(Mandatory = $true)][int]$streamIndex,
	[int]$subIndex = 0,
	[string]$inputDirectory = "rip",
	[string]$outputDirectory = "mix",
	[int]$loops = 2
)

ffmpeg -y `
	-i "${inputDirectory}/${streamIndex}_$("{0:D2}" -f $subIndex)_intro.wav" `
	-i "${inputDirectory}/${streamIndex}_$("{0:D2}" -f $subIndex)_loop.wav" `
	-filter_complex "
		[1]aloop=loop=${loops}:size=2e9[loops];
		[0][loops]concat=v=0:a=1;
		" `
	"${outputDirectory}/${streamIndex}_$("{0:D2}" -f $subIndex)_mix.flac"
