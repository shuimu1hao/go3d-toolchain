// Package math3d 提供 3D 游戏引擎所需的最小线性代数：三维向量与 4x4 矩阵。
//
// 约定：
//   - 列主序矩阵（column-major），与 OpenGL/标准图形学一致。
//   - 向量视为列向量，变换 v' = M * v。
//   - 矩阵乘法 Mul(A, B) = A*B，表示“先应用 B，再应用 A”。
//   - 右手坐标系：相机位于原点看向 -Z，Y 朝上。
package math3d

import "math"

// Vec3 是三维向量。
type Vec3 struct {
	X, Y, Z float32
}

// Add 向量加法。
func (v Vec3) Add(o Vec3) Vec3 { return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }

// Sub 向量减法。
func (v Vec3) Sub(o Vec3) Vec3 { return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }

// MulScalar 数乘。
func (v Vec3) MulScalar(s float32) Vec3 { return Vec3{v.X * s, v.Y * s, v.Z * s} }

// Dot 点积。
func (v Vec3) Dot(o Vec3) float32 { return v.X*o.X + v.Y*o.Y + v.Z*o.Z }

// Cross 叉积。
func (v Vec3) Cross(o Vec3) Vec3 {
	return Vec3{
		v.Y*o.Z - v.Z*o.Y,
		v.Z*o.X - v.X*o.Z,
		v.X*o.Y - v.Y*o.X,
	}
}

// Length 向量长度。
func (v Vec3) Length() float32 {
	return float32(math.Sqrt(float64(v.Dot(v))))
}

// Normalized 归一化向量（零向量返回自身）。
func (v Vec3) Normalized() Vec3 {
	l := v.Length()
	if l < 1e-12 {
		return v
	}
	return v.MulScalar(1 / l)
}

// Neg 取反。
func (v Vec3) Neg() Vec3 { return Vec3{-v.X, -v.Y, -v.Z} }

// Lerp 线性插值。
func (v Vec3) Lerp(o Vec3, t float32) Vec3 {
	return Vec3{v.X + (o.X-v.X)*t, v.Y + (o.Y-v.Y)*t, v.Z + (o.Z-v.Z)*t}
}

// Mat4 是列主序 4x4 矩阵。
// 下标 [col*4+row]，即 m[0..3] 是第 0 列（X 轴），m[12..15] 是平移列。
type Mat4 [16]float32

