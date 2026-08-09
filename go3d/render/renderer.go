// Package render 是纯 Go 软件 3D 渲染器（软渲染管线）。
//
// 管线：模型变换 → 视图变换 → near 平面裁剪 → 透视除法 → 视口变换 →
// 背面剔除 → 方向光（flat shading）→ 重心坐标光栅化 + z-buffer。
//
// 输出直接写入 32bpp B,G,R,pad 像素缓冲（与 go2dgame/engine.Canvas.Pixels 布局一致），
// 因此无需任何图形 API 即可上屏，也方便无头离屏渲染测试。
package render

import (
	"go3d/math3d"
	"go3d/mesh"
)

// Light 描述场景光照：一个方向光 + 环境光。
type Light struct {
	// Dir 是“指向光源”的方向（世界空间单位向量）。
	Dir math3d.Vec3
	// Ambient 环境光强度 [0,1]，叠加在所有面上。
	Ambient float32
}

// DefaultLight 返回默认光照：从右上方打过来的方向光。
func DefaultLight() Light {
	return Light{
		Dir:     math3d.Vec3{0.5, 0.8, 0.6}.Normalized(),
		Ambient: 0.25,
	}
}

// Camera 是相机（透视/正交均可）。
type Camera struct {
	Pos        math3d.Vec3
	Yaw, Pitch float32 // 弧度；Yaw 绕 Y 轴水平旋转，Pitch 绕 X 轴俯仰
	FOV        float32 // 垂直视场角（弧度，透视模式）
	Near, Far  float32 // 近/远裁剪距离（正数）
	AutoOrbitYaw float32 // >=0 时每帧叠加的自动环绕角速度（弧度/秒）

	// LookTarget 非 nil 时改用 LookAt(pos→target) 构建视图（orbit 相机）。
	LookTarget *math3d.Vec3
	// Ortho 为 true 时使用正交投影；OrthoSize 为视口高度对应的世界单位。
	Ortho     bool
	OrthoSize float32
}

// DefaultCamera 返回默认相机：位于 +Z 侧，看向原点方向（yaw=0 前向 -Z）。
func DefaultCamera() *Camera {
	return &Camera{
		Pos:          math3d.Vec3{0, 2.2, 7},
		FOV:          60 * Pi / 180,
		Near:         0.1,
		Far:          200,
		AutoOrbitYaw: 0,
	}
}

// Pi 常量转发（math3d 没有 Pi，直接在此定义使用）。
const Pi = 3.14159265358979323846

// Object 是场景中的网格实例。
type Object struct {
	Mesh            *mesh.Mesh
	Pos             math3d.Vec3
	RotX, RotY, RotZ float32 // 弧度
	Scale           float32
	// WireColor 非 nil 时强制线框颜色（Wireframe 模式下）。
	WireColor *mesh.Color
	// ColorOverride 非 nil 时所有面使用此颜色（忽略面颜色；拾取用，无光照）。
	ColorOverride *mesh.Color
	// ColorTint 非 nil 时所有面使用此颜色但保留光照（物体自定义颜色）。
	ColorTint *mesh.Color
}

// scaleVal 返回有效缩放值：0（未设置）按 1 处理。
func (o *Object) scaleVal() float32 {
	if o.Scale == 0 {
		return 1
	}
	return o.Scale
}

// Renderer 是软渲染器。
type Renderer struct {
	W, H      int
	zbuf      []float32
	Light     Light
	Wireframe bool
	// ClearColor 清屏背景色（RGB；像素缓冲为 BGR 布局，Clear 时反转）。
	// 默认深空蓝黑 (18,22,30)，主题切换时由编辑器设置。
	ClearColor mesh.Color
}

// NewRenderer 创建渲染器（W,H 为像素缓冲尺寸）。
func NewRenderer(w, h int) *Renderer {
	return &Renderer{
		W:          w,
		H:          h,
		zbuf:       make([]float32, w*h),
		Light:      DefaultLight(),
		ClearColor: mesh.Col(18, 22, 30),
	}
}

