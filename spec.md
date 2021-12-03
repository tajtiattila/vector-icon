
Icon pack specification
=======================

All numbers are intepreted as little endian,
colors are non-alpha-premultiplied.

Icon pack format
================

1x Icon pack header
Nx Packed icon images (N = NumIcons in icon pack header)

Icon pack header
----------------

Magic [4 x BYTE] "icpk"
NumIcons [UINT32] number of icons in pack

Packed icon image
-----------------

1x Icon header
Nx Icon variant headers (N = NumImage in Icon header)
Nx Icon variant images  (N = NumImage in Icon header)

Variant images are ordered in decreasing size (area) in the icon headers.

Icon header
-----------

FileSize [UINT32] icon size in bytes
NameLen  [BYTE]   icon name length
Name              icon name (UTF-8)
NumImage [BYTE]   number of image variants for this icon

Icon variant header
-------------------

Width    [UINT16] suggested icon width in pixels
Height   [UINT16] suggested icon height in pixels
Offset   [UINT32] image data offset from end of icon variant headers

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
0x02..0x6f Reserved
0x70       MoveTo <x> <y> - Move to position
0x71..0x7f Reserved

0x80..0x9f LineTo <repct> (repct × <x> <y>)
0xa0..0xaf CubicBezierTo <repct> (repct × <x1> <y1> <x2> <y2> <x3> <y3>)
0xb0..0xbf QuadraticBezierTo <repct> (repct × <x1> <y1> <x2> <y2>)
0xc0..0xff Reserved

Colors
------

Colors are 32-bit alpha-premultiplied values in RGBA form.

Coordinate numbers
------------------

This section was taken from iconvg specification at https://github.com/google/iconvg/blob/main/spec/iconvg-spec.md

The encoding of a coordinate number resembles the encoding of a real number. For 1 and 2 byte encodings, the decoded coordinate number equals ((R * scale) - bias), where R is the decoded real number as above. The scale and bias depends on the number of bytes in the encoding.

For a 1 byte encoding, the scale is 1 and the bias is 64, so that a 1 byte coordinate ranges in [-64, +64) at integer granularity. For example, the coordinate 7 can be encoded as 0x8E.

For a 2 byte encoding, the scale is 1/64 and the bias is 128, so that a 2 byte coordinate ranges in [-128, +128) at 1/64 granularity. For example, the coordinate 7.5 can be encoded as 0x81 0x87.

For a 4 byte encoding, the decoded coordinate number simply equals R. For example, the coordinate 7.5 can also be encoded as 0x03 0x00 0xF0 0x40.
