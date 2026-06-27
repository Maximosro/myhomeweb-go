@echo off
title MyHomeWeb
cd /d "%~dp0"
if not exist "data" mkdir data
if not exist "myhomeweb.exe" (
    echo ERROR: myhomeweb.exe no encontrado. Ejecuta 'go build' primero.
    pause
    exit /b 1
)
echo MyHomeWeb — http://localhost:19484
start "" http://localhost:19484
myhomeweb.exe
pause