// Clear 清空像素缓冲为背景色（默认深空蓝黑，主题可改）。
func (r *Renderer) Clear(pixels []byte) {
	for i := 0; i < len(pixels); i += 4 {
		pixels[i] = r.ClearColor.B
		pixels[i+1] = r.ClearColor.G
		pixels[i+2] = r.ClearColor.R
		pixels[i+3] = 0
	}
	for i := range r.zbuf {
		r.zbuf[i] = -1e9 // 1/z_view：远=趋近0(负小)，初始=最远
	}
}

// clipPoint 是 near 裁剪使用的临时点。
type clipPoint struct {
	pos math3d.Vec3 // 视图空间位置
	z   float32     // 视图空间 z（相机前方为负）
}

// Render 渲染一帧到像素缓冲（B,G,R,pad 布局，len(pixels) >= W*H*4）。
// 返回绘制的三角形数（用于调试/统计）。
func (r *Renderer) Render(pixels []byte, cam *Camera, objs []Object) int {
	aspect := float32(r.W) / float32(r.H)
	var proj math3d.Mat4
	if cam.Ortho {
		size := cam.OrthoSize
		if size <= 0 {
			size = 8
		}
		proj = math3d.Ortho(-size*aspect/2, size*aspect/2, -size/2, size/2, cam.Near, cam.Far)
	} else {
		proj = math3d.Perspective(cam.FOV, aspect, cam.Near, cam.Far)
	}
	var view math3d.Mat4
	if cam.LookTarget != nil {
		view = math3d.LookAt(cam.Pos, *cam.LookTarget, math3d.Vec3{0, 1, 0})
	} else {
		view = math3d.FPSView(cam.Pos, cam.Yaw, cam.Pitch)
	}

	triangles := 0
	for oi := range objs {
		obj := &objs[oi]
		m := obj.Mesh
		if m == nil {
			continue
		}
		model := math3d.Mul(math3d.Mul(math3d.Mul(math3d.Mul(math3d.Translate(obj.Pos.X, obj.Pos.Y, obj.Pos.Z),
			math3d.RotateZ(obj.RotZ)), math3d.RotateY(obj.RotY)), math3d.RotateX(obj.RotX)), math3d.Scale(obj.scaleVal()))
		vm := math3d.Mul(view, model)
		// 世界空间法线变换（model 3x3；均匀缩放不影响方向）
		modelN := model.Mat3()

		for fi := range m.Faces {
			f := &m.Faces[fi]
			// 视图空间：near 裁剪在透视除法前做
			pts := make([]clipPoint, 0, 4)
			for _, vi := range []int{f.A, f.B, f.C} {
				p := vm.Transform(m.Positions[vi])
				pts = append(pts, clipPoint{pos: p, z: p.Z})
			}
			pts = clipNear(pts, cam.Near)
			if len(pts) < 3 {
				continue
			}
			// 背面剔除：视图空间面法线 vs 视线方向（相机在原点，看向 -Z）
			nv := pts[1].pos.Sub(pts[0].pos).Cross(pts[2].pos.Sub(pts[0].pos))
			// 视线 = 面中心 → 相机
			center := pts[0].pos.Add(pts[1].pos).Add(pts[2].pos).MulScalar(1.0 / 3.0)
			if nv.Dot(center) >= 0 { // 法线背离相机
				continue
			}

			// 光照：世界空间面法线；ColorOverride 时跳过光照（纯色，拾取用）
			var cr, cg, cb uint8
			if obj.ColorOverride != nil {
				cr, cg, cb = obj.ColorOverride.R, obj.ColorOverride.G, obj.ColorOverride.B
			} else {
				nw := modelN.Transform(mesh.FaceNormal(m, f)).Normalized()
				diff := nw.Dot(r.Light.Dir)
				if diff < 0 {
					diff = 0
				}
				bright := r.Light.Ambient + (1-r.Light.Ambient)*diff
				col := f.Col
				if obj.ColorTint != nil {
					col = *obj.ColorTint
				}
				cr = uint8(float32(col.R) * bright)
				cg = uint8(float32(col.G) * bright)
				cb = uint8(float32(col.B) * bright)
			}

			// 投影 + 视口
			var sx, sy, sz, sw, svz []float32
			for i := range pts {
				x, y, z, w := proj.TransformW(pts[i].pos)
				if w <= 1e-9 {
					continue
				}
				ndcX := x / w
				ndcY := y / w
				ndcZ := z / w
				if ndcZ < -1-1e-4 || ndcZ > 1+1e-4 {
					continue
				}
				sx = append(sx, (ndcX+1)*0.5*float32(r.W))
				sy = append(sy, (1-(ndcY+1)*0.5)*float32(r.H))
				sz = append(sz, ndcZ)
				sw = append(sw, w) // 视深（透视校正权重）
				svz = append(svz, -w) // 视图 z（负）
			}
			if len(sx) < 3 {
				continue
			}
			raster := rasterTri{
				x: sx, y: sy, w: sw, zv: svz,
				cr: cr, cg: cg, cb: cb,
			}
			if r.Wireframe || obj.WireColor != nil {
				if obj.WireColor != nil {
					raster.cr, raster.cg, raster.cb = obj.WireColor.R, obj.WireColor.G, obj.WireColor.B
				}
				r.drawWire(pixels, raster)
			} else {
				r.drawFilled(pixels, raster)
			}
			triangles++
		}
	}
	return triangles
}

