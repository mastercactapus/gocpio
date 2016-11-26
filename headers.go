package cpio

//go:generate stringer -type EncodingType

import "time"

// EncodingType is the header encoding type
type EncodingType int

// Header encoding types
const (
	// EncodingTypeASCIISUSv2 is also known as "odc" or "old character" format
	EncodingTypeASCIISUSv2 EncodingType = iota

	// EncodingTypeASCIISVR4 is also known as "newc" or "new character" format
	EncodingTypeASCIISVR4

	// EncodingTypeASCIISVR4CRC is also known as "crc" format
	EncodingTypeASCIISVR4CRC
	EncodingTypeBinaryLE
	EncodingTypeBinaryBE
)

// Header is a universal cpio header structure
//
// DevMinor and RDevMinor are only relevant for types:
// - EncodingTypeASCIISVR4
// - EncodingTypeASCIISVR4CRC
//
// Furthermore, Checksum is only valid for: EncodingTypeASCIISVR4CRC
type Header struct {
	Name      string       // name of header file entry
	Mode      int64        // permission and mode bits
	DevMajor  int          // Device number (major) from disk
	DevMinor  int          // Device number (minor) from disk
	Inode     int          // inode number from disk
	UID       int          // user id of owner
	GID       int          // group id of owner
	NLink     int          // number of links to this file
	RDevMajor int          // associated device number (major) for special and character entries
	RDevMinor int          // associated device number (minor) for special and character entries
	ModTime   time.Time    // modified time
	Size      int64        // length in bytes
	Checksum  int          // checksum (if `Encoding` is `EncodingTypeASCIISVR4CRC`)
	Encoding  EncodingType // encoding type for the header
}

type binaryHeader struct {
	Dev      uint16
	Inode    uint16
	Mode     uint16
	UID      uint16
	GID      uint16
	NLink    uint16
	RDev     uint16
	ModTime  [2]uint16
	Namesize uint16
	Filesize [2]uint16
}

type asciiSUSv2Header struct {
	Dev      [6]byte
	Inode    [6]byte
	Mode     [6]byte
	UID      [6]byte
	GID      [6]byte
	NLink    [6]byte
	RDev     [6]byte
	ModTime  [11]byte
	Namesize [6]byte
	Filesize [11]byte
}

type asciiSVR4Header struct {
	Inode     [8]byte
	Mode      [8]byte
	UID       [8]byte
	GID       [8]byte
	NLink     [8]byte
	ModTime   [8]byte
	Filesize  [8]byte
	DevMajor  [8]byte
	DevMinor  [8]byte
	RDevMajor [8]byte
	RDevMinor [8]byte
	Namesize  [8]byte
	Checksum  [8]byte
}
