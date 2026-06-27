#!/usr/bin/env bash
set -e
cd "$(dirname "$0")"
mkdir -p data
if [ ! -f "./myhomeweb" ]; then
    echo "ERROR: myhomeweb no encontrado. Ejecuta 'go build' primero."
    exit 1
fi
echo "MyHomeWeb — http://localhost:19484"
(sleep 2 && xdg-open "http://localhost:19484" 2>/dev/null || open "http://localhost:19484" 2>/dev/null) &
./myhomeweb
