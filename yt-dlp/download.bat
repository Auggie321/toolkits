@echo off
chcp 65001 >nul
cd /d "%~dp0"

echo ============================================
echo            yt-dlp 批量下载工具
echo ============================================
echo [输出目录] D:\yt-dlpDown
echo [URL 列表] %~dp0urls.txt
echo.

REM cookies priority: cookies.txt > Chrome(if closed) > Firefox > Edge
if exist "%~dp0cookies.txt" (
    echo [Cookies] using file: cookies.txt (override)
    set "COOKIE_ARG=--cookies "%~dp0cookies.txt""
    goto :do_download
)

if not exist "%LOCALAPPDATA%\Google\Chrome\User Data" goto :try_firefox
tasklist /FI "IMAGENAME eq chrome.exe" 2>nul | find /I "chrome.exe" >nul
if errorlevel 1 (
    echo [Cookies] reading Chrome cookies
    set "COOKIE_ARG=--cookies-from-browser chrome"
    goto :do_download
)
echo [Cookies] Chrome is running (DB locked), falling back to Firefox

:try_firefox
if exist "%APPDATA%\Mozilla\Firefox\Profiles" (
    echo [Cookies] reading Firefox cookies
    set "COOKIE_ARG=--cookies-from-browser firefox"
    goto :do_download
)

REM last resort: Edge
echo [Cookies] reading Edge cookies
set "COOKIE_ARG=--cookies-from-browser edge"

:do_download

echo.
echo 开始下载...
echo ============================================

yt-dlp ^
    --config-location "%~dp0yt-dlp.conf" ^
    %COOKIE_ARG% ^
    --batch-file "%~dp0urls.txt"

echo.
echo ============================================
echo 下载完成，开始处理字幕...
echo ============================================
python "%~dp0merge_subs.py" "D:/yt-dlpDown"

echo.
echo ============================================
echo 全部完成，输出目录: D:\yt-dlpDown
echo ============================================
pause
