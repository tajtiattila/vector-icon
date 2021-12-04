
#include "IconPack.h"

#include <optional>

namespace vectoricon {

namespace detail {

uint32_t readUint32(std::istream& strm) {
	uint8_t b[4];
	strm.read((char*)b, 4);

	return (uint32_t(b[3]) << 24) | (uint32_t(b[2]) << 16) |
		(uint32_t(b[1]) << 8) | uint32_t(b[0]);
}

uint16_t readUint16(std::istream& strm) {
	uint8_t b[2];
	strm.read((char*)b, 2);

	return (uint16_t(b[1]) << 8) | uint16_t(b[0]);
}

std::optional<Icon> loadIcon(std::istream& strm) {
	size_t fileSize = readUint32(strm);

	if (fileSize > 1<<20) { // 1M
		return std::nullopt;
	}

	uint8_t nameLen = (uint8_t)strm.get();
	if (!strm.good() || nameLen == 0) {
		return std::nullopt;
	}

	char nameBuf[256];
	strm.read(nameBuf, nameLen);
	if (!strm.good()) {
		return std::nullopt;
	}

	Icon icon;
	icon.name = std::string(nameBuf, size_t(nameLen));

	uint8_t numImages = (uint8_t)strm.get();
	if (!strm.good() || numImages == 0) {
		return std::nullopt;
	}

	icon.images.reserve(numImages);
	uint32_t ofs = 0;
	for (uint8_t i = 0; i < numImages; i++) {
		RawImage ri;
		ri.dx = readUint16(strm);
		ri.dy = readUint16(strm);
		ri.offset = ofs;
		icon.images.push_back(ri);

		ofs += readUint32(strm);
	}

	if (!strm.good()) {
		return std::nullopt;
	}

	size_t headerLen = 1 + size_t(nameLen) + 1 + 8*numImages;
	size_t dataBytes = fileSize - headerLen;

	icon.data.resize(dataBytes);
	strm.read((char*)icon.data.data(), dataBytes);

	if (!strm.good()) {
		return std::nullopt;
	}

	return icon;
}

} // end namespace detail

bool Pack::load(std::istream& strm) {
	char buf[4];
	strm.read(buf, 4);

	if (memcmp(buf, "icpk", 4) != 0) {
		return false;
	}

	uint32_t nicons = detail::readUint32(strm);

	for (uint32_t i = 0; i < nicons; i++) {
		std::optional<Icon> x = detail::loadIcon(strm);
		if (!x.has_value() || !strm.good()) {
			return false;
		}

		size_t idx = icons_.size();
		icons_.push_back(std::move(*x));
		nameToIndex_.insert({x->name, idx});
	}

	return true;
}

const Icon* Pack::find(std::string const& name) const {
	auto it = nameToIndex_.find(name);
	if (it == nameToIndex_.end()) {
		return nullptr;
	}

	size_t i = it->second;
	if (i > icons_.size()) {
		return nullptr;
	}

	return &icons_[i];
}

namespace detail {

float floatFromBits(uint32_t u) {
	float v;
	static_assert(sizeof(u) == sizeof(v), "float and uint32_t sizes differ");
	memcpy(&v, &u, sizeof(u));
	return v;
}

class ProgMem {
public:
	ProgMem(const uint8_t* p, const uint8_t* end) :
		p(p), end(end) {
	}

	bool good() const {
		return p < end;
	}

	uint8_t byte() {
		if (p != end) {
			return *p++;
		}
		return 0;
	}

	float coord() {
		if (!good()) {
			return 0.f;
		}

		uint8_t b0 = byte();
		if ((b0 & 0x01) != 0) {
			return float(int(b0>>1) - 64);
		}

		uint8_t b1 = byte();
		if ((b0 & 0x02) != 0) {
			uint16_t u = uint16_t(b0) | (uint16_t(b1) << 8);
			return float((int(u >> 2)) - 128*64) / 64;
		}

		float flt = 1.0f;
		uint8_t buf[4];
		memcpy(buf, &flt, 4);

		uint8_t b2 = byte();
		uint8_t b3 = byte();
		uint32_t v =
			(uint32_t(b0)) |
			(uint32_t(b1)<<8) |
			(uint32_t(b2)<<16) |
			(uint32_t(b3)<<24);
		return floatFromBits(v);
	}

	Point point() {
		return Point{coord(), coord()};
	}

	void points(std::vector<Point>& dest, size_t n) {
		dest.clear();
		dest.reserve(n);
		for (size_t i = 0; i < n; i++) {
			dest.push_back(point());
		}
	}

private:
	const uint8_t* p;
	const uint8_t* end;
};

bool drawIcon(Icon const& icon, uint32_t ofs, DrawEngine* eng) {
	if (ofs > icon.data.size()) {
		return true;
	}

	const uint8_t* base = icon.data.data();
	ProgMem pm(base+ofs, base+icon.data.size());

	float xmin = pm.coord();
	float ymin = pm.coord();
	float xmax = pm.coord();
	float ymax = pm.coord();
	if (!pm.good()) {
		return false;
	}

	eng->ViewBox(xmin, ymin, xmax, ymax);

	bool hasPath = false;
	auto z = [eng, &hasPath]() {
		if (hasPath) {
			eng->ClosePath();
			hasPath = false;
		}
	};

	std::vector<Point> ptbuf;
	while (pm.good()) {
		uint8_t op = pm.byte();
		switch (op & 0xf0) {
		case 0x00:
			z();
			switch (op) {

			case 0x00:
				// Stop
				return true;

			case 0x01: {
				// Set solid fill
				uint8_t r = pm.byte();
				uint8_t g = pm.byte();
				uint8_t b = pm.byte();
				uint8_t a = pm.byte();
				eng->SetSolidFill(r, g, b, a);
				break;
			}

			default:
				return false;
			}
			break;

		case 0x70:
			z();
			if (op == 0x70) {
				// MoveTo
				eng->MoveTo(pm.point());
			} else {
				return false;
			}
			break;

		case 0x80, 0x90: {
			// LineTo
			size_t rep = 1 + size_t(op - 0x80);
			pm.points(ptbuf, rep);
			eng->LineTo(ptbuf);
			hasPath = true;
			break;
		}

		case 0xa0: {
			// CubicBézierTo
			size_t rep = 1 + size_t(op - 0xa0);
			pm.points(ptbuf, rep*3);
			eng->CubicBezierTo(ptbuf);
			hasPath = true;
			break;
		}

		case 0xb0: {
			// QuadraticBézierTo
			size_t rep = 1 + size_t(op - 0xb0);
			pm.points(ptbuf, rep*2);
			eng->QuadraticBezierTo(ptbuf);
			hasPath = true;
			break;
		}

		default:
			return false;
		}
	}

	return false;
}

} // end namespace detail

bool DrawIcon(Icon const& icon, uint16_t dx, uint16_t dy, DrawEngine* eng) {
	if (icon.images.empty()) {
		return false;
	}

	for (auto& m : icon.images) {
		if (m.dx <= dx && m.dy <= dy) {
			return detail::drawIcon(icon, m.offset, eng);
		}
	}

	return detail::drawIcon(icon, icon.images.back().offset, eng);
}

} // end namespace vectoricon
