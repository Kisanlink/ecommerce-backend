@echo off
echo Setting up Firebase environment variables for Windows...

REM Set Firebase Web API Key
set FIREBASE_WEB_API_KEY=AIzaSyCPzNrFV3Q6a4rcAQF-60G3h3K1WBQqQFI

REM Set Firebase Project ID
set FIREBASE_PROJECT_ID=amrti-m1355

echo.
echo Environment variables set:
echo FIREBASE_WEB_API_KEY=%FIREBASE_WEB_API_KEY%
echo FIREBASE_PROJECT_ID=%FIREBASE_PROJECT_ID%
echo.
echo To make these permanent, add them to your system environment variables.
echo Or run your Go server from this command prompt.
echo.
echo Starting Go server...
go run cmd/server/main.go
