package app

import (
	"math"

	"go3d/math3d"
	"go3d/mesh"
)

// CSG 布尔运算类型。
type CSGOp int

const (
	CSGUnion CSGOp = iota // 并集
	CSGSubtract           // 差集 A - B
	CSGIntersect          // 交集
)

// csgFace 是带平面方程的三角形。
type csgFace struct {
	a, b, c math3d.Vec3
	n       math3d.Vec3 // 单位法线
	d       float32     // 平面: n·p + d = 0
	col     mesh.Color
}

func newCSGFace(a, b, c math3d.Vec3, col mesh.Color) csgFace {
	n := b.Sub(a).Cross(c.Sub(a))
	l := n.Length()
	if l < 1e-12 {
		n = math3d.Vec3{0, 1, 0}
		l = 1
	}
	n = n.MulScalar(1 / l)
	return csgFace{a: a, b: b, c: c, n: n, d: -n.Dot(a), col: col}
}

// planeDist 点到平面的有符号距离（>0 在法线侧）。
func (f *csgFace) planeDist(p math3d.Vec3) float32 {
	return f.n.Dot(p) + f.d
}

// clipPoly 用平面把多边形裁剪成 front/back 两块（Sutherland-Hodgman）。
// 返回两块多边形，以及平面上产生的切割交点列表（成对，用于补面）。
func (f *csgFace) clipPoly(poly []math3d.Vec3) (front, back []math3d.Vec3, cuts []math3d.Vec3) {
	if len(poly) < 3 {
		return
	}
	for i := 0; i < len(poly); i++ {
		cur := poly[i]
		next := poly[(i+1)%len(poly)]
		dc := f.planeDist(cur)
		dn := f.planeDist(next)
		if dc >= -1e-7 {
			front = append(front, cur)
		}
		if dc <= 1e-7 {
			back = append(back, cur)
		}
		if dc*dn < 0 {
			t := dc / (dc - dn)
			ip := cur.Lerp(next, t)
			front = append(front, ip)
			back = append(back, ip)
			cuts = append(cuts, ip)
		}
	}
	return
}

// triangulatePoly 把任意多边形三角化（扇形，对凸多边形正确；裁剪产物通常凸）。
func triangulatePoly(poly []math3d.Vec3) [][3]math3d.Vec3 {
	var out [][3]math3d.Vec3
	if len(poly) < 3 {
		return out
	}
	// 简单扇形（适用于凸多边形）；裁剪后的多边形可能凹，用耳切在 3D 投影
	// 计算平面法线确定投影方向
	if len(poly) == 3 {
		return append(out, [3]math3d.Vec3{poly[0], poly[1], poly[2]})
	}
	// 用耳切法（2D 投影）
	// 找最大法线分量方向
	n := poly[1].Sub(poly[0]).Cross(poly[2].Sub(poly[0]))
	axis := 0
	nx, ny, nz := n.X, n.Y, n.Z
	if math.Abs(float64(ny)) > math.Abs(float64(nx)) {
		axis = 1
	}
	if math.Abs(float64(nz)) > math.Abs(float64(axisVec(axis, nx, ny, nz))) {
		axis = 2
	}
	type p2 struct{ u, v float32 }
	pts2 := make([]Vec2, len(poly))
	idx := make([]int, len(poly))
	for i, p := range poly {
		switch axis {
		case 0:
			pts2[i] = Vec2{p.Y, p.Z}
		case 1:
			pts2[i] = Vec2{p.X, p.Z}
		default:
			pts2[i] = Vec2{p.X, p.Y}
		}
		idx[i] = i
	}
	tris := TriangulateEar(pts2)
	for _, t := range tris {
		out = append(out, [3]math3d.Vec3{poly[t[0]], poly[t[1]], poly[t[2]]})
	}
	return out
}

