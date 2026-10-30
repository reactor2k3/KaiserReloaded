# Startet Spielserver (Port 8080) und Vite-Dev-Server (Port 5173) in zwei eigenen Fenstern.
# Danach im Browser http://localhost:5173 öffnen; zweiter Spieler in einem weiteren Tab.
# Voraussetzungen: Go 1.27 und Node 24. Optional ein portables Go: $env:KAISER_GO = Ordner mit go.exe,
# sonst wird go aus dem PATH genommen.
$root = $PSScriptRoot
$prefix = if ($env:KAISER_GO) { "`$env:PATH='$($env:KAISER_GO);'+`$env:PATH; `$env:GOTOOLCHAIN='local'; " } else { "" }

$server = $prefix + "Set-Location '$root\server'; go run ./cmd/kaiserd -data ..\data -static ''"
Start-Process pwsh -ArgumentList '-NoExit', '-Command', $server

if (-not (Test-Path "$root\client\node_modules")) { npm --prefix "$root\client" install }
Start-Process pwsh -ArgumentList '-NoExit', '-Command', "Set-Location '$root\client'; npm run dev"
