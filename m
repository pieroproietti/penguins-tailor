#!/bin/sh
set -e
make clean package "$@"

. /etc/os-release

case "$ID" in
    alpine)
        sudo apk add --allow-untrusted penguins-tailor-*.apk
        ;;

    arch)
        sudo pacman -U --noconfirm penguins-tailor-*.pkg.tar.zst
        ;;

    debian)
        sudo dpkg -i penguins-tailor_*.deb
        ;;
    fedora)
        sudo dnf reinstall -y penguins-tailor-*.rpm
        ;;
    opensuse*)
        sudo zypper --no-gpg-checks install -y penguins-tailor-*.rpm
        ;;
    *)
        # fallback su LIKE_ID
        case "$ID_LIKE" in
            *arch*)   sudo pacman -U --noconfirm penguins-tailor-*.pkg.tar.zst ;;
            *debian*|*devuan*|*ubuntu*) sudo dpkg -i penguins-tailor_*.deb ;;
            *fedora*|*rhel*) sudo dnf install -y penguins-tailor-*.rpm ;;
            *) echo "Distro non supportata: $ID"; exit 1 ;;
        esac
        ;;
esac