// pointInMesh 射线法：点是否在封闭网格内部（奇数次相交）。
func pointInMesh(p math3d.Vec3, faces []csgFace) bool {
	// 射线方向：选一个不平行于任何面法线的方向
	dir := math3d.Vec3{0.3377, 0.9003, 0.2749} // 固定任意方向
	hits := 0
	for i := range faces {
		f := &faces[i]
		if math.Abs(float64(f.n.Dot(dir))) < 1e-6 {
			continue
		}
		t := -(f.n.Dot(p) + f.d) / f.n.Dot(dir)
		if t <= 1e-6 {
			continue
		}
		q := p.Add(dir.MulScalar(t))
		if pointInTri3D(q, f.a, f.b, f.c) {
			hits++
		}
	}
	return hits%2 == 1
}

// pointInTri3D 判断 q 是否在三角形内（投影到三角形平面）。
func pointInTri3D(q, a, b, c math3d.Vec3) bool {
	n := b.Sub(a).Cross(c.Sub(a))
	// 重心坐标（3D 面积比）
	u := b.Sub(a).Cross(q.Sub(a)).Dot(n)
	v := c.Sub(b).Cross(q.Sub(b)).Dot(n)
	w := a.Sub(c).Cross(q.Sub(c)).Dot(n)
	// 与总面积同号
	neg := (u < 0) || (v < 0) || (w < 0)
	pos := (u > 0) || (v > 0) || (w > 0)
	return !(neg && pos)
}

// CSGBoolean 对 A 和 B 做布尔运算，返回结果网格。
// 两个输入都必须是封闭网格（无孔洞）。
func CSGBoolean(ma, mb *mesh.Mesh, op CSGOp, colA, colB mesh.Color) *mesh.Mesh {
	facesA := meshToCSGFaces(ma, colA)
	facesB := meshToCSGFaces(mb, colB)

	// 1. A 的面被 B 的所有面裁剪
	clippedA, boundaryA := clipAll(facesA, facesB)
	// 2. B 的面被 A 的所有面裁剪
	clippedB, boundaryB := clipAll(facesB, facesA)

	var out []csgFace
	// 分类点：碎片中心沿法线微偏移（共面表面碎片稳定判定内外）
	classP := func(f csgFace) math3d.Vec3 {
		return midPoint(f).Add(f.n.MulScalar(1e-4))
	}
	switch op {
	case CSGUnion:
		for _, f := range clippedA {
			if !pointInMesh(classP(f), facesB) {
				out = append(out, f)
			}
		}
		for _, f := range clippedB {
			if !pointInMesh(classP(f), facesA) {
				out = append(out, f)
			}
		}
		// union 的缺口由对方表面覆盖，无需补面
	case CSGSubtract:
		for _, f := range clippedA {
			if !pointInMesh(classP(f), facesB) {
				out = append(out, f)
			}
		}
		for _, f := range clippedB {
			if pointInMesh(classP(f), facesA) {
				// 翻转法线（差集的内表面）
				fl := f
				fl.a, fl.c = f.c, f.a
				out = append(out, fl)
			}
		}
	case CSGIntersect:
		for _, f := range clippedA {
			if pointInMesh(classP(f), facesB) {
				out = append(out, f)
			}
		}
		for _, f := range clippedB {
			if pointInMesh(classP(f), facesA) {
				out = append(out, f)
			}
		}
	}
	// 补面：差集/交集的裁剪边界需要封闭内表面
	planesA := uniquePlanes(facesA)
	planesB := uniquePlanes(facesB)
	switch op {
	case CSGSubtract:
		out = append(out, buildCaps(boundaryA, planesB, colB)...)
		out = append(out, buildCaps(boundaryB, planesA, colA)...)
	case CSGIntersect:
		out = append(out, buildCaps(boundaryA, planesB, colB)...)
		out = append(out, buildCaps(boundaryB, planesA, colA)...)
	}
	return csgFacesToMesh(out)
}

// uniquePlanes 返回去重后的平面列表（对应 clipAll 的 planes 顺序）。
func uniquePlanes(faces []csgFace) []csgFace {
	var out []csgFace
	for _, c := range faces {
		dup := false
		for _, p := range out {
			if p.n.Dot(c.n) > 0.999 && math.Abs(float64(p.d-c.d)) < 1e-4 {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, c)
		}
	}
	return out
}

// midPoint 三角形重心。
func midPoint(f csgFace) math3d.Vec3 {
	return f.a.Add(f.b).Add(f.c).MulScalar(1.0 / 3.0)
}

