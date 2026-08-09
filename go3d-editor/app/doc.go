// Package app 实现 go3d-editor：SolidWorks 风格模型树 + Blender 风格视口操作的 3D 建模编辑器。
package app

import (
	"fmt"

	"go3d/math3d"
	"go3d/mesh"
	"go3d/render"
)

// ObjType 是基本体类型（模型树特征类型）。
type ObjType int

const (
	TCube ObjType = iota
	TSphere
	TCylinder
	TCone
	TTorus
	TPlane
)

// Name 返回类型中文名。
func (t ObjType) Name() string {
	switch t {
	case TCube:
		return "立方体"
	case TSphere:
		return "球体"
	case TCylinder:
		return "圆柱"
	case TCone:
		return "圆锥"
	case TTorus:
		return "圆环"
	case TPlane:
		return "平面"
	}
	return "?"
}

// ShortName 返回树节点英文/短标签（配合图标）。
func (t ObjType) ShortName() string {
	switch t {
	case TCube:
		return "Cube"
	case TSphere:
		return "Sphere"
	case TCylinder:
		return "Cyl"
	case TCone:
		return "Cone"
	case TTorus:
		return "Torus"
	case TPlane:
		return "Plane"
	}
	return "?"
}

// NewMesh 生成该类型的默认网格（世界坐标原点居中）。
func (t ObjType) NewMesh() *mesh.Mesh {
	switch t {
	case TCube:
		return mesh.Cube(1.0)
	case TSphere:
		return mesh.Sphere(0.6, 24, 12)
	case TCylinder:
		return mesh.Cylinder(0.5, 1.0, 24)
	case TCone:
		return mesh.Cone(0.5, 1.0, 24)
	case TTorus:
		return mesh.Torus(0.7, 0.28, 24, 12)
	case TPlane:
		return mesh.Grid(0, 2.0, 1) // 单格平面
	}
	return mesh.Cube(1.0)
}

// DefaultColor 返回类型默认颜色。
func (t ObjType) DefaultColor() mesh.Color {
	switch t {
	case TCube:
		return mesh.Col(90, 150, 220)
	case TSphere:
		return mesh.Col(220, 120, 150)
	case TCylinder:
		return mesh.Col(120, 180, 110)
	case TCone:
		return mesh.Col(230, 150, 70)
	case TTorus:
		return mesh.Col(170, 120, 220)
	case TPlane:
		return mesh.Col(150, 150, 160)
	}
	return mesh.Col(200, 200, 200)
}

// Object 是场景中的一个特征（模型树叶子节点）。
type Object struct {
	Name    string
	Type    ObjType
	Pos     math3d.Vec3
	RotX    float32 // 弧度
	RotY    float32
	RotZ    float32
	Scale   float32
	Color   mesh.Color
	Visible bool

	// CustomMesh 非 nil 时替代 Type.NewMesh（草图拉伸/CSG 结果/导入模型）
	CustomMesh *mesh.Mesh

	// 骨骼动画扩展
	Skeleton    *Skeleton
	Weights     []SkinWeight // 顶点绑定权重（对应渲染网格顶点）
	Anim        *Animation
	AnimTime    float32
	AnimPlaying bool
}

// RenderMesh 返回实际渲染的网格。
func (o *Object) RenderMesh() *mesh.Mesh {
	if o.CustomMesh != nil {
		return o.CustomMesh
	}
	return o.Type.NewMesh()
}

// ScaleVal 返回有效缩放（0 按 1 处理，与渲染器一致）。
func (o *Object) ScaleVal() float32 {
	if o.Scale == 0 {
		return 1
	}
	return o.Scale
}

// TransformPoint 把网格局部坐标变换到世界坐标（与渲染器模型矩阵一致）。
func (o *Object) TransformPoint(p math3d.Vec3) math3d.Vec3 {
	m := math3d.Mul(math3d.Mul(math3d.Mul(math3d.Mul(math3d.Translate(o.Pos.X, o.Pos.Y, o.Pos.Z),
		math3d.RotateZ(o.RotZ)), math3d.RotateY(o.RotY)), math3d.RotateX(o.RotX)), math3d.Scale(o.ScaleVal()))
	return m.Transform(p)
}

