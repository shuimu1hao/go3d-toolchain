// 扩展基本体：圆柱、圆锥、圆环。环绕顺序逆时针（外侧看）。
package mesh

import (
	"math"

	"go3d/math3d"
)

// Cylinder 生成圆柱：底面在 y=-h/2，顶面在 y=+h/2，半径 radius。
// segments 为圆周分段数（>=3）。
func Cylinder(radius, height float32, segments int) *Mesh {
	if segments < 3 {
		segments = 3
	}
	half := height / 2
	var v []math3d.Vec3
	// 顶点：底圆（0..segments-1）、顶圆（segments..2*segments-1）、底心、顶心
	for i := 0; i < segments; i++ {
		a := 2 * math.Pi * float64(i) / float64(segments)
		ca, sa := float32(math.Cos(a)), float32(math.Sin(a))
		v = append(v, math3d.Vec3{radius * ca, -half, radius * sa})
	}
	for i := 0; i < segments; i++ {
		a := 2 * math.Pi * float64(i) / float64(segments)
		ca, sa := float32(math.Cos(a)), float32(math.Sin(a))
		v = append(v, math3d.Vec3{radius * ca, half, radius * sa})
	}
	bottomC := 2 * segments
	topC := 2*segments + 1
	v = append(v, math3d.Vec3{0, -half, 0}, math3d.Vec3{0, half, 0})

	var faces []Face
	baseCol := Color{120, 170, 220}
	capCol := Color{200, 130, 90}
	for i := 0; i < segments; i++ {
		j := (i + 1) % segments
		// 侧壁：底面 i→顶面 i→顶面 j，再底面 i→顶面 j→底面 j（逆时针从外侧看）
		faces = append(faces,
			Face{i, segments + i, segments + j, baseCol},
			Face{i, segments + j, j, baseCol},
			// 底盖（从下方看逆时针：j→i→底心）
			Face{j, i, bottomC, capCol},
			// 顶盖（从上方看逆时针：i→j→顶心）
			Face{segments + i, segments + j, topC, capCol},
		)
	}
	return &Mesh{Positions: v, Faces: faces}
}

// Cone 生成圆锥：底面在 y=-h/2，顶点在 y=+h/2，半径 radius。
func Cone(radius, height float32, segments int) *Mesh {
	if segments < 3 {
		segments = 3
	}
	half := height / 2
	var v []math3d.Vec3
	for i := 0; i < segments; i++ {
		a := 2 * math.Pi * float64(i) / float64(segments)
		ca, sa := float32(math.Cos(a)), float32(math.Sin(a))
		v = append(v, math3d.Vec3{radius * ca, -half, radius * sa})
	}
	apex := segments
	bottomC := segments + 1
	v = append(v, math3d.Vec3{0, half, 0}, math3d.Vec3{0, -half, 0})

	var faces []Face
	sideCol := Color{230, 150, 70}
	capCol := Color{120, 190, 110}
	for i := 0; i < segments; i++ {
		j := (i + 1) % segments
		// 侧面：底 i→底 j→顶点（逆时针从外侧看）
		faces = append(faces, Face{i, j, apex, sideCol})
		// 底盖：从下方看逆时针 j→i→底心
		faces = append(faces, Face{j, i, bottomC, capCol})
	}
	return &Mesh{Positions: v, Faces: faces}
}

// Torus 生成圆环：主半径 R（环中心到管中心），管半径 r。
// segments 为环绕主圆的分段数，rings 为管截面分段数。
func Torus(R, r float32, segments, rings int) *Mesh {
	if segments < 3 {
		segments = 3
	}
	if rings < 3 {
		rings = 3
	}
	var v []math3d.Vec3
	idx := func(s, g int) int { return s*rings + g%rings }
	for s := 0; s < segments; s++ {
		th := 2 * math.Pi * float64(s) / float64(segments)
		ct, st := float32(math.Cos(th)), float32(math.Sin(th))
		for g := 0; g < rings; g++ {
			ph := 2 * math.Pi * float64(g) / float64(rings)
			cp, sp := float32(math.Cos(ph)), float32(math.Sin(ph))
			// 管中心绕主圆：x=R*ct, z=R*st；管内点 = 管中心 + r*(cp*ct, sp, cp*st)
			v = append(v, math3d.Vec3{
				(R + r*cp) * ct,
				r * sp,
				(R + r*cp) * st,
			})
		}
	}
	var faces []Face
	col := Color{160, 120, 220}
	for s := 0; s < segments; s++ {
		for g := 0; g < rings; g++ {
			a, b := idx(s, g), idx(s+1, g)
			c, d := idx(s, g+1), idx(s+1, g+1)
			faces = append(faces, Face{a, b, d, col}, Face{a, d, c, col})
		}
	}
	return &Mesh{Positions: v, Faces: faces}
}