// axisVec 辅助：取对应轴的法线分量。
func axisVec(axis int, x, y, z float32) float32 {
	switch axis {
	case 0:
		return x
	case 1:
		return y
	default:
		return z
	}
}

// meshToCSGFaces 网格 → CSG 面。
func meshToCSGFaces(m *mesh.Mesh, tint mesh.Color) []csgFace {
	var out []csgFace
	for i := range m.Faces {
		f := &m.Faces[i]
		col := f.Col
		if tint.R != 0 || tint.G != 0 || tint.B != 0 {
			col = tint
		}
		out = append(out, newCSGFace(
			m.Positions[f.A], m.Positions[f.B], m.Positions[f.C], col))
	}
	return out
}

// clipAll 用 cutter 的所有面平面把 faces 全部裁剪成碎片，并收集补面边界。
// 返回 (碎片, 补面边 map[平面ID][]边)。
func clipAll(faces, cutter []csgFace) ([]csgFace, map[int][][2]math3d.Vec3) {
	// 收集所有平面（去重：法线/距离近似）
	var planes []csgFace
	for _, c := range cutter {
		dup := false
		for _, p := range planes {
			if p.n.Dot(c.n) > 0.999 && math.Abs(float64(p.d-c.d)) < 1e-4 {
				dup = true
				break
			}
		}
		if !dup {
			planes = append(planes, c)
		}
	}
	// 平面 ID：平面数组中索引
	planeID := func(f csgFace) int {
		for i, p := range planes {
			if p.n.Dot(f.n) > 0.999 && math.Abs(float64(p.d-f.d)) < 1e-4 {
				return i
			}
		}
		return -1
	}
	boundary := map[int][][2]math3d.Vec3{}
	var out []csgFace
	for _, f := range faces {
		polys := [][]math3d.Vec3{{f.a, f.b, f.c}}
		col := f.col
		// 记录每次裁剪产生的平面切割边
		for _, p := range planes {
			pid := planeID(p)
			var next [][]math3d.Vec3
			for _, poly := range polys {
				// 包围盒剔除：碎片与裁剪三角形不相交则跳过（大幅减少曲面网格的裁剪量）
				if !polyBBoxOverlap(poly, p) {
					next = append(next, poly)
					continue
				}
				fr, bk, cuts := p.clipPoly(poly)
				// 切割交点成对 → 补面边
				for ci := 0; ci+1 < len(cuts); ci += 2 {
					boundary[pid] = append(boundary[pid], [2]math3d.Vec3{cuts[ci], cuts[ci+1]})
				}
				if len(fr) >= 3 {
					next = append(next, fr)
				}
				if len(bk) >= 3 {
					next = append(next, bk)
				}
			}
			polys = next
			if len(polys) == 0 {
				break
			}
		}
		for _, poly := range polys {
			for _, tri := range triangulatePoly(poly) {
				out = append(out, newCSGFace(tri[0], tri[1], tri[2], col))
			}
		}
	}
	return out, boundary
}

// polyBBoxOverlap 判断多边形与三角形包围盒是否相交。
func polyBBoxOverlap(poly []math3d.Vec3, f csgFace) bool {
	minP, maxP := poly[0], poly[0]
	for _, p := range poly[1:] {
		if p.X < minP.X {
			minP.X = p.X
		}
		if p.Y < minP.Y {
			minP.Y = p.Y
		}
		if p.Z < minP.Z {
			minP.Z = p.Z
		}
		if p.X > maxP.X {
			maxP.X = p.X
		}
		if p.Y > maxP.Y {
			maxP.Y = p.Y
		}
		if p.Z > maxP.Z {
			maxP.Z = p.Z
		}
	}
	// 三角形 bbox（含 1e-4 容差）
	eps := float32(1e-4)
	minF := f.a
	maxF := f.a
	for _, q := range []math3d.Vec3{f.b, f.c} {
		if q.X < minF.X {
			minF.X = q.X
		}
		if q.Y < minF.Y {
			minF.Y = q.Y
		}
		if q.Z < minF.Z {
			minF.Z = q.Z
		}
		if q.X > maxF.X {
			maxF.X = q.X
		}
		if q.Y > maxF.Y {
			maxF.Y = q.Y
		}
		if q.Z > maxF.Z {
			maxF.Z = q.Z
		}
	}
	return minP.X <= maxF.X+eps && maxP.X >= minF.X-eps &&
		minP.Y <= maxF.Y+eps && maxP.Y >= minF.Y-eps &&
		minP.Z <= maxF.Z+eps && maxP.Z >= minF.Z-eps
}

