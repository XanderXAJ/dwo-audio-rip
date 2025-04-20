param (
	[Parameter(Mandatory = $true)][int]$streamIndex,
	[string]$inputDirectory = "rip",
	[string]$outputDirectory = "mix",
	[int]$loops = 2
)

ffmpeg -y `
	-i "${inputDirectory}/${streamIndex}_02_intro.wav" `
	-i "${inputDirectory}/${streamIndex}_04_intro.wav" `
	-i "${inputDirectory}/${streamIndex}_06_intro.wav" `
	-i "${inputDirectory}/${streamIndex}_08_intro.wav" `
	-i "${inputDirectory}/${streamIndex}_10_intro.wav" `
	-i "${inputDirectory}/${streamIndex}_02_loop.wav" `
	-i "${inputDirectory}/${streamIndex}_04_loop.wav" `
	-i "${inputDirectory}/${streamIndex}_06_loop.wav" `
	-i "${inputDirectory}/${streamIndex}_08_loop.wav" `
	-i "${inputDirectory}/${streamIndex}_10_loop.wav" `
	-filter_complex "
		[0][1][2][3][4]amix=inputs=5[intro];
		[5][6][7][8][9]amix=inputs=5[loop];
		[loop]aloop=loop=${loops}:size=2e9[loops];
		[intro][loops]concat=v=0:a=1;
		" `
	"${outputDirectory}/${streamIndex}_mix.flac"
