package zip

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

var crxMagic = []byte("Cr24")

// IsCRX reports whether data starts with the CRX magic header.
func IsCRX(data []byte) bool {
	return len(data) >= 4 && bytes.Equal(data[:4], crxMagic)
}

// ExtractZipFromCRX strips the CRX2/CRX3 header and returns the embedded zip
// archive. The Chrome Web Store upload endpoint only accepts raw zips, so CRX
// files must be unwrapped before uploading.
func ExtractZipFromCRX(data []byte) ([]byte, error) {
	if !IsCRX(data) {
		return nil, fmt.Errorf("not a CRX file (missing Cr24 magic header)")
	}
	if len(data) < 12 {
		return nil, fmt.Errorf("CRX file is truncated")
	}

	version := binary.LittleEndian.Uint32(data[4:8])
	switch version {
	case 2:
		if len(data) < 16 {
			return nil, fmt.Errorf("CRX2 file is truncated")
		}
		pubKeyLen := binary.LittleEndian.Uint32(data[8:12])
		sigLen := binary.LittleEndian.Uint32(data[12:16])
		offset := uint64(16) + uint64(pubKeyLen) + uint64(sigLen)
		if offset > uint64(len(data)) {
			return nil, fmt.Errorf("CRX2 header exceeds file size")
		}
		return data[offset:], nil
	case 3:
		headerLen := binary.LittleEndian.Uint32(data[8:12])
		offset := uint64(12) + uint64(headerLen)
		if offset > uint64(len(data)) {
			return nil, fmt.Errorf("CRX3 header exceeds file size")
		}
		return data[offset:], nil
	default:
		return nil, fmt.Errorf("unsupported CRX version %d", version)
	}
}
