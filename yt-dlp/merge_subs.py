#!/usr/bin/env python3
"""
Subtitle post-processor for yt-dlp downloads.
- Converts SRT to ASS (green color, adaptive font size via PlayRes)
- Auto-detects original language from downloaded .srt files
- If original is Chinese: only generates zh-Hans.ass
- If original is non-Chinese: generates <lang>.ass + zh-Hans.ass + <lang>-zh.ass
- Removes all intermediate .srt and .vtt files

Usage: python merge_subs.py <download_dir>
"""

import json
import re
import subprocess
import sys
from pathlib import Path

_VIDEO_EXTS = (".mp4", ".mkv", ".webm", ".avi", ".mov")

_ASS_HEADER = """\
[Script Info]
ScriptType: v4.00+
WrapStyle: 0
ScaledBorderAndShadow: yes
PlayResX: 1280
PlayResY: 720

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: EnBottom,Arial,30,&H0000FF00,&H000000FF,&H00000000,&H80000000,0,0,0,0,100,100,0,0,1,2.5,0.5,2,128,128,18,1
Style: ZhBottom,Microsoft YaHei,30,&H0000FF00,&H000000FF,&H00000000,&H80000000,0,0,0,0,100,100,0,0,1,2.5,0.5,2,128,128,18,1
Style: BiBottom,Microsoft YaHei,30,&H0000FF00,&H000000FF,&H00000000,&H80000000,0,0,0,0,100,100,0,0,1,2.5,0.5,2,128,128,22,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
"""

def _srt_ts_to_ms(s):
    s = s.strip().replace(".", ",")
    h, m, rest = s.split(":")
    sec, ms_str = rest.split(",")
    return int(h)*3600000 + int(m)*60000 + int(sec)*1000 + int(ms_str[:3])

def _ms_to_ass(ms):
    h = ms // 3600000; ms %= 3600000
    m = ms // 60000;   ms %= 60000
    s = ms // 1000;    cs = (ms % 1000) // 10
    return f"{h}:{m:02d}:{s:02d}.{cs:02d}"

_SRT_RE = re.compile(
    r"\d+\s*\n(\d{2}:\d{2}:\d{2}[,\.]\d{2,3})\s*-->\s*(\d{2}:\d{2}:\d{2}[,\.]\d{2,3})[^\n]*\n([\s\S]*?)(?=\n\s*\n|\Z)",
    re.MULTILINE)
_HTML_RE = re.compile(r"<[^>]+>")

# Language classification
_ZH_LANGS = {"zh-Hans", "zh-Hant", "zh", "zh-CN", "zh-TW"}
_CJK_LANGS = {"zh-Hans", "zh-Hant", "zh", "zh-CN", "zh-TW", "ja", "ko"}

def _orig_font(lang):
    return "Microsoft YaHei" if lang in _CJK_LANGS else "Arial"

def parse_srt(path):
    text = path.read_text(encoding="utf-8-sig").replace("\r\n", "\n").replace("\r", "\n")
    blocks = []
    for m in _SRT_RE.finditer(text.strip()):
        s1, s2, content = m.groups()
        try:
            ms1 = _srt_ts_to_ms(s1); ms2 = _srt_ts_to_ms(s2)
        except Exception:
            continue
        clean = _HTML_RE.sub("", content).strip()
        if clean:
            blocks.append({"ms": ms1, "ms_end": ms2, "s": _ms_to_ass(ms1), "e": _ms_to_ass(ms2),
                           "t": clean.replace("\n", "\\N")})
    return blocks

def _dlg(b, style):
    return f"Dialogue: 0,{b['s']},{b['e']},{style},,0,0,0,,{b['t']}"

def _merge_bilingual(orig_blocks, zh_blocks, orig_lang):
    """Merge orig/zh into single bottom events: original line 1, Chinese line 2."""
    fn = _orig_font(orig_lang)
    boundaries = set()
    for b in orig_blocks + zh_blocks:
        boundaries.add(b['ms'])
        boundaries.add(b['ms_end'])
    boundaries = sorted(boundaries)
    rows = []
    for i in range(len(boundaries) - 1):
        t0, t1 = boundaries[i], boundaries[i + 1]
        if t0 >= t1:
            continue
        orig_t = next((b['t'] for b in orig_blocks if b['ms'] <= t0 < b['ms_end']), '')
        zh_t   = next((b['t'] for b in zh_blocks  if b['ms'] <= t0 < b['ms_end']), '')
        if not orig_t and not zh_t:
            continue
        s, e = _ms_to_ass(t0), _ms_to_ass(t1)
        if orig_t and zh_t:
            text = f"{{\\fn{fn}}}{orig_t}\\N{{\\fnMicrosoft YaHei}}{zh_t}"
        elif orig_t:
            text = f"{{\\fn{fn}}}{orig_t}"
        else:
            text = f"{{\\fnMicrosoft YaHei}}{zh_t}"
        rows.append(f"Dialogue: 0,{s},{e},BiBottom,,0,0,0,,{text}")
    return rows

def write_ass(rows, out):
    out.write_text(_ASS_HEADER + "\n".join(rows) + "\n", encoding="utf-8-sig")
    print(f"  [ok] {out.name}")

