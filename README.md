vector-icon
===========

The vector-icon project consist of an svg processor tool `procsvg` written in Go,
which generates icon packs in a binary format, and a C++ GdiPlusDemo that paints
icon pack icons.

A specification of the binary icon pack format can be found at `spec.md`.

procsvg
-------

Procsvg processes svg files using Inkspace (object/stroke to path conversion)
and generates binary icon packs.

Inkscape 1.2 from December 2012 or later is required for procsvg to work.

If the `inkspace` is not in `PATH`, the environment variable
`PROCSVG_INKSCAPE` needs to be set.

Details of icon packs such as location of source icons and
the target icon pack file can be specified in TOML project files.

For details of the project file format see `procsvg/project.go`.

GdiPlusDemo
-----------

The demo project contains two main modules:

* `IconPack.cpp` contains classes that handles icon pack loading and
  icon painting using a DrawEngine.
* `GdiPlusIcon.cpp` GDI+ DrawEngine to draw icon pack icons
  in a HDC (GDI device context).
