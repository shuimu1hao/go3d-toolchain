package app

import (
	"math"

	"go3d/math3d"
	"go3d/mesh"
)

// SketchPlane 草图平面类型（SolidWorks 基准面）。
type SketchPlane int

const (
	PlaneXY SketchPlane = iota // 前视基准面：局部(u,v) → 世界(u,v,0)
	PlaneXZ                    // 顶视基准面：局部(u,v) → 世界(u,0,v)
	PlaneYZ                    // 右视基准面：局部(u,v) → 世界(0,u,v)
)

// Sketch 是一个 2D 草图（点序列 + 闭合状态）。
type Sketch struct {
	Plane  SketchPlane
	Points []Vec2
	Closed bool
}

// Vec2 是草图平面坐标。
type Vec2 struct {
	U, V float32
}

// NewSketch 创建空草图。
func NewSketch(plane SketchPlane) *Sketch {
	return &Sketch{Plane: plane, Points: []Vec2{}}
}

// AddPoint 追加点。
func (s *Sketch) AddPoint(u, v float32) {
	s.Points = append(s.Points, Vec2{u, v})
}

// Clear 清空。
func (s *Sketch) Clear() {
	s.Points = s.Points[:0]
	s.Closed = false
}

// LocalToWorld 局部草图坐标 → 世界坐标（平面在原点）。
func (s *Sketch) LocalToWorld(p Vec2) math3d.Vec3 {
	switch s.Plane {
	case PlaneXY:
		return math3d.Vec3{p.U, p.V, 0}
	case PlaneXZ:
		return math3d.Vec3{p.U, 0, p.V}
	case PlaneYZ:
		return math3d.Vec3{0, p.U, p.V}
	}
	return math3d.Vec3{}
}

// WorldToLocal 世界坐标 → 局部草图坐标。
func (s *Sketch) WorldToLocal(wp math3d.Vec3) Vec2 {
	switch s.Plane {
	case PlaneXY:
		return Vec2{wp.X, wp.Y}
	case PlaneXZ:
		return Vec2{wp.X, wp.Z}
	case PlaneYZ:
		return Vec2{wp.Y, wp.Z}
	}
	return Vec2{}
}

// Normal 返回草图平面法线（世界，单位向量）。
func (s *Sketch) Normal() math3d.Vec3 {
	switch s.Plane {
	case PlaneXY:
		return math3d.Vec3{0, 0, 1}
	case PlaneXZ:
		return math3d.Vec3{0, 1, 0}
	case PlaneYZ:
		return math3d.Vec3{1, 0, 0}
	}
	return math3d.Vec3{0, 0, 1}
}

// ---------- 2D 几何工具 ----------

// triArea2 返回有符号面积×2（CCW 为正）。
func triArea2(a, b, c Vec2) float32 {
	return (b.U-a.U)*(c.V-a.V) - (b.V-a.V)*(c.U-a.U)
}

// convexAt 判断索引 i 处是否为凸（内角 < 180°）。
// 多边形为 CCW 时，triArea2(prev, cur, next) > 0 为凸。
func convexAt(pts []Vec2, i int) bool {
	n := len(pts)
	return triArea2(pts[(i+n-1)%n], pts[i], pts[(i+1)%n]) > 0
}

// pointInTri 判断 p 是否在三角形内（含边界）。
func pointInTri(p, a, b, c Vec2) bool {
	d1 := triArea2(a, b, p)
	d2 := triArea2(b, c, p)
	d3 := triArea2(c, a, p)
	// 与三角形方向同号（CCW：全 >= 0）
	neg := (d1 < 0) || (d2 < 0) || (d3 < 0)
	pos := (d1 > 0) || (d2 > 0) || (d3 > 0)
	return !(neg && pos)
}

