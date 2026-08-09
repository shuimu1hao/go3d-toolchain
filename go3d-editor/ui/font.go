// Package ui 提供编辑器 UI 基础：CJK 中文渲染（NotoSansCJK + opentype）。
package ui

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"strings"

	"go2dgame/engine"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

var cjkFace font.Face

// LoadCJK 加载中文字体（ttf/otf/ttc）。
// ttc（TrueType Collection）自动取简体中文（CJKsc）子字体。
func LoadCJK(path string, size float64) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var f *opentype.Font
	if len(data) >= 4 && string(data[:4]) == "ttcf" {
		coll, err := opentype.ParseCollection(data)
		if err != nil {
			return err
		}
		var buf sfnt.Buffer
		for i := 0; i < coll.NumFonts(); i++ {
			ft, err := coll.Font(i)
			if err != nil {
				continue
			}
			name, err := ft.Name(&buf, sfnt.NameIDPostScript)
			if err == nil && strings.Contains(name, "CJKsc") {
				f = ft
				break
			}
		}
		if f == nil {
			return errors.New("ttc 中未找到 CJKsc 子字体")
		}
	} else {
		f, err = opentype.Parse(data)
		if err != nil {
			return err
		}
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	cjkFace = face
	fmt.Printf("[ui] 中文字体加载成功: %s (size=%g)\n", path, size)
	return nil
}

// TextW 返回文本像素宽度。
func TextW(s string) int {
	if cjkFace == nil {
		return len(s) * 7
	}
	return font.MeasureString(cjkFace, s).Ceil()
}

// LineH 返回行高（像素）。
func LineH() int {
	if cjkFace == nil {
		return 14
	}
	m := cjkFace.Metrics()
	return (m.Ascent + m.Descent).Ceil()
}

// DrawText 在画布 (x,y) 左上角绘制文本（中文/ASCII 均支持）。
func DrawText(c *engine.Canvas, x, y int, s string, col engine.Color) {
	if s == "" {
		return
	}
	if cjkFace == nil {
		c.Text(x, y, s, col, 1)
		return
	}
	w := TextW(s)
	m := cjkFace.Metrics()
	h := (m.Ascent + m.Descent).Ceil()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{R: col.R, G: col.G, B: col.B, A: 255}),
		Face: cjkFace,
		Dot:  fixed.P(0, m.Ascent.Ceil()),
	}
	d.DrawString(s)
	for yy := 0; yy < h; yy++ {
		cy := y + yy
		if cy < 0 || cy >= c.H {
			continue
		}
		for xx := 0; xx < w; xx++ {
			cx := x + xx
			if cx < 0 || cx >= c.W {
				continue
			}
			r, g, b, a := img.At(xx, yy).RGBA()
			if a == 0 {
				continue
			}
			i := (cy*c.W + cx) * 4
			sr, sg, sb := byte(r>>8), byte(g>>8), byte(b>>8)
			sa := int(a >> 8)
			c.Pixels[i] = byte((int(sb)*sa + int(c.Pixels[i])*(255-sa)) / 255)
			c.Pixels[i+1] = byte((int(sg)*sa + int(c.Pixels[i+1])*(255-sa)) / 255)
			c.Pixels[i+2] = byte((int(sr)*sa + int(c.Pixels[i+2])*(255-sa)) / 255)
			c.Pixels[i+3] = 0
		}
	}
}

// DrawTextCenter 水平居中绘制（cx 是中心 x）。
func DrawTextCenter(c *engine.Canvas, cx, y int, s string, col engine.Color) {
	DrawText(c, cx-TextW(s)/2, y, s, col)
}

// DrawTextRight 右对齐绘制（right 是右边缘 x）。
func DrawTextRight(c *engine.Canvas, right, y int, s string, col engine.Color) {
	DrawText(c, right-TextW(s), y, s, col)
}

// DrawTextShadow 带深色阴影的文本。
func DrawTextShadow(c *engine.Canvas, x, y int, s string, col engine.Color) {
	DrawText(c, x+1, y+1, s, engine.Color{R: 20, G: 18, B: 24})
	DrawText(c, x, y, s, col)
}
