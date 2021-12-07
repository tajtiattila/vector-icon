
#include <cstdint>
#include <istream>
#include <vector>
#include <string>
#include <unordered_map>

namespace vectoricon {

class Icon;
class Pack;
class DrawEngine;

// RGBA is a non-alpha-premultiplied color.
struct RGBA {
	uint8_t r, g, b, a;
};

using Palette = std::vector<RGBA>;
using PaletteVector = std::vector<Palette>;

struct RawImage {
	uint16_t dx, dy;
	uint32_t offset; // icon data offset
	uint32_t size; // icon data size
};

struct IconData {
	std::string name;
	std::vector<RawImage> images;
	std::vector<uint8_t> data;
	std::shared_ptr<PaletteVector> palvec;
};

class Icon {
public:
	Icon(std::shared_ptr<IconData> d0) : d(d0) { }

	std::string Name() const { return d != nullptr ? d->name : ""; }

	void Draw(DrawEngine* eng, uint16_t dx, uint16_t dy, size_t paletteIndex = 0) const;

private:
	std::shared_ptr<IconData> d;
};

class Pack {
public:
	bool load(std::istream& strm);

	auto begin() const { return icons_.begin(); }
	auto end()   const { return icons_.end(); }

	const Icon* find(std::string const& name) const;

	size_t size() const { return icons_.size(); }

private:
	std::vector<Icon> icons_;
	std::unordered_map<std::string, size_t> nameToIndex_;
};

struct Point {
	float x, y;
};

// DrawError represents an error drawing icons.
class DrawError {
public:
	virtual ~DrawError() { }

	// Pos reports the byte position of the error.
	virtual size_t Pos() const { return 0; }

	virtual std::string Msg() const = 0;
};

// DrawEngine paints icons.
class DrawEngine {
public:
	virtual ~DrawEngine() { }

	virtual void Error(DrawError const&) { }

	// ViewBox sets up the view box for painting the icon.
	virtual void ViewBox(float xmin, float ymin, float xmax, float ymax) = 0;

	// SetSolidFill sets up solid fill mode.
	virtual void SetSolidFill(uint8_t r, uint8_t g, uint8_t b, uint8_t a) = 0;

	virtual void MoveTo(Point p) = 0;
	virtual void LineTo(std::vector<Point> const& p) = 0;
	virtual void CubicBezierTo(std::vector<Point> const& p) = 0;
	virtual void QuadraticBezierTo(std::vector<Point> const& p) = 0;

	// ClosePath paints the path with the current fill style.
	virtual void ClosePath() = 0;
};

namespace error {

class EmptyImage : public DrawError {
public:
	std::string Msg() const override;
};

class InvalidPaletteIndex : public DrawError {
public:
	InvalidPaletteIndex(size_t p, size_t i) :
		p(p), i(i) { }

	size_t Pos() const override { return p; }
	std::string Msg() const override;

	size_t Index() const { return i; }
private:
	size_t p, i;
};

class InvalidOpCode : public DrawError {
public:
	InvalidOpCode(size_t p, uint8_t op) :
		p(p), op(op) { }

	size_t Pos() const override { return p; }
	std::string Msg() const override;

	size_t OpCode() const { return op; }
private:
	size_t p;
	uint8_t op;
};

} // end namespace

} // end namespace vectoricon