// Identity 返回单位矩阵。
func Identity() Mat4 {
	return Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// Mul 矩阵乘法 a*b（先应用 b 再应用 a）。
func Mul(a, b Mat4) Mat4 {
	var m Mat4
	for c := 0; c < 4; c++ {
		for r := 0; r < 4; r++ {
			var s float32
			for k := 0; k < 4; k++ {
				s += a[k*4+r] * b[c*4+k]
			}
			m[c*4+r] = s
		}
	}
	return m
}

// Transform 变换向量（v 视为 (x,y,z,1)，忽略平移分量的投影除法）。
func (m Mat4) Transform(v Vec3) Vec3 {
	x := m[0]*v.X + m[4]*v.Y + m[8]*v.Z + m[12]
	y := m[1]*v.X + m[5]*v.Y + m[9]*v.Z + m[13]
	z := m[2]*v.X + m[6]*v.Y + m[10]*v.Z + m[14]
	return Vec3{x, y, z}
}

// TransformW 返回齐次变换的 (x,y,z,w)，供透视除法使用。
func (m Mat4) TransformW(v Vec3) (x, y, z, w float32) {
	x = m[0]*v.X + m[4]*v.Y + m[8]*v.Z + m[12]
	y = m[1]*v.X + m[5]*v.Y + m[9]*v.Z + m[13]
	z = m[2]*v.X + m[6]*v.Y + m[10]*v.Z + m[14]
	w = m[3]*v.X + m[7]*v.Y + m[11]*v.Z + m[15]
	return
}

// Mat3 返回左上 3x3（旋转/缩放部分，用于法线变换）。
func (m Mat4) Mat3() Mat4 {
	return Mat4{
		m[0], m[1], m[2], 0,
		m[4], m[5], m[6], 0,
		m[8], m[9], m[10], 0,
		0, 0, 0, 1,
	}
}

// Translate 平移矩阵。
func Translate(x, y, z float32) Mat4 {
	m := Identity()
	m[12], m[13], m[14] = x, y, z
	return m
}

// Scale 均匀缩放矩阵。
func Scale(s float32) Mat4 {
	return Mat4{
		s, 0, 0, 0,
		0, s, 0, 0,
		0, 0, s, 0,
		0, 0, 0, 1,
	}
}

// RotateX 绕 X 轴旋转（弧度）。
func RotateX(a float32) Mat4 {
	c, s := float32(math.Cos(float64(a))), float32(math.Sin(float64(a)))
	return Mat4{
		1, 0, 0, 0,
		0, c, s, 0,
		0, -s, c, 0,
		0, 0, 0, 1,
	}
}

// RotateY 绕 Y 轴旋转（弧度）。
func RotateY(a float32) Mat4 {
	c, s := float32(math.Cos(float64(a))), float32(math.Sin(float64(a)))
	return Mat4{
		c, 0, -s, 0,
		0, 1, 0, 0,
		s, 0, c, 0,
		0, 0, 0, 1,
	}
}

// RotateZ 绕 Z 轴旋转（弧度）。
func RotateZ(a float32) Mat4 {
	c, s := float32(math.Cos(float64(a))), float32(math.Sin(float64(a)))
	return Mat4{
		c, s, 0, 0,
		-s, c, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// Perspective 构建透视投影矩阵。
// fovY 是垂直视场角（弧度），aspect = W/H，near/far 为近远裁剪距离（正数）。
func Perspective(fovY, aspect, near, far float32) Mat4 {
	f := float32(1.0 / math.Tan(float64(fovY)/2))
	nf := 1 / (near - far)
	return Mat4{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) * nf, -1,
		0, 0, 2 * far * near * nf, 0,
	}
}

// Ortho 构建正交投影矩阵（OpenGL 约定：near 平面 z=-n → NDC -1，far → +1）。
// l/r/b/t 为左右下上平面，n/f 为近远距离（正数，相机看向 -Z）。
func Ortho(l, r, b, t, n, f float32) Mat4 {
	return Mat4{
		2 / (r - l), 0, 0, 0,
		0, 2 / (t - b), 0, 0,
		0, 0, -2 / (f - n), 0,
		-(r + l) / (r - l), -(t + b) / (t - b), -(f + n) / (f - n), 1,
	}
}

// LookAt 构建视图矩阵：相机在 eye，看向 center，up 为上方向。
func LookAt(eye, center, up Vec3) Mat4 {
	f := center.Sub(eye).Normalized()
	s := f.Cross(up).Normalized()
	u := s.Cross(f)
	return Mat4{
		s.X, u.X, -f.X, 0,
		s.Y, u.Y, -f.Y, 0,
		s.Z, u.Z, -f.Z, 0,
		-s.Dot(eye), -u.Dot(eye), f.Dot(eye), 1,
	}
}

// FPSView 由位置/偏航/俯仰构建视图矩阵（第一人称）。
// yaw 绕 Y 轴（水平旋转），pitch 绕 X 轴（俯仰），弧度。
// 约定：yaw=0 时相机看向 -Z（与 Perspective 一致），yaw 顺时针为正。
func FPSView(pos Vec3, yaw, pitch float32) Mat4 {
	cp := float32(math.Cos(float64(pitch)))
	sp := float32(math.Sin(float64(pitch)))
	cy := float32(math.Cos(float64(yaw)))
	sy := float32(math.Sin(float64(yaw)))
	// 前向向量（未归一化的问题交给 LookAt，直接构造正交基）
	f := Vec3{cp * sy, sp, -cp * cy}.Normalized()
	up := Vec3{0, 1, 0}
	return LookAt(pos, pos.Add(f), up)
}

// Forward 返回 yaw/pitch 相机的前向单位向量（yaw=0 → -Z）。
func Forward(yaw, pitch float32) Vec3 {
	cp := float32(math.Cos(float64(pitch)))
	return Vec3{cp * float32(math.Sin(float64(yaw))), float32(math.Sin(float64(pitch))), -cp * float32(math.Cos(float64(yaw)))}
}

// Invert 求 4x4 矩阵的逆（高斯-约当消元，对仿射/透视矩阵均正确）。
// 不可逆时返回零矩阵并 ok=false。
func (m Mat4) Invert() (Mat4, bool) {
	// 增广矩阵 [m | I]
	a := m
	var inv Mat4
	for i := 0; i < 4; i++ {
		inv[i*4+i] = 1
	}
	for col := 0; col < 4; col++ {
		// 找主元
		pivot := col
		for row := col + 1; row < 4; row++ {
			if float32(math.Abs(float64(a[row*4+col]))) > float32(math.Abs(float64(a[pivot*4+col]))) {
				pivot = row
			}
		}
		if float32(math.Abs(float64(a[pivot*4+col]))) < 1e-12 {
			return Mat4{}, false
		}
		if pivot != col {
			for k := 0; k < 4; k++ {
				a[pivot*4+k], a[col*4+k] = a[col*4+k], a[pivot*4+k]
				inv[pivot*4+k], inv[col*4+k] = inv[col*4+k], inv[pivot*4+k]
			}
		}
		pv := a[col*4+col]
		for k := 0; k < 4; k++ {
			a[col*4+k] /= pv
			inv[col*4+k] /= pv
		}
		for row := 0; row < 4; row++ {
			if row == col {
				continue
			}
			f := a[row*4+col]
			if f == 0 {
				continue
			}
			for k := 0; k < 4; k++ {
				a[row*4+k] -= f * a[col*4+k]
				inv[row*4+k] -= f * inv[col*4+k]
			}
		}
	}
	return inv, true
}

// Unproject 把 NDC 坐标（-1..1）反投影回世界空间。
// ndcX/ndcY/ndcZ 用 viewproj 的逆矩阵；返回世界坐标。
func (m Mat4) Unproject(ndcX, ndcY, ndcZ float32) (Vec3, bool) {
	inv, ok := m.Invert()
	if !ok {
		return Vec3{}, false
	}
	// 齐次：x' = M⁻¹ * (x,y,z,1)
	x := inv[0]*ndcX + inv[4]*ndcY + inv[8]*ndcZ + inv[12]
	y := inv[1]*ndcX + inv[5]*ndcY + inv[9]*ndcZ + inv[13]
	z := inv[2]*ndcX + inv[6]*ndcY + inv[10]*ndcZ + inv[14]
	w := inv[3]*ndcX + inv[7]*ndcY + inv[11]*ndcZ + inv[15]
	if float32(math.Abs(float64(w))) < 1e-12 {
		return Vec3{}, false
	}
	return Vec3{x / w, y / w, z / w}, true
}