// AddObjFromMesh 添加自定义网格对象（草图拉伸/CSG/OBJ 结果）。
func (d *Document) AddObjFromMesh(name string, m *mesh.Mesh) *Object {
	o := &Object{
		Name:       name,
		Type:       TCube, // 类型标签，网格用 CustomMesh
		CustomMesh: m,
		Pos:        math3d.Vec3{0, 0, 0},
		Scale:      1,
		Color:      mesh.Col(200, 170, 120),
		Visible:    true,
	}
	d.Objs = append(d.Objs, o)
	return o
}

// HasAnim 是否有可播放动画。
func (o *Object) HasAnim() bool { return o.Anim != nil && o.Skeleton != nil && len(o.Skeleton.Bones) > 0 }

// UpdateAnim 推进动画时间。
func (o *Object) UpdateAnim(dt float64) {
	if !o.HasAnim() || !o.AnimPlaying {
		return
	}
	o.AnimTime += float32(dt)
	if o.Anim.Duration > 0 && o.AnimTime >= o.Anim.Duration {
		if o.Anim.Loop {
			o.AnimTime -= o.Anim.Duration
		} else {
			o.AnimTime = o.Anim.Duration
			o.AnimPlaying = false
		}
	}
	o.Anim.ApplyToSkeleton(o.Skeleton, o.AnimTime)
}

// RenderObj 转换为渲染器对象（含蒙皮变形）。
func (o *Object) RenderObj() render.Object {
	m := o.RenderMesh()
	if o.Skeleton != nil && len(o.Weights) == len(m.Positions) && len(o.Weights) > 0 {
		m = o.Skeleton.SkinMesh(m, o.Weights)
	}
	return render.Object{
		Mesh:      m,
		Pos:       o.Pos,
		RotX:      o.RotX,
		RotY:      o.RotY,
		RotZ:      o.RotZ,
		Scale:     o.Scale,
		ColorTint: &o.Color,
	}
}

// Document 是编辑器文档：特征列表按模型树顺序排列（树顺序=数组顺序）。
type Document struct {
	Name string
	Objs []*Object
}

// NewDocument 创建空文档。
func NewDocument(name string) *Document {
	return &Document{Name: name, Objs: []*Object{}}
}

// Add 添加特征，返回新对象。
func (d *Document) Add(t ObjType) *Object {
	o := &Object{
		Name:    fmt.Sprintf("%s%d", t.ShortName(), len(d.Objs)+1),
		Type:    t,
		Pos:     math3d.Vec3{0, 0.5, 0},
		Scale:   1,
		Color:   t.DefaultColor(),
		Visible: true,
	}
	d.Objs = append(d.Objs, o)
	return o
}

// Remove 按索引删除特征。
func (d *Document) Remove(idx int) {
	if idx < 0 || idx >= len(d.Objs) {
		return
	}
	d.Objs = append(d.Objs[:idx], d.Objs[idx+1:]...)
}

// MoveUp 上移特征（树中更靠前）。
func (d *Document) MoveUp(idx int) {
	if idx <= 0 || idx >= len(d.Objs) {
		return
	}
	d.Objs[idx-1], d.Objs[idx] = d.Objs[idx], d.Objs[idx-1]
}

// MoveDown 下移特征。
func (d *Document) MoveDown(idx int) {
	if idx < 0 || idx >= len(d.Objs)-1 {
		return
	}
	d.Objs[idx], d.Objs[idx+1] = d.Objs[idx+1], d.Objs[idx]
}

// Duplicate 复制特征（名称加副本）。
func (d *Document) Duplicate(idx int) *Object {
	if idx < 0 || idx >= len(d.Objs) {
		return nil
	}
	src := d.Objs[idx]
	cp := *src
	cp.Name = src.Name + "_copy"
	cp.Pos = src.Pos.Add(math3d.Vec3{0.3, 0, 0.3})
	// 插入到原特征后面
	rest := make([]*Object, 0, len(d.Objs)+1)
	rest = append(rest, d.Objs[:idx+1]...)
	rest = append(rest, &cp)
	rest = append(rest, d.Objs[idx+1:]...)
	d.Objs = rest
	return &cp
}

// VisibleObjs 返回可见特征（渲染用）。
func (d *Document) VisibleObjs() []*Object {
	var out []*Object
	for _, o := range d.Objs {
		if o.Visible {
			out = append(out, o)
		}
	}
	return out
}
