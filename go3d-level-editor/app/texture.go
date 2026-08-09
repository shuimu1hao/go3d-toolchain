package app

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"go2dgame/engine"
)

// Sprite 2D 贴图素材（PNG/JPEG），作为公告板精灵渲染在关卡中。
type Sprite struct {
	Name string
	RGBA []byte // RGBA 像素（非预乘）
	W, H int
}

// LoadSprite 从文件加载图片为精灵。
func LoadSprite(path string) (*Sprite, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	pix := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, a := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			oi := (y*w + x) * 4
			pix[oi+0] = byte(r >> 8)
			pix[oi+1] = byte(g >> 8)
			pix[oi+2] = byte(bl >> 8)
			pix[oi+3] = byte(a >> 8)
		}
	}
	name := path
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			name = path[i+1:]
			break
		}
	}
	return &Sprite{Name: name, RGBA: pix, W: w, H: h}, nil
}

// DrawSprite 把精灵绘制到画布（缩放 + 居中）。
func DrawSprite(c *engine.Canvas, s *Sprite, cx, cy, w, h int) {
	if s == nil || w <= 0 || h <= 0 {
		return
	}
	// 最近邻缩放绘制（带 alpha）
	for dy := 0; dy < h; dy++ {
		sy := dy * s.H / h
		if sy >= s.H {
			sy = s.H - 1
		}
		for dx := 0; dx < w; dx++ {
			sx := dx * s.W / w
			if sx >= s.W {
				sx = s.W - 1
			}
			si := (sy*s.W + sx) * 4
			px := cx - w/2 + dx
			py := cy - h/2 + dy
			if px < 0 || px >= c.W || py < 0 || py >= c.H {
				continue
			}
			alpha := s.RGBA[si+3]
			if alpha == 0 {
				continue
			}
			if alpha == 255 {
				c.SetPixel(px, py, engine.Color{R: s.RGBA[si], G: s.RGBA[si+1], B: s.RGBA[si+2]})
				continue
			}
			old := c.GetPixel(px, py)
			a := float32(alpha) / 255
			c.SetPixel(px, py, engine.Color{
				R: byte(float32(s.RGBA[si])*a + float32(old.R)*(1-a)),
				G: byte(float32(s.RGBA[si+1])*a + float32(old.G)*(1-a)),
				B: byte(float32(s.RGBA[si+2])*a + float32(old.B)*(1-a)),
			})
		}
	}
}
