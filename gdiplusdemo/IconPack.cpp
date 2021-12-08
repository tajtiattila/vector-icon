
#include "IconPack.h"

#include <optional>
#include <sstream>

namespace vectoricon {

namespace detail {

uint32_t readUint32(std::istream& strm) {
	uint8_t b[4];
	strm.read((char*)b, 4);

	if (!strm.good() || strm.gcount() != 4) {
		strm.setstate(strm.failbit);
	}

	return (uint32_t(b[3]) << 24) | (uint32_t(b[2]) << 16) |
		(uint32_t(b[1]) << 8) | uint32_t(b[0]);
}

uint16_t readUint16(std::istream& strm) {
	uint8_t b[2];
	strm.read((char*)b, 2);

	return (uint16_t(b[1]) << 8) | uint16_t(b[0]);
}

struct SectionHeader {
	char magic[4];
	uint32_t size;
};

std::optional<SectionHeader> readSectionHeader(std::istream& strm) {
	SectionHeader sh;
	strm.read(sh.magic, 4);
	if (strm.eof() && strm.gcount() == 0) {
		strm.clear(strm.goodbit | strm.eofbit);
		return std::nullopt;
	}

	if (strm.gcount() != 4) {
		strm.setstate(strm.failbit);
		return std::nullopt;
	}

	sh.size = readUint32(strm);
	if (!strm.good()) {
		return std::nullopt;
	}

	return sh;
}

bool loadPalette(std::istream& strm, size_t sectSize, 
		std::shared_ptr<PaletteVector>& /*in-out*/ pv) {
	uint8_t buf[4];
	strm.read((char*)&buf, 2);
	if (!strm.good()) {
		return false;
	}

	size_t idx = buf[0];
	size_t count = buf[1];
	if (2 + 4*count != sectSize) {
		// invalid palette size
		return false;
	}

	if (count == 0) {
		// empty palette
		return true;
	}

	if (pv == nullptr) {
		pv = std::make_shared<PaletteVector>();
	}

	auto& v = *pv;
	if (v.size() <= idx) {
		v.resize(idx+1);
	}

	auto& palette = v[idx];
	palette.resize(count);
	for (auto &c : palette) {
		strm.read((char*)&buf, 4);
		if (!strm.good()) {
			return false;
		}

		c.r = buf[0];
		c.g = buf[1];
		c.b = buf[2];
		c.a = buf[3];
	}

	return true;
}

std::optional<Icon> loadIcon(std::istream& strm, size_t sectSize,
		std::shared_ptr<PaletteVector> const& pv) {
	uint8_t nameLen = (uint8_t)strm.get();
	if (!strm.good() || nameLen == 0) {
		return std::nullopt;
	}

	char nameBuf[256];
	strm.read(nameBuf, nameLen);
	if (!strm.good()) {
		return std::nullopt;
	}

	auto icondata = std::make_shared<IconData>();
	auto& icon = *icondata;

	icon.name = std::string(nameBuf, size_t(nameLen));
	icon.palvec = pv;

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
		ri.size = readUint32(strm);

		ri.offset = ofs;

		icon.images.push_back(ri);

		ofs += ri.size;
	}

	if (!strm.good()) {
		return std::nullopt;
	}

	size_t headerLen = 1 + size_t(nameLen) + 1 + 8*numImages;
	size_t dataBytes = sectSize - headerLen;

	icon.data.resize(dataBytes);
	strm.read((char*)icon.data.data(), dataBytes);

	if (!strm.good()) {
		return std::nullopt;
	}

	return Icon(icondata);
}

} // end namespace detail

bool Pack::load(std::istream& strm) {
	char buf[4];
	strm.read(buf, 4);

	if (memcmp(buf, "icpk", 4) != 0) {
		return false;
	}

	uint32_t nicons = detail::readUint32(strm);
	icons_.reserve(icons_.size() + nicons);

	std::shared_ptr<PaletteVector> pv;
	while (!strm.eof()) {
		auto oh = detail::readSectionHeader(strm);
		if (!oh.has_value()) {
			return strm.eof() && !strm.fail();
		}

		auto& h = *oh;

		if (memcmp(h.magic, "PALT", 4) == 0) {
			if (!detail::loadPalette(strm, h.size, pv)) {
				return false;
			}
		} else if (memcmp(h.magic, "ICON", 4) == 0) {
			std::optional<Icon> x = detail::loadIcon(strm, h.size, pv);
			if (!x.has_value() || !strm.good()) {
				return false;
			}

			size_t idx = icons_.size();
			icons_.push_back(std::move(*x));
			nameToIndex_.insert({icons_.back().Name(), idx});
		} else {
			// skip unknown section
			strm.seekg(h.size, strm.cur);
		}
	}

	return true;
}

