package engine

import (
	"image"
	"image/draw"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// Text 用内置位图字体在画布上绘制文本。
// x, y 是文本左上角；scale >= 1 时按整数倍放大（像素风放大）。
func (c *Canvas) Text(x, y int, text string, col Color, scale int) {
	if scale < 1 {
		scale = 1
	}
	face := basicfont.Face7x13
	w := len(text) * 7
	h := 13
	// 渲染到临时 RGBA 图
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(colorToRGBA(col)),
		Face: face,
		Dot:  fixed.P(0, 12),
	}
	d.DrawString(text)

	// 拷贝到画布（放大）
	for sy := 0; sy < h; sy++ {
		for sx := 0; sx < w; sx++ {
			r, g, b, a := img.At(sx, sy).RGBA()
			if a == 0 {
				continue
			}
			_ = r
			_ = g
			_ = b
			if scale == 1 {
				c.SetPixel(x+sx, y+sy, col)
			} else {
				c.FillRect(x+sx*scale, y+sy*scale, scale, scale, col)
			}
		}
	}
}

// TextCentered 绘制水平居中的文本。
func (c *Canvas) TextCentered(x, y int, text string, col Color, scale int) {
	w := len(text) * 7 * scale
	c.Text(x-w/2, y, text, col, scale)
}

// TextRight 绘制右对齐文本（right 是文本右边缘 x）。
func (c *Canvas) TextRight(right, y int, text string, col Color, scale int) {
	w := len(text) * 7 * scale
	c.Text(right-w, y, text, col, scale)
}

// textSize 返回文本像素尺寸（7x13 基准）。
func textSize(text string, scale int) (int, int) {
	if scale < 1 {
		scale = 1
	}
	return len(text) * 7 * scale, 13 * scale
}

var _ = draw.Draw // 保留 image/draw 引用（后续精灵缩放可能用到）
