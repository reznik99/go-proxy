
$buildPath="Build/proxy"

$Env:GOOS="linux"
$Env:GOARCH="arm"
$Env:GOARM="7"

go build -o $buildPath

Write-Output "Successfully built go-proxy for $Env:GOOS:$Env:GOARCH-$Env:GOARM at $buildPath"