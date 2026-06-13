package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

func noitaStreamInfoPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home,
		".steam/steam/steamapps/compatdata/881100/pfx/drive_c/users/steamuser",
		"AppData/LocalLow/Nolla_Games_Noita/save00/world/.stream_info")
}

// readNoitaSeed reads the world seed from a Noita .stream_info file.
// File layout: [le u32 compressed_size][le u32 decompressed_size][fastlz data]
// Decompressed stream_info starts with: [be u32 version=24][be u32 world_seed]
func readNoitaSeed(path string) (uint32, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if len(raw) < 12 {
		return 0, fmt.Errorf("file too short (%d bytes)", len(raw))
	}
	compSize := int(binary.LittleEndian.Uint32(raw[:4]))
	payload := raw[8:]
	if len(payload) < compSize {
		return 0, fmt.Errorf("truncated: need %d compressed bytes, have %d", compSize, len(payload))
	}
	dec, err := fastlzDecompress(payload[:compSize])
	if err != nil {
		return 0, fmt.Errorf("decompress: %w", err)
	}
	if len(dec) < 8 {
		return 0, fmt.Errorf("decompressed data too short (%d bytes)", len(dec))
	}
	return binary.BigEndian.Uint32(dec[4:8]), nil
}

// fastlzDecompress decompresses a FastLZ byte slice (level 1 or 2).
// Control byte encoding:
//   ctrl < 32: literal run of (ctrl+1) bytes
//   ctrl >= 32: back-reference; length = (ctrl>>5 - 1) [+ext if ctrl>>5==7] + 3
//               offset = ((ctrl&0x1f)<<8 | next_byte) + 1
func fastlzDecompress(src []byte) ([]byte, error) {
	out := make([]byte, 0, len(src)*4)
	ip := 0

	if ip >= len(src) {
		return out, nil
	}
	ctrl := int(src[ip]) & 0x1f // first ctrl uses only lower 5 bits
	ip++

	for {
		if ctrl < 32 {
			n := ctrl + 1
			if ip+n > len(src) {
				return nil, fmt.Errorf("literal overrun at ip=%d n=%d", ip, n)
			}
			out = append(out, src[ip:ip+n]...)
			ip += n
		} else {
			lenCode := (ctrl >> 5) - 1
			ofsHigh := (ctrl & 0x1f) << 8
			if lenCode == 6 {
				if ip >= len(src) {
					return nil, fmt.Errorf("missing length extension at ip=%d", ip)
				}
				lenCode += int(src[ip])
				ip++
			}
			if ip >= len(src) {
				return nil, fmt.Errorf("missing offset byte at ip=%d", ip)
			}
			offset := (ofsHigh | int(src[ip])) + 1
			ip++
			ref := len(out) - offset
			if ref < 0 {
				return nil, fmt.Errorf("back-ref underflow: len=%d offset=%d", len(out), offset)
			}
			length := lenCode + 3
			for i := 0; i < length; i++ {
				out = append(out, out[ref+i])
			}
		}

		if ip >= len(src)-1 {
			break
		}
		ctrl = int(src[ip])
		ip++
	}

	return out, nil
}
