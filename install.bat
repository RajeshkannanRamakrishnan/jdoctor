@echo off
set APP_NAME=jdoctor.exe
set INSTALL_DIR=%USERPROFILE%\bin
set SRC_DIR=./cmd/jdoctor

echo Building %APP_NAME%...
go build -o %APP_NAME% %SRC_DIR%

if %errorlevel% neq 0 (
    echo Build failed.
    exit /b %errorlevel%
)

echo Build successful!

if not exist "%INSTALL_DIR%" (
    echo Creating directory %INSTALL_DIR%...
    mkdir "%INSTALL_DIR%"
)

echo Installing to %INSTALL_DIR%...
move /Y %APP_NAME% "%INSTALL_DIR%\"

echo %APP_NAME% installed successfully!
echo.
echo Ensure %INSTALL_DIR% is in your User PATH environment variable to run it from anywhere.
pause
