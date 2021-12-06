
Icon pack specification
=======================

All numbers are intepreted as little endian,
colors are non-alpha-premultiplied.

Icon pack format
================

4xBYTE Magic 'icpk'
UINT32 Number of icons
Nx File segments

File segments begin with a 4 byte identifier, and an UINT32 byte size.
The size doesn't include the identifier and the size itself.

Nx Packed icon images (N = NumIcons in icon pack header)

Palette segment
===============

4×BYTE   Magic 'PALT'
UINT32   Palette data size in bytes
UINT8    Palette index
UINT8    Number of colors (N)
N×4×BYTE Palette color entries

Palette color entries are 4 byte non-premultiplied RGBA.

Packed icon image segment
=========================

1x Icon header
Nx Icon variant headers (N = NumImage in Icon header)
Nx Icon variant image data (N = NumImage in Icon header)

Variant images are ordered in decreasing size (area) in the icon headers.

Icon header
-----------

4xBYTE  Magic 'ICON'
UINT32  Icon data size in bytes
BYTE    Icon name length
        Icon name (UTF-8)
BYTE    Number of image variants for this icon

Icon variant header
-------------------

Width    [UINT16] suggested icon width in pixels
Height   [UINT16] suggested icon height in pixels
ByteSize [UINT32] size of image data in bytes

Icon variant image
------------------

ViewBox  [4 x COORD] icon view box (left, top, right, bottom)
Program              icon program

Program opcodes
===============

Commands below 0x70 paints open paths with the
currently selected style (no stroke with solid fill for now).

0x00       Stop - End program
0x01       SetSolidFill <color> - Set solid fill color
0x02       SetSolidFill <palette-index> - Set solid fill color
0x03..0x6f Reserved
0x70       BeginMoveTo <x> <y> - Begin a new path at position
0x71       MoveTo <x> <y> - Move to position
0x72..0x7f Reserved

0x80..0x9f LineTo <repct> (repct × <x> <y>)
0xa0..0xaf CubicBezierTo <repct> (repct × <x1> <y1> <x2> <y2> <x3> <y3>)
0xb0..0xbf QuadraticBezierTo <repct> (repct × <x1> <y1> <x2> <y2>)
0xc0..0xff Reserved

Colors
------

Colors are 32-bit alpha-premultiplied values in RGBA form.

Coordinate numbers
------------------

The two lowest order bits specify the encoding:
- If bit 0 is set, the encoded value is shifted 1 bit and uses 1 byte encoding.
- If bit 0 is clear and bit 1 is set, the encoded value is shifted 2 bits and uses 2 byte encoding.
- If both bit 0 and 1 are clear, the value uses 4 byte encoding.

The encoding of a coordinate number resembles the encoding of a real number. For 1 and 2 byte encodings, the decoded coordinate number equals ((R * scale) - bias), where R is the decoded real number as above. The scale and bias depends on the number of bytes in the encoding.

For a 1 byte encoding, the scale is 1 and the bias is 64, so that a 1 byte coordinate ranges in [-64, +64) at integer granularity. For example, the coordinate 7 can be encoded as 0x8F.

For a 2 byte encoding, the scale is 1/64 and the bias is 128, so that a 2 byte coordinate ranges in [-128, +128) at 1/64 granularity. For example, the coordinate 7.5 can be encoded as 0x82 0x87.

For a 4 byte encoding, the decoded coordinate number simply equals R. For example, the coordinate 7.5 can also be encoded as 0x00 0x00 0xF0 0x40.