// clipNear 用 Sutherland-Hodgman 对 z < near 平面裁剪（视图空间，z 为负值朝前）。
func clipNear(pts []clipPoint, near float32) []clipPoint {
	if len(pts) < 3 {
		return pts
	}
	out := make([]clipPoint, 0, len(pts)+1)
	zNear := -near // 相机前方 z = -near
	for i := 0; i < len(pts); i++ {
		cur := pts[i]
		next := pts[(i+1)%len(pts)]
		curIn := cur.z <= zNear
		nextIn := next.z <= zNear
		if curIn {
			out = append(out, cur)
		}
		if curIn != nextIn {
			// 求交点
			t := (zNear - cur.z) / (next.z - cur.z)
			ip := cur.pos.Lerp(next.pos, t)
			out = append(out, clipPoint{pos: ip, z: zNear})
		}
	}
	return out
}

// rasterTri 是待光栅化的屏幕空间三角形（z 为 NDC 深度）。
type rasterTri struct {
	x, y          []float32
	w             []float32 // 视深（-z_eye，透视校正权重）
	zv            []float32 // 视图 z（负值，近=更负）
	cr, cg, cb    uint8
}

// drawFilled 重心坐标填充光栅化 + z-buffer。
func (r *Renderer) drawFilled(pixels []byte, t rasterTri) {
	x0 := r.W - 1
	x1 := 0
	y0 := r.H - 1
	y1 := 0
	for i := 0; i < 3; i++ {
		xi := int(t.x[i])
		yi := int(t.y[i])
		if xi < x0 {
			x0 = xi
		}
		if xi > x1 {
			x1 = xi
		}
		if yi < y0 {
			y0 = yi
		}
		if yi > y1 {
			y1 = yi
		}
	}
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 >= r.W {
		x1 = r.W - 1
	}
	if y1 >= r.H {
		y1 = r.H - 1
	}
	if x0 > x1 || y0 > y1 {
		return
	}

	ax, ay := t.x[0], t.y[0]
	bx, by := t.x[1], t.y[1]
	cx, cy := t.x[2], t.y[2]
	za, zb, zc := 1/t.zv[0], 1/t.zv[1], 1/t.zv[2] // 1/z_view（屏幕线性，透视正确；近=绝对值小=大）
	area := edge(bx-ax, by-ay, cx-ax, cy-ay) // 2*有符号面积
	if area == 0 {
		return
	}
	if area < 0 {
		// 环绕顺序为顺时针：交换 b/c 使 area 恒正，重心坐标统一非负
		bx, cx = cx, bx
		by, cy = cy, by
		zb, zc = zc, zb
		area = -area
	}
	invArea := 1 / area
	for yy := y0; yy <= y1; yy++ {
		base := yy * r.W * 4
		for xx := x0; xx <= x1; xx++ {
			// 重心坐标（带符号面积比）
			w0 := edge(bx-ax, by-ay, float32(xx)-ax, float32(yy)-ay) * invArea
			w1 := edge(cx-bx, cy-by, float32(xx)-bx, float32(yy)-by) * invArea
			w2 := edge(ax-cx, ay-cy, float32(xx)-cx, float32(yy)-cy) * invArea
			if w0 < 0 || w1 < 0 || w2 < 0 {
				continue
			}
			z := w0*za + w1*zb + w2*zc // 1/z_view 屏幕线性插值
			di := base + xx*4
			if z > r.zbuf[yy*r.W+xx] { // 1/z 大 = 近
				r.zbuf[yy*r.W+xx] = z
				pixels[di] = t.cb
				pixels[di+1] = t.cg
				pixels[di+2] = t.cr
				pixels[di+3] = 0
			}
		}
	}
}

