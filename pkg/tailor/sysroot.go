package tailor

import "fmt"

// sysrootRsyncCmd builds the rsync command that applies a sysroot/ (or dirs/)
// overlay onto the filesystem rooted at dest ("/" in production).
//
// The wardrobe lives in the invoking user's home ("tailor get" chowns it to
// SUDO_USER) but the overlay is applied as root. A plain "rsync -a" preserves
// the owner, group and modes of that checkout, so for anyone running tailor
// through sudo every overlay file, and every directory the overlay touches
// (/, /etc, /etc/skel, /usr, /usr/local, /usr/local/bin, ...), would end up
// owned by that user, who could then replace files that root executes later.
//
// The overlay is therefore always installed as root:root, and modes are
// normalised to what git actually records (only the executable bit):
// directories 0755, files 0644, or 0755 when the executable bit is set.
func sysrootRsyncCmd(sysrootPath, dest string) string {
	return fmt.Sprintf("rsync -aAX --no-o --no-g --chown=root:root --chmod=D755,Fu=rwX,Fgo=rX %s/ %s", sysrootPath, dest)
}
