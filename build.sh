#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-1.0.0}"

GO=$(command -v go 2>/dev/null || command -v /usr/local/go/bin/go 2>/dev/null || command -v /usr/lib/go/bin/go 2>/dev/null || true)
if [ -z "$GO" ]; then
  echo "Go not found. Install: sudo dnf install golang"
  exit 1
fi

ROOT="$(cd "$(dirname "$0")" && pwd)"

# Indirme motoru saf Go (internal/engine + internal/td): hazirlik adimi,
# gomulu yt-dlp/ffmpeg, ek tag yok. Tek duz `go build` kendi kendine yeterli
# binary uretir.

echo "Select target OS:"
echo "1) Linux"
echo "2) Windows"
echo "3) macOS"
read -p "Choice [1-3]: " os_choice

case "$os_choice" in
  1)
    echo "Select format:"
    echo "1) AppImage"
    echo "2) RPM"
    echo "3) tar.gz"
    echo "4) Binary only"
    read -p "Choice [1-4]: " fmt

    mkdir -p build
    echo "==> Building binary..."
    CGO_ENABLED=1 $GO build -ldflags="-s -w -X main.version=$VERSION" -o build/music-le .
    upx --best --lzma build/music-le 2>/dev/null || true

    case "$fmt" in
      1)
        echo "==> Building AppImage..."
        APPDIR=build/MusicLe.AppDir
        mkdir -p $APPDIR/usr/bin $APPDIR/usr/lib $APPDIR/usr/share/applications $APPDIR/usr/share/icons/hicolor/1024x1024/apps

        # NOT: Sistem kutuphaneleri (orn. libasound) BILEREK paketlenmiyor:
        # derleme makinesindeki kopya daha yeni glibc isteyip eski
        # dagitimlarda uygulamayi acilmaz yapabiliyor.
        cp build/music-le $APPDIR/usr/bin/
        cp assets/MusicLe.png $APPDIR/musicle.png
        ln -sf musicle.png $APPDIR/.DirIcon
        ln -sf musicle.png $APPDIR/usr/share/icons/hicolor/1024x1024/apps/musicle.png

        cat > $APPDIR/AppRun << 'APPRUN'
#!/bin/sh
HERE="$(dirname "$(readlink -f "$0")")"
export APPDIR="$HERE"
# Args verilince (--version, --help ...) veya gercek bir terminal varsa
# dogrudan calistir; sadece masaustunden tiklaninca terminal ac.
if [ $# -gt 0 ] || [ -t 0 ]; then
  exec "$HERE/usr/bin/music-le" "$@"
else
  for term in x-terminal-emulator gnome-terminal konsole xterm alacritty kitty; do
    if command -v "$term" >/dev/null 2>&1; then
      exec "$term" -e "$HERE/usr/bin/music-le" "$@"
    fi
  done
  exec "$HERE/usr/bin/music-le" "$@"
fi
APPRUN
        chmod +x $APPDIR/AppRun

        cat > $APPDIR/musicle.desktop << 'DESKTOP'
[Desktop Entry]
Name=MusicLe
Exec=music-le
Icon=musicle
Terminal=true
Type=Application
Categories=AudioVideo;Audio;Music;Player;
StartupNotify=false
DESKTOP
        cp $APPDIR/musicle.desktop $APPDIR/usr/share/applications/
        if command -v appimagetool >/dev/null 2>&1; then
            APPIMAGETOOL="$(command -v appimagetool)"
        else
            APPIMAGETOOL="$(pwd)/build/appimagetool"
            if [ ! -s "$APPIMAGETOOL" ]; then
                echo "==> appimagetool indiriliyor..."
                APPIMAGETOOL_URL="https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage"
                if command -v curl >/dev/null 2>&1; then
                    curl -fL --retry 3 --retry-delay 2 --connect-timeout 15 --max-time 120 "$APPIMAGETOOL_URL" -o "$APPIMAGETOOL" || rm -f "$APPIMAGETOOL"
                fi
                if [ ! -s "$APPIMAGETOOL" ] && command -v wget >/dev/null 2>&1; then
                    wget -O "$APPIMAGETOOL" "$APPIMAGETOOL_URL" || rm -f "$APPIMAGETOOL"
                fi
                if [ ! -s "$APPIMAGETOOL" ]; then
                    echo "HATA: appimagetool indirilemedi. Internet baglantisini kontrol et"
                    exit 1
                fi
                chmod +x "$APPIMAGETOOL"
            fi
        fi
        cd build
        "$APPIMAGETOOL" --appimage-extract > /dev/null 2>&1 || true
        # Hedef AppImage eski bir çalışan/mountlu sürüm olabilir (ETXTBSY);
        # önce geçici isimle üret, sonra mv -f ile üzerine yaz.
        TMP_APPIMAGE=MusicLe.build.AppImage
        rm -f "$TMP_APPIMAGE"
        # type2 runtime'i onbellege indir; appimagetool'un kendi indirmesi
        # zaman zaman 503 aliyor. Basarisiz olursa appimagetool kendi dener.
        APPIMAGE_RUNTIME="$(pwd)/runtime-x86_64"
        if [ ! -s "$APPIMAGE_RUNTIME" ]; then
            echo "==> AppImage runtime indiriliyor..."
            RUNTIME_URL="https://github.com/AppImage/type2-runtime/releases/download/continuous/runtime-x86_64"
            rm -f "$APPIMAGE_RUNTIME"
            if command -v curl >/dev/null 2>&1; then
                curl -fL --retry 3 --retry-delay 2 --connect-timeout 15 --max-time 120 "$RUNTIME_URL" -o "$APPIMAGE_RUNTIME" || rm -f "$APPIMAGE_RUNTIME"
            fi
            if [ ! -s "$APPIMAGE_RUNTIME" ] && command -v wget >/dev/null 2>&1; then
                wget -O "$APPIMAGE_RUNTIME" "$RUNTIME_URL" || rm -f "$APPIMAGE_RUNTIME"
            fi
            [ -s "$APPIMAGE_RUNTIME" ] && chmod +x "$APPIMAGE_RUNTIME"
        fi
        if [ -s "$APPIMAGE_RUNTIME" ]; then
            ./squashfs-root/AppRun --runtime-file "$APPIMAGE_RUNTIME" -n MusicLe.AppDir "$TMP_APPIMAGE"
        else
            echo "UYARI: runtime indirilemedi; appimagetool kendi deneyecek."
            ./squashfs-root/AppRun -n MusicLe.AppDir "$TMP_APPIMAGE"
        fi
        mv -f "$TMP_APPIMAGE" MusicLe.AppImage
        rm -rf squashfs-root MusicLe.AppDir
        cd ..
        echo "==> build/MusicLe.AppImage"
        ;;
      2)
        echo "==> Building RPM..."
        mkdir -p build/rpm/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
        cat > build/rpm/SPECS/musicle.spec << SPEC
