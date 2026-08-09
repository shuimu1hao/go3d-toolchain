// lipics - 角色立绘生成器：程序化绘制 Q 版像素立绘并输出 PNG
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

// RGBA 像素颜色（程序化立绘生成用）。
type RGBA struct{ R, G, B uint8 }

// Portrait 立绘（像素矩阵 + 尺寸）。
type Portrait struct {
	Name      string // 输出文件名（不含扩展名）
	Hair, HD  RGBA   // 头发/阴影
	Skin      RGBA
	Eye       RGBA
	Blush     RGBA
	Cloth, CD RGBA   // 衣服/阴影
	Acc       RGBA   // 发饰
	Bust      int    // 0 贫乳 1 普通 2 大 3 超大
	Style     string // long / twins / ponytail / bob
	BG        RGBA
}

const (
	W  = 40 // 画布宽
	H  = 64 // 画布高
	SC = 8  // 放大倍数
)

func std(c RGBA) color.RGBA { return color.RGBA{c.R, c.G, c.B, 255} }

func at(b *image.RGBA, x, y int) RGBA {
	c := b.RGBAAt(x, y)
	return RGBA{c.R, c.G, c.B}
}

func lerp(a, b RGBA, t float64) RGBA {
	return RGBA{
		uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
	}
}

func main() {
	outDir := os.Args[1]
	if outDir == "" {
		outDir = "."
	}
	os.MkdirAll(outDir, 0755)
	portraits := []Portrait{
		{Name: "01_hero_taoyao", Style: "long", Bust: 0,
			Hair: RGBA{60, 46, 40}, HD: RGBA{40, 30, 26}, Skin: RGBA{248, 210, 178},
			Eye: RGBA{90, 70, 160}, Blush: RGBA{250, 170, 160}, Cloth: RGBA{70, 110, 190}, CD: RGBA{50, 86, 152},
			Acc: RGBA{252, 170, 200}, BG: RGBA{70, 100, 170}},
		{Name: "02_qing", Style: "long", Bust: 2,
			Hair: RGBA{96, 64, 48}, HD: RGBA{70, 48, 36}, Skin: RGBA{248, 210, 178},
			Eye: RGBA{70, 110, 80}, Blush: RGBA{250, 170, 160}, Cloth: RGBA{100, 190, 110}, CD: RGBA{74, 150, 86},
			Acc: RGBA{255, 240, 200}, BG: RGBA{60, 130, 80}},
		{Name: "03_luoyao", Style: "ponytail", Bust: 2,
			Hair: RGBA{42, 34, 38}, HD: RGBA{26, 20, 24}, Skin: RGBA{246, 208, 176},
			Eye: RGBA{80, 60, 140}, Blush: RGBA{248, 160, 150}, Cloth: RGBA{208, 82, 70}, CD: RGBA{166, 60, 54},
			Acc: RGBA{255, 210, 100}, BG: RGBA{150, 70, 60}},
		{Name: "04_yueli", Style: "long", Bust: 2,
			Hair: RGBA{216, 216, 230}, HD: RGBA{176, 176, 194}, Skin: RGBA{242, 220, 204},
			Eye: RGBA{90, 120, 190}, Blush: RGBA{240, 180, 170}, Cloth: RGBA{235, 240, 248}, CD: RGBA{196, 204, 220},
			Acc: RGBA{160, 205, 245}, BG: RGBA{120, 140, 180}},
		{Name: "05_xiaoman", Style: "twins", Bust: 2,
			Hair: RGBA{128, 76, 44}, HD: RGBA{94, 54, 32}, Skin: RGBA{250, 214, 180},
			Eye: RGBA{120, 80, 60}, Blush: RGBA{252, 176, 158}, Cloth: RGBA{238, 158, 66}, CD: RGBA{196, 122, 50},
			Acc: RGBA{255, 235, 150}, BG: RGBA{190, 130, 60}},
		{Name: "06_suxue", Style: "long", Bust: 2,
			Hair: RGBA{62, 50, 82}, HD: RGBA{42, 32, 58}, Skin: RGBA{244, 212, 188},
			Eye: RGBA{140, 90, 190}, Blush: RGBA{240, 172, 165}, Cloth: RGBA{158, 108, 190}, CD: RGBA{122, 82, 152},
			Acc: RGBA{210, 170, 250}, BG: RGBA{110, 80, 140}},
		{Name: "07_zhuzhai_feng", Style: "long", Bust: 3,
			Hair: RGBA{176, 60, 56}, HD: RGBA{130, 40, 40}, Skin: RGBA{246, 206, 176},
			Eye: RGBA{200, 60, 60}, Blush: RGBA{244, 150, 140}, Cloth: RGBA{120, 52, 96}, CD: RGBA{88, 36, 72},
			Acc: RGBA{255, 215, 110}, BG: RGBA{140, 60, 90}},
	}
	for _, p := range portraits {
		img := render(p)
		out := filepath.Join(outDir, p.Name+".png")
		f, err := os.Create(out)
		if err != nil {
			panic(err)
		}
		if err := png.Encode(f, img); err != nil {
			panic(err)
		}
		f.Close()
		// 终端 ASCII 预览（验证形状）
		preview(p.Name, img)
	}
}

