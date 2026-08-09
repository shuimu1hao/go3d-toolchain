package engine

import (
	"image/color"

	"github.com/BurntSushi/xgb/xproto"
)

// 像素格式：跟随 X server 的 depth=24 / bpp=32，内存布局 B,G,R,pad
// （termux-x11 实测 root visual 24bit，PixmapFormat bpp=32，LSB 顺序）。
const (
	pixBpp = 4 // 每像素字节数
)

// Color 是引擎的 RGBA 颜色（A 忽略，软件渲染默认不透明）。
type Color struct {
	R, G, B uint8
}

// 常用颜色
var (
	ColBlack  = Color{0, 0, 0}
	ColWhite  = Color{255, 255, 255}
	ColRed    = Color{255, 60, 60}
	ColGreen  = Color{80, 255, 120}
	ColBlue   = Color{80, 160, 255}
	ColYellow = Color{255, 220, 80}
	ColCyan   = Color{80, 230, 230}
	ColPink   = Color{255, 120, 200}
	ColGray   = Color{120, 120, 120}
)

// Canvas 是软件渲染画布：RGBA 像素缓冲 + 绘图原语 + 分块 PutImage 上屏。
type Canvas struct {
	W, H    int
	Pixels  []byte // 每像素 4 字节 B,G,R,pad，长度 W*H*4
	WindowW int    // 实际窗口尺寸（WM 可能调整，画布逻辑尺寸不变）
	WindowH int
}

// NewCanvas 创建画布。
func NewCanvas(w, h int) *Canvas {
	return &Canvas{
		W:      w,
		H:      h,
		Pixels: make([]byte, w*h*pixBpp),
		WindowW: w,
		WindowH: h,
	}
}

// idx 返回 (x,y) 的像素起始下标（越界返回 -1）。
func (c *Canvas) idx(x, y int) int {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return -1
	}
	return (y*c.W + x) * pixBpp
}

// SetPixel 画单个像素。
func (c *Canvas) SetPixel(x, y int, col Color) {
	i := c.idx(x, y)
	if i < 0 {
		return
	}
	c.Pixels[i] = col.B
	c.Pixels[i+1] = col.G
	c.Pixels[i+2] = col.R
	c.Pixels[i+3] = 0
}

// GetPixel 读取像素颜色。
func (c *Canvas) GetPixel(x, y int) Color {
	i := c.idx(x, y)
	if i < 0 {
		return ColBlack
	}
	return Color{R: c.Pixels[i+2], G: c.Pixels[i+1], B: c.Pixels[i]}
}

// Clear 用颜色填充整个画布。
func (c *Canvas) Clear(col Color) {
	for i := 0; i < len(c.Pixels); i += pixBpp {
		c.Pixels[i] = col.B
		c.Pixels[i+1] = col.G
		c.Pixels[i+2] = col.R
		c.Pixels[i+3] = 0
	}
}

// FillRect 填充矩形（含边界裁剪）。
func (c *Canvas) FillRect(x, y, w, h int, col Color) {
	if w <= 0 || h <= 0 {
		return
	}
	x0, y0 := x, y
	x1, y1 := x+w, y+h
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > c.W {
		x1 = c.W
	}
	if y1 > c.H {
		y1 = c.H
	}
	for yy := y0; yy < y1; yy++ {
		row := yy * c.W * pixBpp
		for xx := x0; xx < x1; xx++ {
			i := row + xx*pixBpp
			c.Pixels[i] = col.B
			c.Pixels[i+1] = col.G
			c.Pixels[i+2] = col.R
			c.Pixels[i+3] = 0
		}
	}
}

// Rect 画矩形边框。
func (c *Canvas) Rect(x, y, w, h int, col Color) {
	c.FillRect(x, y, w, 1, col)
	c.FillRect(x, y+h-1, w, 1, col)
	c.FillRect(x, y, 1, h, col)
	c.FillRect(x+w-1, y, 1, h, col)
}

// Line 画线段（Bresenham）。
func (c *Canvas) Line(x0, y0, x1, y1 int, col Color) {
	dx := x1 - x0
	dy := y1 - y0
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	steps := dx
	if dy > steps {
		steps = dy
	}
	if steps == 0 {
		c.SetPixel(x0, y0, col)
		return
	}
	stepX := float32(x1-x0) / float32(steps)
	stepY := float32(y1-y0) / float32(steps)
	px, py := float32(x0), float32(y0)
	for i := 0; i <= steps; i++ {
		c.SetPixel(int(px+0.5), int(py+0.5), col)
		px += stepX
		py += stepY
	}
}

// Circle 画实心圆（中点圆算法填充）。
func (c *Canvas) Circle(cx, cy, r int, col Color) {
	if r <= 0 {
		return
	}
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r*r {
				c.SetPixel(x, y, col)
			}
		}
	}
}

// CircleOutline 画空心圆。
func (c *Canvas) CircleOutline(cx, cy, r int, col Color) {
	if r <= 0 {
		return
	}
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx := x - cx
			dy := y - cy
			d2 := dx*dx + dy*dy
			if d2 >= (r-1)*(r-1) && d2 <= r*r {
				c.SetPixel(x, y, col)
			}
		}
	}
}

// Blit 把 src（RGBA 像素）画到 (x,y)，支持透明色 key（alpha 为 0 的像素跳过）。
// src 布局: B,G,R,A。
func (c *Canvas) Blit(x, y int, src []byte, sw, sh int) {
	for sy := 0; sy < sh; sy++ {
		dy := y + sy
		if dy < 0 || dy >= c.H {
			continue
		}
		for sx := 0; sx < sw; sx++ {
			dx := x + sx
			if dx < 0 || dx >= c.W {
				continue
			}
			si := (sy*sw + sx) * 4
			if src[si+3] == 0 { // 透明
				continue
			}
			di := (dy*c.W + dx) * 4
			c.Pixels[di] = src[si]
			c.Pixels[di+1] = src[si+1]
			c.Pixels[di+2] = src[si+2]
			c.Pixels[di+3] = 0
		}
	}
}

// Present 把画布分块上传到窗口。
// 关键：xgb 生成的 PutImage 用 16 位请求长度字段，单请求最多 ~262140 字节
// （否则 BadLength），所以按行切成小块逐块上传。
const (
	maxRequestData = 240000 // 每块最大字节数（留余量 < 262140）
)

func (c *Canvas) Present(e *Engine) {
	w := c.W
	h := c.H
	// 每块行数
	rowsPerChunk := maxRequestData / (w * pixBpp)
	if rowsPerChunk < 1 {
		rowsPerChunk = 1
	}
	for y0 := 0; y0 < h; y0 += rowsPerChunk {
		y1 := y0 + rowsPerChunk
		if y1 > h {
			y1 = h
		}
		rows := y1 - y0
		start := y0 * w * pixBpp
		end := y1 * w * pixBpp
		xproto.PutImage(e.Conn, xproto.ImageFormatZPixmap,
			xproto.Drawable(e.Win), e.GC,
			uint16(w), uint16(rows), 0, int16(y0), 0, 24, c.Pixels[start:end])
	}
	e.Conn.Sync()
}

// colorToRGBA 转换（供字体渲染使用）。
func colorToRGBA(col Color) color.RGBA {
	return color.RGBA{R: col.R, G: col.G, B: col.B, A: 255}
}