def _ffprobe_subs(video_path):
    """Return subtitle stream list [{index, codec, lang}] from video via ffprobe."""
    try:
        r = subprocess.run(
            ["ffprobe", "-v", "quiet", "-print_format", "json",
             "-show_streams", "-select_streams", "s", str(video_path)],
            capture_output=True, text=True, timeout=30
        )
        if r.returncode != 0:
            return []
        streams = []
        for s in json.loads(r.stdout).get("streams", []):
            lang = s.get("tags", {}).get("language", "und")
            streams.append({"index": s["index"], "codec": s.get("codec_name", ""), "lang": lang})
        return streams
    except Exception:
        return []

def extract_embedded_subs(video_path, d, base):
    """Extract soft subtitle streams from video container → SRT files.
    Original video file is NOT modified; subtitle streams remain intact.
    Returns list of extracted SRT paths."""
    streams = _ffprobe_subs(video_path)
    if not streams:
        return []
    print(f"  [embed] {len(streams)} subtitle stream(s) found in {video_path.name}")
    extracted = []
    for st in streams:
        lang = st["lang"]
        out_srt = d / f"{base}.{lang}.srt"
        if out_srt.exists():
            extracted.append(out_srt)
            continue
        try:
            r = subprocess.run(
                ["ffmpeg", "-y", "-v", "quiet",
                 "-i", str(video_path),
                 "-map", f"0:{st['index']}",
                 str(out_srt)],
                capture_output=True, timeout=60
            )
            if r.returncode == 0 and out_srt.exists():
                print(f"  [extract] stream {st['index']} ({lang}) → {out_srt.name}")
                extracted.append(out_srt)
            else:
                print(f"  [warn] failed to extract stream {st['index']} ({lang})")
        except Exception as e:
            print(f"  [warn] extract error: {e}")
    return extracted

def process_dir(d):
    srts = list(d.glob("*.srt"))

    # No external SRT: try extracting soft subtitle streams from video file
    if not srts:
        # Skip if already processed (.ass files present)
        if list(d.glob("*.ass")):
            return
        videos = [f for ext in _VIDEO_EXTS for f in d.glob(f"*{ext}")]
        if not videos:
            return
        video = sorted(videos)[0]
        base_name = video.stem
        print(f"\n[subtitle] {d.name}")
        print(f"  [info] no external subtitles, checking embedded streams in {video.name}")
        extracted = extract_embedded_subs(video, d, base_name)
        if not extracted:
            print("  [skip] no soft subtitle streams found (burned-in subtitles cannot be extracted)")
            return
        srts = list(d.glob("*.srt"))

    # Extract base name from any .LANG.srt file (split at last dot before .srt)
    base = None
    for p in srts:
        stem = p.name[:-4]  # strip .srt
        if "." in stem:
            base = stem.rsplit(".", 1)[0]
            break
    if not base:
        return
    print(f"\n[subtitle] {d.name}")

    # Find best zh source
    zh_src = None
    for sfx in (".zh-Hans.srt", ".zh-Hant.srt", ".zh.srt"):
        p = d / (base + sfx)
        if p.exists():
            zh_src = p; break

    # Find original (non-zh) source — take first alphabetically
    orig_src = None
    orig_lang = None
    for p in sorted(srts):
        if not p.name.startswith(base + "."):
            continue
        lang_tag = p.name[len(base) + 1:-4]  # base.LANG.srt → LANG
        if lang_tag not in _ZH_LANGS:
            orig_src = p
            orig_lang = lang_tag
            break

    zh_blocks   = parse_srt(zh_src)   if zh_src   else []
    orig_blocks = parse_srt(orig_src) if orig_src else []

    if orig_src is None:
        # Chinese original: only zh.ass
        if zh_blocks:
            write_ass([_dlg(b, "ZhBottom") for b in zh_blocks],
                      d / f"{base}.zh-Hans.ass")
        else:
            print("  [skip] no usable subtitle found")
    else:
        # Non-Chinese original: orig.ass + zh-Hans.ass + bilingual
        print(f"  [lang] detected original language: {orig_lang}")
        if orig_blocks:
            write_ass([_dlg(b, "EnBottom") for b in orig_blocks],
                      d / f"{base}.{orig_lang}.ass")
        if zh_blocks:
            write_ass([_dlg(b, "ZhBottom") for b in zh_blocks],
                      d / f"{base}.zh-Hans.ass")
        if orig_blocks and zh_blocks:
            write_ass(_merge_bilingual(orig_blocks, zh_blocks, orig_lang),
                      d / f"{base}.{orig_lang}-zh.ass")
        else:
            print("  [skip] bilingual requires both original and zh subtitles")

    for p in list(d.glob("*.srt")) + list(d.glob("*.vtt")):
        p.unlink(); print(f"  [rm]   {p.name}")

def main():
    if len(sys.argv) < 2:
        print("Usage: merge_subs.py <download_dir>"); sys.exit(1)
    root = Path(sys.argv[1])
    if not root.exists():
        print(f"[error] not found: {root}"); sys.exit(1)
    processed = 0
    for d in sorted(root.iterdir()):
        if not d.is_dir():
            continue
        has_srt   = any(d.glob("*.srt"))
        has_video = any(f for ext in _VIDEO_EXTS for f in d.glob(f"*{ext}"))
        if has_srt or has_video:
            process_dir(d)
            processed += 1
    if processed == 0:
        print("[subtitle] no subtitle or video files found")

if __name__ == "__main__":
    main()
