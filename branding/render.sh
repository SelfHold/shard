#!/usr/bin/env bash
# Renders the committed PNG icons from the SVG sources next to this script
# (run it through `make icons` after changing an SVG). Needs rsvg-convert (librsvg).
#   shard_icon.svg                  -> app/assets/icon/shard.png (window icon, desktop entry),
#                                      Android legacy launcher icons
#   shard_icon_{fore,back}ground.svg -> Android adaptive launcher icon layers (API 26+)
#   shard_tray.svg (if present)      -> app/assets/tray/tray.png
# Linux packages install shard_icon.svg itself as the scalable hicolor icon.
set -euo pipefail
cd "$(dirname "$0")/.."
APP=shard
RES=app/android/app/src/main/res

# rsvg-convert crashes on the libraries a snap (VS Code) puts into the environment.
unset_snap=()
while IFS='=' read -r k v; do
  [[ "$k" != PATH && "$v" == */snap/* ]] && unset_snap+=(-u "$k")
done < <(env)
render() { # svg size out
  mkdir -p "$(dirname "$3")"
  env "${unset_snap[@]}" rsvg-convert -w "$2" -h "$2" "branding/$1" -o "$3"
}

render "${APP}_icon.svg" 256 "app/assets/icon/${APP}.png"
[[ -f "branding/${APP}_tray.svg" ]] && render "${APP}_tray.svg" 64 app/assets/tray/tray.png

# Android: launcher icons are 48 dp, adaptive layers 108 dp.
for d in mdpi:1 hdpi:1.5 xhdpi:2 xxhdpi:3 xxxhdpi:4; do
  name=${d%%:*} scale=${d#*:}
  px() { awk -v a="$1" -v s="$scale" 'BEGIN { printf "%d", a * s + 0.5 }'; }
  render "${APP}_icon.svg" "$(px 48)" "$RES/mipmap-$name/ic_launcher.png"
  render "${APP}_icon_foreground.svg" "$(px 108)" "$RES/mipmap-$name/ic_launcher_foreground.png"
  render "${APP}_icon_background.svg" "$(px 108)" "$RES/mipmap-$name/ic_launcher_background.png"
done
echo "icons rendered from branding/"