// drawWire 线框光栅化：仅当像素落在三角形边上时绘制。
// 判定：重心坐标任一分量小于阈值（等价于接近边）。
func (r *Renderer) drawWire(pixels []byte, t rasterTri) {
	// 包围盒
	x0 := r.W - 1
	x1 := 0
	y0 := r.H - 1
	y1 := 0
	for i := 0; i < 3; i++ {
		xi := int(t.x[i])
		yi := int(t.y[i])
		if xi < x0 {
			x0 = xi
		}
		if xi > x1 {
			x1 = xi
		}
		if yi < y0 {
			y0 = yi
		}
		if yi > y1 {
			y1 = yi
		}
	}
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 >= r.W {
		x1 = r.W - 1
	}
	if y1 >= r.H {
		y1 = r.H - 1
	}
	if x0 > x1 || y0 > y1 {
		return
	}
	ax, ay := t.x[0], t.y[0]
	bx, by := t.x[1], t.y[1]
	cx, cy := t.x[2], t.y[2]
	za, zb, zc := 1/t.zv[0], 1/t.zv[1], 1/t.zv[2]
	area := edge(bx-ax, by-ay, cx-ax, cy-ay)
	if area == 0 {
		return
	}
	if area < 0 {
		bx, cx = cx, bx
		by, cy = cy, by
		zb, zc = zc, zb
		area = -area
	}
	invArea := 1 / area
	// 线宽阈值：相对边长的 ~2%，同时保证至少 1 像素附近
	var eps float32 = 0.02
	if area < 4 {
		eps = 0.3 // 小三角形：放宽，保证有像素
	}
	for yy := y0; yy <= y1; yy++ {
		base := yy * r.W * 4
		for xx := x0; xx <= x1; xx++ {
			w0 := edge(bx-ax, by-ay, float32(xx)-ax, float32(yy)-ay) * invArea
			w1 := edge(cx-bx, cy-by, float32(xx)-bx, float32(yy)-by) * invArea
			w2 := edge(ax-cx, ay-cy, float32(xx)-cx, float32(yy)-cy) * invArea
			if w0 < 0 || w1 < 0 || w2 < 0 {
				continue
			}
			if w0 > eps && w1 > eps && w2 > eps {
				continue // 完全在内部
			}
			z := w0*za + w1*zb + w2*zc
			di := base + xx*4
			if z > r.zbuf[yy*r.W+xx] {
				r.zbuf[yy*r.W+xx] = z
				pixels[di] = t.cb
				pixels[di+1] = t.cg
				pixels[di+2] = t.cr
				pixels[di+3] = 0
			}
		}
	}
}

// edge 计算二维叉积 (p1-p0) x (p2-p0) 的 z 分量。
func edge(dx1, dy1, dx2, dy2 float32) float32 {
	return dx1*dy2 - dy1*dx2
}

// PickColor 把对象 id（>=1）编码为 RGB 颜色（0 保留给背景/无对象）。
func PickColor(id int) mesh.Color {
	return mesh.Color{R: uint8(id & 0xFF), G: uint8((id >> 8) & 0xFF), B: uint8((id >> 16) & 0xFF)}
}

// PickIDFromColor 解码拾取颜色 → 对象 id（0 = 背景）。
func PickIDFromColor(r, g, b uint8) int {
	return int(r) | int(g)<<8 | int(b)<<16
}

// RenderPick 把对象按 ID 颜色渲染到像素缓冲（拾取用，无光照）。
// 渲染后读 (x,y) 像素，用 PickIDFromColor 解码得到对象 id（1 起始，0=背景）。
func (r *Renderer) RenderPick(pixels []byte, cam *Camera, objs []Object) int {
	picks := make([]Object, len(objs))
	for i := range objs {
		picks[i] = objs[i]
		c := PickColor(i + 1)
		picks[i].ColorOverride = &c
	}
	return r.Render(pixels, cam, picks)
}
