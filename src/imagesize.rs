//! Intrinsic pixel dimensions from common image headers.
//!
//! Only the file header is read, so callers can avoid decoding the whole
//! image. Returns `None` for unsupported or malformed input.

/// Return the `(width, height)` in pixels for a PNG, JPEG, or WebP image.
pub fn dimensions(bytes: &[u8]) -> Option<(u32, u32)> {
    png(bytes).or_else(|| jpeg(bytes)).or_else(|| webp(bytes))
}

fn be16(bytes: &[u8], at: usize) -> Option<u16> {
    let raw: [u8; 2] = bytes.get(at..at + 2)?.try_into().ok()?;
    Some(u16::from_be_bytes(raw))
}

fn le16(bytes: &[u8], at: usize) -> Option<u16> {
    let raw: [u8; 2] = bytes.get(at..at + 2)?.try_into().ok()?;
    Some(u16::from_le_bytes(raw))
}

fn be32(bytes: &[u8], at: usize) -> Option<u32> {
    let raw: [u8; 4] = bytes.get(at..at + 4)?.try_into().ok()?;
    Some(u32::from_be_bytes(raw))
}

fn le24(bytes: &[u8], at: usize) -> Option<u32> {
    let raw = bytes.get(at..at + 3)?;
    Some(u32::from_le_bytes([raw[0], raw[1], raw[2], 0]))
}

fn non_zero(width: u32, height: u32) -> Option<(u32, u32)> {
    (width > 0 && height > 0).then_some((width, height))
}

fn png(bytes: &[u8]) -> Option<(u32, u32)> {
    const SIGNATURE: &[u8] = b"\x89PNG\r\n\x1a\n";
    if !bytes.starts_with(SIGNATURE) || bytes.get(12..16)? != b"IHDR" {
        return None;
    }
    non_zero(be32(bytes, 16)?, be32(bytes, 20)?)
}

fn jpeg(bytes: &[u8]) -> Option<(u32, u32)> {
    if !bytes.starts_with(&[0xff, 0xd8]) {
        return None;
    }
    let mut i = 2;
    while i + 1 < bytes.len() {
        if bytes[i] != 0xff {
            i += 1;
            continue;
        }
        let marker = bytes[i + 1];
        if marker == 0xff {
            i += 1;
            continue;
        }
        // Standalone markers carry no length payload.
        if marker == 0x01 || (0xd0..=0xd9).contains(&marker) {
            i += 2;
            continue;
        }
        // The start of scan means the frame header already passed.
        if marker == 0xda {
            return None;
        }
        let length = be16(bytes, i + 2)? as usize;
        if length < 2 {
            return None;
        }
        let is_sof = matches!(
            marker,
            0xc0..=0xc3 | 0xc5..=0xc7 | 0xc9..=0xcb | 0xcd..=0xcf
        );
        if is_sof {
            return non_zero(be16(bytes, i + 7)? as u32, be16(bytes, i + 5)? as u32);
        }
        i += 2 + length;
    }
    None
}

fn webp(bytes: &[u8]) -> Option<(u32, u32)> {
    if bytes.get(0..4)? != b"RIFF" || bytes.get(8..12)? != b"WEBP" {
        return None;
    }
    match bytes.get(12..16)? {
        b"VP8 " => {
            // Lossy: 3-byte frame tag, then the start code 0x9d 0x01 0x2a.
            if bytes.get(23..26)? != [0x9d, 0x01, 0x2a] {
                return None;
            }
            let width = (le16(bytes, 26)? & 0x3fff) as u32;
            let height = (le16(bytes, 28)? & 0x3fff) as u32;
            non_zero(width, height)
        }
        b"VP8L" => {
            // Lossless: signature 0x2f then packed 14-bit (size - 1) values.
            if bytes.get(20) != Some(&0x2f) {
                return None;
            }
            let packed = u32::from_le_bytes(bytes.get(21..25)?.try_into().ok()?);
            non_zero((packed & 0x3fff) + 1, ((packed >> 14) & 0x3fff) + 1)
        }
        b"VP8X" => {
            // Extended: 24-bit (canvas size - 1) values after the feature flags.
            non_zero(le24(bytes, 24)? + 1, le24(bytes, 27)? + 1)
        }
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn png_header(width: u32, height: u32) -> Vec<u8> {
        let mut bytes = b"\x89PNG\r\n\x1a\n".to_vec();
        bytes.extend_from_slice(&[0, 0, 0, 13]);
        bytes.extend_from_slice(b"IHDR");
        bytes.extend_from_slice(&width.to_be_bytes());
        bytes.extend_from_slice(&height.to_be_bytes());
        bytes
    }

    #[test]
    fn reads_png_header() {
        assert_eq!(dimensions(&png_header(902, 372)), Some((902, 372)));
    }

    #[test]
    fn reads_real_png() {
        let path = concat!(
            env!("CARGO_MANIFEST_DIR"),
            "/tests/fixtures/lio2026skola/cuska/teksts/cuska.png"
        );
        let bytes = std::fs::read(path).expect("read cuska.png");
        assert_eq!(dimensions(&bytes), Some((7169, 4290)));
    }

    #[test]
    fn reads_jpeg_sof0() {
        let mut bytes = vec![0xff, 0xd8];
        // APP0 segment: marker, 2-byte length (16), 14 bytes payload.
        bytes.extend_from_slice(&[0xff, 0xe0, 0x00, 0x10]);
        bytes.extend_from_slice(&[0u8; 14]);
        // SOF0: marker, length 17, precision, height, width, components.
        bytes.extend_from_slice(&[0xff, 0xc0, 0x00, 0x11, 0x08]);
        bytes.extend_from_slice(&372u16.to_be_bytes());
        bytes.extend_from_slice(&902u16.to_be_bytes());
        bytes.extend_from_slice(&[0x03, 0x01, 0x11, 0x00]);
        assert_eq!(dimensions(&bytes), Some((902, 372)));
    }

    #[test]
    fn reads_webp_lossy() {
        let mut bytes = b"RIFF\x00\x00\x00\x00WEBPVP8 ".to_vec();
        bytes.extend_from_slice(&[0, 0, 0, 0]); // chunk size
        bytes.extend_from_slice(&[0, 0, 0]); // frame tag
        bytes.extend_from_slice(&[0x9d, 0x01, 0x2a]);
        bytes.extend_from_slice(&902u16.to_le_bytes());
        bytes.extend_from_slice(&372u16.to_le_bytes());
        assert_eq!(dimensions(&bytes), Some((902, 372)));
    }

    #[test]
    fn rejects_unknown_input() {
        assert_eq!(dimensions(b"not an image"), None);
        assert_eq!(dimensions(&[]), None);
    }
}