// buildCaps 根据补面边界边重建闭合多边形并三角化补面。
// 每条边界边属于某个裁剪平面；按平面分组后把边链接成环。
func buildCaps(boundary map[int][][2]math3d.Vec3, planes []csgFace, col mesh.Color) []csgFace {
	var out []csgFace
	for pid, rawEdges := range boundary {
		if len(rawEdges) < 3 {
			continue
		}
		_ = planes[pid] // 平面法线/距离由环自身携带，此处仅分组
		// 去重：同一几何边只保留一条（无序比较端点）
		edges := dedupEdges(rawEdges)
		if len(edges) < 3 {
			continue
		}
		// 边链接成环：贪心匹配端点
		used := make([]bool, len(edges))
		var loops [][]math3d.Vec3
		for i := 0; i < len(edges); i++ {
			if used[i] {
				continue
			}
			used[i] = true
			loop := []math3d.Vec3{edges[i][0], edges[i][1]}
			for {
				advanced := false
				tail := loop[len(loop)-1]
				for j := 0; j < len(edges); j++ {
					if used[j] {
						continue
					}
					if dist3(edges[j][0], tail) < 1e-4 {
						loop = append(loop, edges[j][1])
						used[j] = true
						advanced = true
						break
					}
					if dist3(edges[j][1], tail) < 1e-4 {
						loop = append(loop, edges[j][0])
						used[j] = true
						advanced = true
						break
					}
				}
				if !advanced {
					break
				}
			}
			if len(loop) >= 3 {
				loops = append(loops, loop)
			}
		}
		// 三角化每个环
		for _, loop := range loops {
			// 去环首尾重复点
			for len(loop) > 1 && dist3(loop[0], loop[len(loop)-1]) < 1e-4 {
				loop = loop[:len(loop)-1]
			}
			if len(loop) < 3 {
				continue
			}
			for _, tri := range triangulatePoly(loop) {
				out = append(out, newCSGFace(tri[0], tri[1], tri[2], col))
			}
		}
	}
	return out
}

// dedupEdges 边去重：同一几何边（无序）只保留一条。
func dedupEdges(edges [][2]math3d.Vec3) [][2]math3d.Vec3 {
	seen := map[[6]float32]bool{}
	var out [][2]math3d.Vec3
	for _, e := range edges {
		a, b := e[0], e[1]
		// 归一化：按字典序排端点
		key := edgeKey(a, b)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, e)
	}
	return out
}

// edgeKey 生成边的无序标识（量化容差 1e-3）。
func edgeKey(a, b math3d.Vec3) [6]float32 {
	q := func(v float32) float32 { return float32(math.Round(float64(v)*1e3)) / 1e3 }
	av := [3]float32{q(a.X), q(a.Y), q(a.Z)}
	bv := [3]float32{q(b.X), q(b.Y), q(b.Z)}
	if av[0] > bv[0] || (av[0] == bv[0] && av[1] > bv[1]) || (av[0] == bv[0] && av[1] == bv[1] && av[2] > bv[2]) {
		av, bv = bv, av
	}
	return [6]float32{av[0], av[1], av[2], bv[0], bv[1], bv[2]}
}

// dist3 两点距离。
func dist3(a, b math3d.Vec3) float32 {
	return a.Sub(b).Length()
}

// csgFacesToMesh 面列表 → 网格（每个三角形独立顶点）。
func csgFacesToMesh(faces []csgFace) *mesh.Mesh {
	m := &mesh.Mesh{}
	for _, f := range faces {
		base := len(m.Positions)
		m.Positions = append(m.Positions, f.a, f.b, f.c)
		m.Faces = append(m.Faces, mesh.Face{A: base, B: base + 1, C: base + 2, Col: f.col})
	}
	return m
}
