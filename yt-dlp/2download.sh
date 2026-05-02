#!/usr/bin/env bash
set -euo pipefail

# 脚本所在目录（无论从哪里执行都正确）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

CONF="$SCRIPT_DIR/yt-dlp.conf"
URLS="$SCRIPT_DIR/urls.txt"
COOKIES="$SCRIPT_DIR/cookies.txt"

echo "============================================"
echo "           yt-dlp 批量下载工具"
echo "============================================"
echo "[输出目录] D:\\yt-dlpDown"
echo "[URL 列表] $URLS"
echo ""

# cookies priority: cookies.txt > Chrome(if closed) > Firefox > Edge
# NOTE: Chrome DB is exclusively locked while Chrome is running.
#       To avoid closing Chrome every time, log into YouTube in Firefox once.
CHROME_RUNNING=false
if tasklist.exe 2>/dev/null | grep -qi "chrome.exe"; then CHROME_RUNNING=true; fi

if [[ -f "$COOKIES" ]]; then
    echo "[Cookies] using file: cookies.txt (override)"
    COOKIE_ARG="--cookies $COOKIES"
elif [[ -d "$LOCALAPPDATA/Google/Chrome/User Data" ]] && [[ "$CHROME_RUNNING" == "false" ]]; then
    echo "[Cookies] reading Chrome cookies"
    COOKIE_ARG="--cookies-from-browser chrome"
elif [[ -d "$APPDATA/Mozilla/Firefox/Profiles" ]]; then
    if [[ "$CHROME_RUNNING" == "true" ]]; then
        echo "[Cookies] Chrome is running (DB locked) -> falling back to Firefox"
        echo "          TIP: log into YouTube in Firefox once to avoid this fallback"
    else
        echo "[Cookies] reading Firefox cookies"
    fi
    COOKIE_ARG="--cookies-from-browser firefox"
elif [[ "$CHROME_RUNNING" == "true" ]]; then
    echo "[ERROR] Chrome is running and Firefox is not installed."
    echo "        Please close Chrome and retry, or install Firefox and log into YouTube."
    exit 1
else
    echo "[Cookies] reading Edge cookies"
    COOKIE_ARG="--cookies-from-browser edge"
fi

echo ""
echo "开始下载..."
echo "============================================"

# shellcheck disable=SC2086
yt-dlp \
    --config-location "$CONF" \
    $COOKIE_ARG \
    --batch-file "$URLS"


# post-process: convert SRT -> styled ASS (green, adaptive size) + bilingual
echo ""
echo "[字幕] 处理字幕..."
python "$SCRIPT_DIR/merge_subs.py" "D:/yt-dlpDown"

echo ""
echo "============================================"
echo "全部完成，输出目录: D:\\yt-dlpDown"
echo "============================================"