Icon Pack::at(size_t i) const {
  if (i >= icons_.size()) {
    return nullptr;
  }

  return icons_[i];
}

Icon Pack::find(std::string const& name) const {
	auto it = nameToIndex_.find(name);
	if (it == nameToIndex_.end()) {
		return nullptr;
	}

	size_t i = it->second;
	if (i > icons_.size()) {
		return nullptr;
	}

	return icons_[i];
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
		start(p), p(p), end(end) {
	}

	size_t pos() const {
		return p - start;
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
	const uint8_t* start;
	const uint8_t* p;
	const uint8_t* end;
};

std::optional<RGBA> paletteColor(IconData const& icon,
		size_t paletteIndex, size_t colorIndex) {
	if (icon.palvec == nullptr) {
		return std::nullopt;
	}

	auto& v = *icon.palvec;
	if (paletteIndex >= v.size()) {
		return std::nullopt;
	}

	auto& w = v[paletteIndex];
	if (colorIndex >= w.size()) {
		return std::nullopt;
	}

	return w[colorIndex];
}

void drawImage(IconData const& icon, size_t paletteIndex,
	uint32_t ofs, uint32_t sz, DrawEngine* eng) {

	if (ofs+sz > icon.data.size()) {
		eng->Error(error::EmptyImage{});
		return;
	}

	const uint8_t* base = icon.data.data();
	ProgMem pm(base+ofs, base+ofs+sz);

	float xmin = pm.coord();
	float ymin = pm.coord();
	float xmax = pm.coord();
	float ymax = pm.coord();
	if (!pm.good()) {
		eng->Error(error::EmptyImage{});
		return;
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
		size_t opPos = pm.pos();
		uint8_t op = pm.byte();
		switch (op & 0xf0) {
		case 0x00:
			z();
			switch (op) {

			case 0x00:
				// Stop
				return;

			case 0x01: {
				// Set solid RGBA fill
				uint8_t r = pm.byte();
				uint8_t g = pm.byte();
				uint8_t b = pm.byte();
				uint8_t a = pm.byte();
				eng->SetSolidFill(r, g, b, a);
				break;
			}

			case 0x02: {
				// Set solid palette fill
				size_t i = pm.byte();
				auto oc = paletteColor(icon, paletteIndex, i);
				if (!oc) {
					eng->Error(error::InvalidPaletteIndex{opPos, i});
					return;
				}
				auto c = *oc;
				eng->SetSolidFill(c.r, c.g, c.b, c.a);
				break;
			}

			default:
				eng->Error(error::InvalidOpCode{opPos, op});
				return;
			}
			break;

		case 0x70:
			if (op == 0x70 || op == 0x71) {
				// MoveTo
				if (op == 0x70) {
					z();
				}
				eng->MoveTo(pm.point());
			} else {
				eng->Error(error::InvalidOpCode{opPos, op});
				return;
			}
			break;

		case 0x80:
		case 0x90: {
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
			eng->Error(error::InvalidOpCode{opPos, op});
			return;
		}
	}
}

} // end namespace detail

namespace error {

std::string EmptyImage::Msg() const {
	return "empty image";
}

std::string InvalidPaletteIndex::Msg() const {
	std::ostringstream s;
	s << "invalid palette index " << i << " at byte " << p;
	return s.str();
}

std::string InvalidOpCode::Msg() const {
	std::ostringstream s;
	s << "invalid opcode " << std::hex << std::showbase << int(op)
		<< " at byte " << std::dec << std::noshowbase << p;
	return s.str();
}

} // end namespace error

void Icon::Draw(DrawEngine* eng, uint16_t dx, uint16_t dy, size_t paletteIndex) const {
	if (d == nullptr) {
		// empty icon
		eng->Error(error::EmptyImage{});
		return;
	}

	IconData const& icon = *d;
	if (icon.images.empty()) {
		eng->Error(error::EmptyImage{});
		return;
	}

	for (auto& m : icon.images) {
		if (m.dx <= dx && m.dy <= dy) {
			return detail::drawImage(icon, paletteIndex, m.offset, m.size, eng);
		}
	}

	auto const& m = icon.images.back();
	return detail::drawImage(icon, paletteIndex, m.offset, m.size, eng);
}

} // end namespace vectoricon
