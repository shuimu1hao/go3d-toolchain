package app

import (
	"go3d/math3d"
	"go3d/mesh"
	"go3d/render"
	"go3deditor/app"
)

// ModelRes 模型资源（从建模编辑器加载：网格/骨骼/动画/权重）。
type ModelRes struct {
	Name     string
	Mesh     *mesh.Mesh
	Skeleton *app.Skeleton
	Weights  []app.SkinWeight
	Anim     *app.Animation
	Color    mesh.Color
}

// ModelResFromObject 从建模编辑器对象构造资源。
func ModelResFromObject(o *app.Object) *ModelRes {
	return &ModelRes{
		Name:     o.Name,
		Mesh:     o.RenderMesh(),
		Skeleton: o.Skeleton,
		Weights:  o.Weights,
		Anim:     o.Anim,
		Color:    o.Color,
	}
}

// Instance 场景实例（引用资源 + 独立变换 + 动画状态）。
type Instance struct {
	Name    string
	Res     *ModelRes
	Pos     math3d.Vec3
	RotX    float32
	RotY    float32
	RotZ    float32
	Scale   float32
	Visible bool

	IsPlayer bool // 运行模式玩家控制
	IsSprite bool // 贴图精灵（公告板）
	Sprite   *Sprite
	SpriteName string

	// 动画状态（骨骼独立副本，避免实例间共享姿势）
	Skel        *app.Skeleton
	AnimTime    float32
	AnimPlaying bool

	// 运行状态
	Vel math3d.Vec3
}

// GetSpriteName 精灵名称。
func (i *Instance) GetSpriteName() string {
	if i.Sprite != nil {
		return i.Sprite.Name
	}
	return i.SpriteName
}

// NewInstance 创建实例（深拷贝骨骼）。
func NewInstance(name string, res *ModelRes) *Instance {
	inst := &Instance{
		Name:    name,
		Res:     res,
		Scale:   1,
		Visible: true,
	}
	if res.Skeleton != nil {
		inst.Skel = res.Skeleton.Clone()
	}
	return inst
}

// HasAnim 是否有动画可播。
func (i *Instance) HasAnim() bool {
	return i.Res != nil && i.Res.Anim != nil && i.Skel != nil && len(i.Skel.Bones) > 0
}

// UpdateAnim 推进动画（骨骼驱动）。
func (i *Instance) UpdateAnim(dt float64) {
	if !i.HasAnim() || !i.AnimPlaying {
		return
	}
	i.AnimTime += float32(dt)
	a := i.Res.Anim
	if a.Duration > 0 && i.AnimTime >= a.Duration {
		if a.Loop {
			i.AnimTime -= a.Duration
		} else {
			i.AnimTime = a.Duration
			i.AnimPlaying = false
		}
	}
	a.ApplyToSkeleton(i.Skel, i.AnimTime)
}

// RenderObj 转换渲染对象（含蒙皮）。
func (i *Instance) RenderObj() render.Object {
	m := i.Res.Mesh
	if i.Skel != nil && len(i.Res.Weights) == len(m.Positions) && len(i.Res.Weights) > 0 {
		m = i.Skel.SkinMesh(m, i.Res.Weights)
	}
	return render.Object{
		Mesh:      m,
		Pos:       i.Pos,
		RotX:      i.RotX,
		RotY:      i.RotY,
		RotZ:      i.RotZ,
		Scale:     i.Scale,
		ColorTint: &i.Res.Color,
	}
}

// Level 关卡：模型资源 + 贴图素材 + 实例列表。
type Level struct {
	Name    string
	Models  []*ModelRes
	Sprites []*Sprite
	Insts   []*Instance
}

// NewLevel 创建空关卡。
func NewLevel(name string) *Level {
	return &Level{Name: name}
}

// AddModel 添加模型资源（去重）。
func (l *Level) AddModel(res *ModelRes) {
	for _, m := range l.Models {
		if m.Name == res.Name {
			return
		}
	}
	l.Models = append(l.Models, res)
}

// AddSprite 添加贴图素材（去重）。
func (l *Level) AddSprite(s *Sprite) {
	for _, sp := range l.Sprites {
		if sp.Name == s.Name {
			return
		}
	}
	l.Sprites = append(l.Sprites, s)
}

// AddInstance 添加实例。
func (l *Level) AddInstance(inst *Instance) {
	l.Insts = append(l.Insts, inst)
}

// FindModel 按名查找资源。
func (l *Level) FindModel(name string) *ModelRes {
	for _, m := range l.Models {
		if m.Name == name {
			return m
		}
	}
	return nil
}

// FindSprite 按名查找素材。
func (l *Level) FindSprite(name string) *Sprite {
	for _, s := range l.Sprites {
		if s.Name == name {
			return s
		}
	}
	return nil
}
