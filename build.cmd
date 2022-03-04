@echo off
FOR /F "tokens=*" %%a in ('git rev-parse --short HEAD') do SET COMMIT=%%a
FOR /F "tokens=*" %%a in ('git describe --tags --always --abbrev^=0') do SET VERSION=%%a

set "FLAGS=-X 'main.Version=%VERSION%' -X 'main.BuildTime=%DATE% %TIME:~0,8%' -X 'main.Commit=%COMMIT%'"

go build -v -ldflags="%FLAGS%"