// TriangulateEar 耳切法三角化简单多边形（CCW 或 CW 均可，无孔洞/自交）。
// 返回三角形顶点索引（相对 pts）。
func TriangulateEar(pts []Vec2) [][3]int {
	n := len(pts)
	if n < 3 {
		return nil
	}
	// 若为 CW，翻转顺序（简化处理）
	area := float32(0)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += pts[i].U*pts[j].V - pts[j].U*pts[i].V
	}
	work := make([]int, n)
	for i := range work {
		work[i] = i
	}
	if area < 0 {
		for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
			work[i], work[j] = work[j], work[i]
		}
	}
	var out [][3]int
	guard := 0
	for len(work) > 3 && guard < n*10 {
		guard++
		removed := false
		m := len(work)
		for i := 0; i < m; i++ {
			a := work[(i+m-1)%m]
			b := work[i]
			c := work[(i+1)%m]
			if triArea2(pts[a], pts[b], pts[c]) <= 0 {
				continue // 凹或退化
			}
			// 检查其他点是否在三角形内
			ear := true
			for j := 0; j < m; j++ {
				k := work[j]
				if k == a || k == b || k == c {
					continue
				}
				if pointInTri(pts[k], pts[a], pts[b], pts[c]) {
					ear = false
					break
				}
			}
			if !ear {
				continue
			}
			out = append(out, [3]int{a, b, c})
			work = append(work[:i], work[i+1:]...)
			removed = true
			break
		}
		if !removed {
			break // 无法继续（退化输入）
		}
	}
	if len(work) == 3 {
		out = append(out, [3]int{work[0], work[1], work[2]})
	}
	return out
}

// ---------- 拉伸（Extrude） ----------

// ExtrudeMesh 把闭合轮廓拉伸成 3D 实体。
// 轮廓在局部平面内；depth 为拉伸深度（沿平面法线）。
// 返回世界坐标网格（平面位于原点，拉伸沿法线方向）。
func ExtrudeMesh(pts []Vec2, plane SketchPlane, depth float32, col mesh.Color) *mesh.Mesh {
	if len(pts) < 3 || depth == 0 {
		return nil
	}
	// 三角化轮廓
	tris := TriangulateEar(pts)
	if len(tris) == 0 {
		return nil
	}
	// 顶点：底部层 + 顶部层
	bottom := make([]math3d.Vec3, len(pts))
	top := make([]math3d.Vec3, len(pts))
	local := func(p Vec2, z float32) math3d.Vec3 {
		switch plane {
		case PlaneXY:
			return math3d.Vec3{p.U, p.V, z}
		case PlaneXZ:
			return math3d.Vec3{p.U, z, p.V}
		case PlaneYZ:
			return math3d.Vec3{z, p.U, p.V}
		}
		return math3d.Vec3{}
	}
	for i, p := range pts {
		bottom[i] = local(p, 0)
		top[i] = local(p, depth)
	}
	var m mesh.Mesh
	m.Positions = append(bottom, top...)
	topOff := len(bottom)

	// 底部面（朝 -法线）：翻转三角化顺序
	for _, t := range tris {
		m.Faces = append(m.Faces, mesh.Face{A: t[0], B: t[2], C: t[1], Col: col})
	}
	// 顶部面（朝 +法线）
	for _, t := range tris {
		m.Faces = append(m.Faces, mesh.Face{A: topOff + t[0], B: topOff + t[1], C: topOff + t[2], Col: col})
	}
	// 侧面：每条轮廓边生成 quad（两三角形），法线朝外
	n := len(pts)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		b0, b1 := i, j
		t0, t1 := topOff+i, topOff+j
		// 侧面三角形顺序需保证法线朝外（取决于轮廓方向；两三角形均给）
		m.Faces = append(m.Faces,
			mesh.Face{A: b0, B: b1, C: t1, Col: col},
			mesh.Face{A: b0, B: t1, C: t0, Col: col},
		)
	}
	return &m
}

// ---------- 草图形状生成 ----------

// RectSketch 生成矩形轮廓点（逆时针）。
func RectSketch(u0, v0, u1, v1 float32) []Vec2 {
	return []Vec2{
		{u0, v0}, {u1, v0}, {u1, v1}, {u0, v1},
	}
}

// CircleSketch 生成圆轮廓点（segments 段，逆时针）。
func CircleSketch(cu, cv, r float32, segments int) []Vec2 {
	if segments < 8 {
		segments = 8
	}
	pts := make([]Vec2, segments)
	for i := 0; i < segments; i++ {
		th := 2 * math.Pi * float64(i) / float64(segments)
		pts[i] = Vec2{cu + r*float32(math.Cos(th)), cv + r*float32(math.Sin(th))}
	}
	return pts
}

// CircleMesh 生成圆盘网格（顶视平面拉伸薄片），供"圆"基本体使用。
func CircleMesh(radius float32, segments int, col mesh.Color) *mesh.Mesh {
	pts := CircleSketch(0, 0, radius, segments)
	return ExtrudeMesh(pts, PlaneXY, 0.1, col)
}
