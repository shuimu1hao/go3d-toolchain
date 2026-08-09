// Package mesh 提供程序化生成的 3D 网格：立方体、球体、金字塔、地面网格。
//
// 网格由顶点数组 + 三角形面数组组成，每个面带一个颜色（flat shading 用）。
// 顶点环绕顺序为逆时针（从外侧看），供背面剔除使用。
package mesh

import (
	"math"

	"go3d/math3d"
)

// Color 是面颜色（RGB）。
type Color struct {
	R, G, B uint8
}

// Face 是一个三角形面：三个顶点索引 + 颜色。
type Face struct {
	A, B, C int
	Col     Color
}

// Mesh 是静态网格。
type Mesh struct {
	Positions []math3d.Vec3
	Faces     []Face
}

// Col 便捷构造颜色。
func Col(r, g, b uint8) Color { return Color{r, g, b} }

// FaceNormal 计算面法线（未归一化，逆时针为正面）。
func FaceNormal(m *Mesh, f *Face) math3d.Vec3 {
	a := m.Positions[f.A]
	b := m.Positions[f.B]
	c := m.Positions[f.C]
	return b.Sub(a).Cross(c.Sub(a))
}

// Cube 生成立方体，size 为边长，各面颜色不同。
func Cube(size float32) *Mesh {
	s := size / 2
	// 8 个顶点：先 -z 面（0-3），再 +z 面（4-7）
	// 顶点索引布局：
	//   0: (-s,-s,-s)  1: ( s,-s,-s)  2: ( s, s,-s)  3: (-s, s,-s)
	//   4: (-s,-s, s)  5: ( s,-s, s)  6: ( s, s, s)  7: (-s, s, s)
	v := []math3d.Vec3{
		{-s, -s, -s}, {s, -s, -s}, {s, s, -s}, {-s, s, -s},
		{-s, -s, s}, {s, -s, s}, {s, s, s}, {-s, s, s},
	}
	// 6 个面，每面 2 三角形，逆时针（外侧看）
	faces := []Face{
		// -Z 面（红）
		{0, 2, 1, Col(220, 60, 60)}, {0, 3, 2, Col(220, 60, 60)},
		// +Z 面（青）
		{4, 5, 6, Col(60, 200, 200)}, {4, 6, 7, Col(60, 200, 200)},
		// -X 面（橙）
		{4, 3, 0, Col(230, 150, 50)}, {4, 7, 3, Col(230, 150, 50)},
		// +X 面（紫）
		{1, 2, 6, Col(170, 90, 220)}, {1, 6, 5, Col(170, 90, 220)},
		// -Y 面（棕/深）
		{0, 1, 5, Col(140, 100, 70)}, {0, 5, 4, Col(140, 100, 70)},
		// +Y 面（绿）
		{3, 6, 2, Col(80, 200, 100)}, {3, 7, 6, Col(80, 200, 100)},
	}
	return &Mesh{Positions: v, Faces: faces}
}

// Pyramid 生成金字塔：底部在 y=-h/2，顶点在 y=+h/2。底面为正方形。
func Pyramid(base, h float32) *Mesh {
	s := base / 2
	// 底面 4 顶点 + 顶点
	v := []math3d.Vec3{
		{-s, -h / 2, -s}, {s, -h / 2, -s}, {s, -h / 2, s}, {-s, -h / 2, s},
		{0, h / 2, 0},
	}
	faces := []Face{
		// 底面（逆时针从下看：0,1,2 / 0,2,3）
		{0, 2, 1, Col(110, 110, 120)}, {0, 3, 2, Col(110, 110, 120)},
		// 侧面
		{0, 1, 4, Col(220, 140, 60)}, // 前（-Z 方向看）
		{1, 2, 4, Col(60, 160, 220)},
		{2, 3, 4, Col(200, 80, 80)},
		{3, 0, 4, Col(120, 190, 90)},
	}
	return &Mesh{Positions: v, Faces: faces}
}

// Sphere 生成经纬球体。
// segments 为水平分段数（>=3），rings 为垂直环数（>=2）。
// 颜色沿纬度渐变（顶部暖色 → 底部冷色）。
func Sphere(radius float32, segments, rings int) *Mesh {
	if segments < 3 {
		segments = 3
	}
	if rings < 2 {
		rings = 2
	}
	var v []math3d.Vec3
	// 顶点：rings+1 行（含两极），每行 segments 个
	for r := 0; r <= rings; r++ {
		phi := math.Pi * float64(r) / float64(rings) // 0(北)..pi(南)
		sp, cp := math.Sin(phi), math.Cos(phi)
		for s := 0; s < segments; s++ {
			theta := 2 * math.Pi * float64(s) / float64(segments)
			st, ct := math.Sin(theta), math.Cos(theta)
			v = append(v, math3d.Vec3{
				radius * float32(sp*ct),
				radius * float32(cp),
				radius * float32(sp*st),
			})
		}
	}
	idx := func(r, s int) int { return r*segments + s%segments }
	var faces []Face
	for r := 0; r < rings; r++ {
		for s := 0; s < segments; s++ {
			a, b := idx(r, s), idx(r+1, s)
			c, d := idx(r, s+1), idx(r+1, s+1)
			t := float32(r) / float32(rings) // 0 顶 .. 1 底
			// 顶部/底部退化三角形（两极）自动合并，无碍渲染
			faces = append(faces, Face{a, b, d, latColor(t)}, Face{a, d, c, latColor(t)})
		}
	}
	return &Mesh{Positions: v, Faces: faces}
}

// latColor 按纬度给球面色（顶部亮粉 → 中部青 → 底部深蓝）。
func latColor(t float32) Color {
	r := uint8(210 - 130*t)
	g := uint8(150 + 40*t)
	b := uint8(230 - 120*t)
	return Color{r, g, b}
}

// Grid 生成地面网格平面（y 高度，size 为边长，cells 为每边格数）。
// 每个格两个三角形，棋盘格颜色。返回的 Mesh 也可以只用来画线框。
func Grid(y float32, size float32, cells int) *Mesh {
	if cells < 1 {
		cells = 1
	}
	half := size / 2
	step := size / float32(cells)
	var v []math3d.Vec3
	for r := 0; r <= cells; r++ {
		z := -half + step*float32(r)
		for c := 0; c <= cells; c++ {
			x := -half + step*float32(c)
			v = append(v, math3d.Vec3{x, y, z})
		}
	}
	idx := func(r, c int) int { return r*(cells+1) + c }
	var faces []Face
	for r := 0; r < cells; r++ {
		for c := 0; c < cells; c++ {
			a, b := idx(r, c), idx(r, c+1)
			d, e := idx(r+1, c), idx(r+1, c+1)
			checker := (r+c)%2 == 0
			var col Color
			if checker {
				col = Color{90, 110, 130}
			} else {
				col = Color{70, 90, 110}
			}
			// 注意：平面法线朝上（+Y），逆时针需在 XZ 平面按从 +Y 看逆时针排列
			// a(b,c) → b(c+1) → e(c+1, r+1)：保证法线朝 +Y
			faces = append(faces, Face{a, b, e, col}, Face{a, e, d, col})
		}
	}
	return &Mesh{Positions: v, Faces: faces}
}