Name: musicle
Version: $VERSION
Release: 1
Summary: Terminal music player
License: MIT
BuildArch: x86_64
%description
MusicLe - terminal music player
%install
mkdir -p %{buildroot}/usr/local/bin
cp %{_sourcedir}/music-le %{buildroot}/usr/local/bin/
%files
/usr/local/bin/music-le
SPEC
        cp build/music-le build/rpm/SOURCES/
        rpmbuild --define "_topdir $(pwd)/build/rpm" -bb build/rpm/SPECS/musicle.spec
        cp build/rpm/RPMS/x86_64/*.rpm build/
        echo "==> build/*.rpm"
        ;;
      3)
        echo "==> Packing tar.gz..."
        mkdir -p build/musicle_Linux_x86_64
        cp build/music-le build/musicle_Linux_x86_64/music-le
        cp README.md build/musicle_Linux_x86_64/ 2>/dev/null || true
        cd build && tar czf musicle_Linux_x86_64.tar.gz musicle_Linux_x86_64 && cd ..
        echo "==> build/musicle_Linux_x86_64.tar.gz"
        ;;
      4)
        echo "==> build/music-le"
        ;;
    esac
    ;;
  2)
    echo "Select format:"
    echo "1) .exe (portable)"
    echo "2) .exe + .zip"
    read -p "Choice [1-2]: " fmt
    mkdir -p build
    echo "==> Building..."
    GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $GO build -ldflags="-s -w -X main.version=$VERSION" -o build/music-le.exe .
    upx --best --lzma build/music-le.exe 2>/dev/null || true
    if [ "$fmt" = "2" ]; then
      mkdir -p build/musicle_Windows_x86_64
      cp build/music-le.exe build/musicle_Windows_x86_64/music-le.exe
      cd build && zip -r musicle_Windows_x86_64.zip musicle_Windows_x86_64 && cd ..
      echo "==> build/musicle_Windows_x86_64.zip"
    else
      echo "==> build/music-le.exe"
    fi
    ;;
  3)
    echo "Select format:"
    echo "1) tar.gz"
    echo "2) Binary only"
    read -p "Choice [1-2]: " fmt
    mkdir -p build
    echo "==> Building..."
    GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $GO build -ldflags="-s -w -X main.version=$VERSION" -o build/music-le-darwin .
    upx --best --lzma build/music-le-darwin 2>/dev/null || true
    if [ "$fmt" = "1" ]; then
      mkdir -p build/musicle_macOS_x86_64
      cp build/music-le-darwin build/musicle_macOS_x86_64/music-le
      cp README.md build/musicle_macOS_x86_64/ 2>/dev/null || true
      cd build && tar czf musicle_macOS_x86_64.tar.gz musicle_macOS_x86_64 && cd ..
      echo "==> build/musicle_macOS_x86_64.tar.gz"
    else
      echo "==> build/music-le-darwin"
    fi
    ;;
esac
