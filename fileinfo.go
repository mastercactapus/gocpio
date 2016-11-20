package cpio

import (
	"fmt"
	"os"
	"path"
	"time"
)

const (
	modeRegular   = 0100000
	modeDirectory = 0040000
	modeSymlink   = 0120000
	modeCharDev   = 0020000
	modeBlkDev    = 0060000
	modeFIFO      = 0010000
	modeSocket    = 0140000
	modeSUID      = 0004000
	modeSGID      = 0002000
	modeSticky    = 0001000
)

// FileInfoHeader creates a partially populated Header
//
// Note for symlinks, the link body must be stored as file data
func FileInfoHeader(fi os.FileInfo) (*Header, error) {
	fm := fi.Mode()
	h := &Header{
		Name:    fi.Name(),
		ModTime: fi.ModTime(),
		Mode:    int64(fm.Perm()),
	}
	switch {
	case fm.IsRegular():
		h.Mode |= modeRegular
		h.Size = fi.Size()
	case fi.IsDir():
		h.Mode |= modeDirectory
		h.Name += "/"
	case fm&os.ModeSymlink != 0:
		h.Mode |= modeSymlink
	case fm&os.ModeDevice != 0:
		if fm&os.ModeCharDevice != 0 {
			h.Mode |= modeCharDev
		} else {
			h.Mode |= modeBlkDev
		}
	case fm&os.ModeNamedPipe != 0:
		h.Mode |= modeFIFO
	case fm&os.ModeSocket != 0:
		h.Mode |= modeSocket
	default:
		return nil, fmt.Errorf("github.com/mastercactapus/gocpio: unknown file mode %v", fm)
	}

	if fm&os.ModeSetuid != 0 {
		h.Mode |= modeSUID
	}
	if fm&os.ModeSetgid != 0 {
		h.Mode |= modeSGID
	}
	if fm&os.ModeSticky != 0 {
		h.Mode |= modeSticky
	}

	if sys, ok := fi.Sys().(*Header); ok {
		// if this FileInfo came from Header, use the original to populate
		// remaining fields
		h.Checksum = sys.Checksum
		h.DevMajor = sys.DevMajor
		h.DevMinor = sys.DevMinor
		h.GID = sys.GID
		h.Inode = sys.Inode
		h.NLink = sys.NLink
		h.RDevMajor = sys.RDevMajor
		h.RDevMinor = sys.RDevMinor
		h.Encoding = sys.Encoding
		h.UID = sys.UID
	}

	return h, nil
}

type headerFileInfo struct {
	h *Header
}

// FileInfo returns an os.FileInfo for the Header
func (h *Header) FileInfo() os.FileInfo {
	return headerFileInfo{h}
}

func (fi headerFileInfo) Size() int64        { return fi.h.Size }
func (fi headerFileInfo) ModTime() time.Time { return fi.h.ModTime }
func (fi headerFileInfo) IsDir() bool        { return fi.Mode().IsDir() }
func (fi headerFileInfo) Sys() interface{}   { return fi.h }

// Name returns the base name of the file
func (fi headerFileInfo) Name() string {
	if fi.IsDir() {
		return path.Base(path.Clean(fi.h.Name))
	}
	return path.Base(fi.h.Name)
}

// Mode returns the permission and mode bits for the headerFileInfo
func (fi headerFileInfo) Mode() (mode os.FileMode) {
	// set permission bits
	mode = os.FileMode(fi.h.Mode).Perm()

	// Set setuid, setgid and sticky bits.
	if fi.h.Mode&modeSUID != 0 {
		// setuid
		mode |= os.ModeSetuid
	}
	if fi.h.Mode&modeSGID != 0 {
		// setgid
		mode |= os.ModeSetgid
	}
	if fi.h.Mode&modeSticky != 0 {
		// sticky
		mode |= os.ModeSticky
	}

	// Set file mode bits.
	// clear perm, setuid, setgid and sticky bits.
	m := os.FileMode(fi.h.Mode) &^ 07777
	if m == modeDirectory {
		// directory
		mode |= os.ModeDir
	}
	if m == modeFIFO {
		// named pipe (FIFO)
		mode |= os.ModeNamedPipe
	}
	if m == modeSymlink {
		// symbolic link
		mode |= os.ModeSymlink
	}
	if m == modeBlkDev {
		// device file
		mode |= os.ModeDevice
	}
	if m == modeCharDev {
		// Unix character device
		mode |= os.ModeDevice
		mode |= os.ModeCharDevice
	}
	if m == modeSocket {
		// Unix domain socket
		mode |= os.ModeSocket
	}

	return mode
}
