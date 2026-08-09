package engine

import "image"

// Sprite 是像素画精灵：RGBA 像素数组 + 宽高。
// 精灵数据可以是程序生成的（像素画/数学纹理），也可以从 PNG 解码后填充。
// 布局: B,G,R,A（A=0 透明）。
type Sprite struct {
	W, H   int
	Pixels []byte
}

// NewSprite 创建空白精灵（全透明）。
func NewSprite(w, h int) *Sprite {
	return &Sprite{
		W:      w,
		H:      h,
		Pixels: make([]byte, w*h*4),
	}
}

// Set 设置精灵内 (x,y) 像素颜色，a=0 透明、255 不透明。
func (s *Sprite) Set(x, y int, col Color, a byte) {
	if x < 0 || y < 0 || x >= s.W || y >= s.H {
		return
	}
	i := (y*s.W + x) * 4
	s.Pixels[i] = col.B
	s.Pixels[i+1] = col.G
	s.Pixels[i+2] = col.R
	s.Pixels[i+3] = a
}

// Draw 把精灵画到画布 (x,y)。
func (s *Sprite) Draw(c *Canvas, x, y int) {
	c.Blit(x, y, s.Pixels, s.W, s.H)
}

// DrawScaled 按整数倍放大绘制（最近邻，像素风）。
func (s *Sprite) DrawScaled(c *Canvas, x, y, scale int) {
	if scale <= 0 {
		scale = 1
	}
	if scale == 1 {
		s.Draw(c, x, y)
		return
	}
	for sy := 0; sy < s.H; sy++ {
		for sx := 0; sx < s.W; sx++ {
			i := (sy*s.W + sx) * 4
			if s.Pixels[i+3] == 0 {
				continue
			}
			c.FillRect(x+sx*scale, y+sy*scale, scale, scale,
				Color{R: s.Pixels[i+2], G: s.Pixels[i+1], B: s.Pixels[i]})
		}
	}
}

// Fill 用颜色填充精灵全部像素。
func (s *Sprite) Fill(col Color) {
	for i := 0; i < len(s.Pixels); i += 4 {
		s.Pixels[i] = col.B
		s.Pixels[i+1] = col.G
		s.Pixels[i+2] = col.R
		s.Pixels[i+3] = 255
	}
}

// MakePixelArt 从字符串数组创建像素画精灵（ASCII 到颜色的映射）。
// 每行字符串长度必须一致；' ' 或 '.' 视为透明。
// 用法：
//
//	spr := engine.MakePixelArt([]string{
//		".RR.",
//		"RYYR",
//		"RBBR",
//		".RR.",
//	}, map[byte]engine.Color{
//		'R': engine.ColRed,
//		'Y': engine.ColYellow,
//		'B': engine.ColBlue,
//	})
func MakePixelArt(rows []string, palette map[byte]Color) *Sprite {
	if len(rows) == 0 {
		return NewSprite(0, 0)
	}
	w := 0
	for _, r := range rows {
		if len(r) > w {
			w = len(r)
		}
	}
	s := NewSprite(w, len(rows))
	for y, row := range rows {
		for x := 0; x < len(row); x++ {
			ch := row[x]
			if ch == ' ' || ch == '.' {
				continue // 透明
			}
			if col, ok := palette[ch]; ok {
				s.Set(x, y, col, 255)
			}
		}
	}
	return s
}

// FromImage 从 image.Image 创建精灵（RGBA，A=0 透明）。
func FromImage(img image.Image) *Sprite {
	b := img.Bounds()
	s := NewSprite(b.Dx(), b.Dy())
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			r, g, bb, a := img.At(x, y).RGBA()
			i := (y*s.W + x) * 4
			s.Pixels[i] = byte(bb >> 8)
			s.Pixels[i+1] = byte(g >> 8)
			s.Pixels[i+2] = byte(r >> 8)
			s.Pixels[i+3] = byte(a >> 8)
		}
	}
	return s
}

// Resize 最近邻缩放精灵。
func (s *Sprite) Resize(nw, nh int) *Sprite {
	out := NewSprite(nw, nh)
	for y := 0; y < nh; y++ {
		for x := 0; x < nw; x++ {
			sx := x * s.W / nw
			sy := y * s.H / nh
			i := (sy*s.W + sx) * 4
			o := (y*nw + x) * 4
			out.Pixels[o] = s.Pixels[i]
			out.Pixels[o+1] = s.Pixels[i+1]
			out.Pixels[o+2] = s.Pixels[i+2]
			out.Pixels[o+3] = s.Pixels[i+3]
		}
	}
	return out
}

// DrawResized 按任意尺寸绘制（最近邻采样）。
func (s *Sprite) DrawResized(c *Canvas, x, y, w, h int) {
	if w <= 0 || h <= 0 || s.W == 0 || s.H == 0 {
		return
	}
	for dy := 0; dy < h; dy++ {
		sy := dy * s.H / h
		for dx := 0; dx < w; dx++ {
			sx := dx * s.W / w
			i := (sy*s.W + sx) * 4
			if s.Pixels[i+3] == 0 {
				continue
			}
			c.SetPixel(x+dx, y+dy, Color{R: s.Pixels[i+2], G: s.Pixels[i+1], B: s.Pixels[i]})
		}
	}
}
