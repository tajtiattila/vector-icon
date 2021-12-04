
#include <cstdint>
#include <istream>
#include <vector>
#include <string>
#include <unordered_map>

namespace vectoricon {

struct RawImage {
	uint16_t dx, dy;
	uint32_t offset; // icon data offset
	uint32_t size; // icon data size
};

struct Icon {
	std::string name;
	std::vector<RawImage> images;
	std::vector<uint8_t> data;
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

class DrawEngine {
public:
	virtual ~DrawEngine() { }

	// ViewBox sets up painting for the icon.
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

bool DrawIcon(Icon const& icon, uint16_t dx, uint16_t dy, DrawEngine* g);

} // end namespace vectoricon
