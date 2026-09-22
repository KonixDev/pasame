package main

// Genera el icono y los metadatos del .exe para ambas arquitecturas de Windows.
//go:generate go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.4.1 -64 -o resource_windows_amd64.syso ../../packaging/windows/versioninfo.json
//go:generate go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.4.1 -arm -64 -o resource_windows_arm64.syso ../../packaging/windows/versioninfo.json