// render 渲染 40x64 立绘并放大。
func render(p Portrait) *image.RGBA {
	base := image.NewRGBA(image.Rect(0, 0, W, H))
	// 背景渐变
	for y := 0; y < H; y++ {
		t := float64(y) / float64(H)
		c := lerp(p.BG, RGBA{uint8(float64(p.BG.R) * 0.45), uint8(float64(p.BG.G) * 0.45), uint8(float64(p.BG.B) * 0.45)}, t)
		for x := 0; x < W; x++ {
			base.Set(x, y, std(c))
		}
	}
	// 地面阴影
	for y := 58; y < H; y++ {
		for x := 6; x < W-6; x++ {
			base.Set(x, y, std(RGBA{0, 0, 0}))
		}
	}
	set := func(x, y int, c RGBA) {
		if x < 0 || y < 0 || x >= W || y >= H {
			return
		}
		base.Set(x, y, std(c))
	}
	ellipse := func(cx, cy, rx, ry int, c RGBA) {
		for y := cy - ry; y <= cy+ry; y++ {
			for x := cx - rx; x <= cx+rx; x++ {
				if (x-cx)*(x-cx)*ry*ry+(y-cy)*(y-cy)*rx*rx <= rx*rx*ry*ry {
					set(x, y, c)
				}
			}
		}
	}
	fill := func(x0, y0, x1, y1 int, c RGBA) {
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				set(x, y, c)
			}
		}
	}
	// ===== 身体 =====
	// 肩
	fill(13, 23, 26, 26, p.Cloth)
	// 胸围宽度（每侧外凸格数）
	var bump [2]int
	switch p.Bust {
	case 0:
		bump = [2]int{0, 0}
	case 1:
		bump = [2]int{1, 1}
	case 2:
		bump = [2]int{3, 3}
	case 3:
		bump = [2]int{5, 5}
	}
	// 胸区 y=27..35：左右外凸
	for y := 27; y <= 35; y++ {
		lx := 13 - bump[0]
		rx := 26 + bump[1]
		// 向下逐渐收窄到腰
		n := y - 27
		if n > 4 {
			lx += n - 4
			rx -= n - 4
		}
		fill(lx, y, rx, y, p.Cloth)
	}
	// 腰 y=36..44
	for y := 36; y <= 44; y++ {
		off := (y - 36) / 2
		fill(14+off, y, 25-off, y, p.Cloth)
	}
	// 衣服阴影（右侧）
	for y := 23; y <= 44; y++ {
		for x := 26; x < W; x++ {
			if at(base, x, y) == p.Cloth {
				set(x, y, p.CD)
			}
		}
	}
	// 胸口衣领（深色 V 领）
	fill(17, 27, 22, 27, p.CD)
	set(18, 28, p.CD)
	set(21, 28, p.CD)
	// ===== 手臂（袖子）=====
	fill(9, 24, 12, 36, p.Cloth)
	fill(27, 24, 30, 36, p.Cloth)
	for x := 27; x <= 30; x++ {
		set(x, 36, p.CD)
	}
	// 手（肤色）
	ellipse(10, 38, 2, 2, p.Skin)
	ellipse(29, 38, 2, 2, p.Skin)
	// ===== 脖子 + 头 =====
	fill(17, 20, 22, 23, p.Skin)
	// 头（圆）
	ellipse(20, 13, 9, 9, p.Skin)
	// 耳朵
	set(10, 14, p.Skin)
	set(29, 14, p.Skin)
	// ===== 发型 =====
	// 头顶发盖
	ellipse(20, 11, 10, 8, p.Hair)
	// 刘海（前额）
	for y := 10; y <= 14; y++ {
		w := 9 - (y - 10)
		for x := 20 - w; x <= 20+w; x++ {
			if at(base, x, y) == p.Skin {
				set(x, y, p.Hair)
			}
		}
	}
	// 刘海锯齿下缘
	for i, dx := range []int{-9, -6, -3, 0, 3, 6, 9} {
		_ = i
		set(20+dx, 15, p.Hair)
	}
	// 侧发
	switch p.Style {
	case "long", "twins":
		fill(10, 9, 12, 30, p.Hair)
		fill(27, 9, 29, 30, p.Hair)
		// 发梢渐变
		set(10, 31, p.HD)
		set(29, 31, p.HD)
	case "ponytail":
		fill(10, 9, 12, 24, p.Hair)
		fill(27, 9, 29, 24, p.Hair)
	}
	// 双马尾
	if p.Style == "twins" {
		ellipse(6, 7, 4, 7, p.Hair)
		ellipse(33, 7, 4, 7, p.Hair)
		ellipse(6, 14, 2, 3, p.HD)
		ellipse(33, 14, 2, 3, p.HD)
		// 发带
		ellipse(6, 16, 2, 1, p.Acc)
		ellipse(33, 16, 2, 1, p.Acc)
	}
	// 高马尾
	if p.Style == "ponytail" {
		ellipse(28, 4, 5, 6, p.Hair)
		ellipse(28, 11, 3, 3, p.HD)
		ellipse(30, 0, 2, 1, p.Acc)
	}
	// 头顶发饰
	if p.Style == "long" {
		// 发饰（头顶右侧小蝴蝶结）
		set(24, 6, p.Acc)
		set(25, 6, p.Acc)
		set(26, 6, p.Acc)
		set(25, 5, p.Acc)
		set(25, 7, p.Acc)
	}
	// ===== 眼睛 =====
	eye := func(cx int) {
		// 眼白
		ellipse(cx, 13, 2, 2, RGBA{255, 255, 255})
		// 瞳孔
		ellipse(cx, 14, 1, 1, p.Eye)
		// 睫毛（上）
		set(cx-1, 11, p.HD)
		set(cx, 11, p.HD)
		set(cx+1, 11, p.HD)
	}
	eye(16)
	eye(24)
	// ===== 腮红 =====
	ellipse(13, 16, 2, 1, p.Blush)
	ellipse(27, 16, 2, 1, p.Blush)
	// ===== 嘴 =====
	fill(19, 17, 20, 17, RGBA{190, 90, 90})
	set(18, 18, RGBA{230, 130, 120})
	set(21, 18, RGBA{230, 130, 120})
	// ===== 衣领挂饰 =====
	ellipse(20, 30, 2, 2, p.Acc)
	// 放大输出
	out := image.NewRGBA(image.Rect(0, 0, W*SC, H*SC))
	for y := 0; y < H*SC; y++ {
		for x := 0; x < W*SC; x++ {
			c := base.RGBAAt(x/SC, y/SC)
			out.Set(x, y, c)
		}
	}
	return out
}

// preview 终端 ASCII 预览（映射角色区域的颜色种类）。
func preview(name string, img *image.RGBA) {
	chars := " .:-=+*#%@"
	for y := 0; y < H; y++ {
		line := ""
		for x := 0; x < W; x++ {
			c := img.RGBAAt(x*SC+2, y*SC+2)
			bright := (int(c.R) + int(c.G) + int(c.B)) / 3
			idx := bright * (len(chars) - 1) / 255
			line += string(chars[idx])
		}
		println(line)
	}
	println("==== " + name + " ====")
}
