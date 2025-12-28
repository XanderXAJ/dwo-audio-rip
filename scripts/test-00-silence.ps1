param(
	# Folder to search (default: current directory)
	[string]$Path = ".",

	# Search recursively
	[switch]$Recurse,

	# Path to sox_ng if it's not on PATH
	[string]$SoxNg = "sox_ng"
)

# Get candidate files whose *name* contains "_02_"
$gciParams = @{
	Path   = $Path
	File   = $true
	Filter = "*_02_*"
}
if ($Recurse) { $gciParams.Recurse = $true }

Get-ChildItem @gciParams | ForEach-Object {
	$src = $_

	# Replace the first occurrence of "_02_" with "_00_" in the filename
	$targetName = $src.Name -replace "_02_", "_00_"
	$targetPath = Join-Path -Path $src.DirectoryName -ChildPath $targetName

	if (-not (Test-Path -LiteralPath $targetPath)) {
		Write-Warning "Missing target for '$($src.FullName)' -> '$targetPath'"
		return
	}

	# Run sox_ng stat (stats usually go to stderr, so capture both)
	$out = & $SoxNg $targetPath -n stat 2>&1

	# Extract min/max amplitude lines
	$minLine = ($out | Where-Object { $_ -match "^\s*Minimum amplitude:\s*" } | Select-Object -First 1)
	$maxLine = ($out | Where-Object { $_ -match "^\s*Maximum amplitude:\s*" } | Select-Object -First 1)

	$min = if ($minLine -match "Minimum amplitude:\s*([-\d\.eE]+)") { $Matches[1] } else { $null }
	$max = if ($maxLine -match "Maximum amplitude:\s*([-\d\.eE]+)") { $Matches[1] } else { $null }

	$isSilent = ($min -eq "0.000000" -and $max -eq "0.000000")

	# Output result object (to be later formatted)
	[pscustomobject]@{
		SourceFile = $src.Name
		TargetFile = $targetName
		Silent     = $isSilent
		MinAmp     = $min
		MaxAmp     = $max
		TargetPath = $targetPath
	}
} | Format-Table -AutoSize # Output tabulated results
