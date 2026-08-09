package app

import (
	"math"

	"go3d/math3d"
	"go3d/mesh"
)

// Bone 是骨骼节点（局部变换相对父骨骼）。
type Bone struct {
	Name        string
	Parent      int // -1 = 根
	Pos         math3d.Vec3
	RotX, RotY  float32
	RotZ        float32
}

// Skeleton 是骨骼层级树。
type Skeleton struct {
	Bones     []*Bone
	BindPoses []Bone // 绑定姿势副本（蒙皮逆矩阵用绑定姿势计算）
}

// NewSkeleton 创建空骨骼。
func NewSkeleton() *Skeleton {
	return &Skeleton{Bones: []*Bone{}}
}

// AddBone 添加骨骼（parent=-1 为根）。返回索引。
func (s *Skeleton) AddBone(name string, parent int, pos math3d.Vec3) int {
	if parent >= len(s.Bones) {
		parent = -1
	}
	b := &Bone{Name: name, Parent: parent, Pos: pos}
	s.Bones = append(s.Bones, b)
	s.BindPoses = append(s.BindPoses, *b)
	return len(s.Bones) - 1
}

// Clone 深拷贝骨骼树（实例独立姿势）。
func (s *Skeleton) Clone() *Skeleton {
	ns := NewSkeleton()
	for i, b := range s.Bones {
		cp := *b
		ns.Bones = append(ns.Bones, &cp)
		ns.BindPoses = append(ns.BindPoses, s.BindPoses[i])
		_ = i
	}
	return ns
}

// ResetBindPose 把当前姿势保存为绑定姿势（编辑器"设定绑定姿势"）。
func (s *Skeleton) ResetBindPose() {
	s.BindPoses = s.BindPoses[:0]
	for _, b := range s.Bones {
		s.BindPoses = append(s.BindPoses, *b)
	}
}

// bindWorld 绑定姿势世界矩阵（用 BindPoses 级联）。
func (s *Skeleton) bindWorld(i int) math3d.Mat4 {
	if i < 0 || i >= len(s.Bones) {
		return math3d.Identity()
	}
	b := &s.BindPoses[i]
	t := math3d.Translate(b.Pos.X, b.Pos.Y, b.Pos.Z)
	r := math3d.Mul(math3d.RotateZ(b.RotZ), math3d.Mul(math3d.RotateY(b.RotY), math3d.RotateX(b.RotX)))
	local := math3d.Mul(t, r)
	if b.Parent >= 0 {
		return math3d.Mul(s.bindWorld(b.Parent), local)
	}
	return local
}

// BoneLocal 骨骼局部矩阵。
func (b *Bone) BoneLocal() math3d.Mat4 {
	t := math3d.Translate(b.Pos.X, b.Pos.Y, b.Pos.Z)
	r := math3d.Mul(math3d.RotateZ(b.RotZ), math3d.Mul(math3d.RotateY(b.RotY), math3d.RotateX(b.RotX)))
	return math3d.Mul(t, r)
}

// BoneWorld 骨骼世界矩阵（级联父链）。
func (s *Skeleton) BoneWorld(i int) math3d.Mat4 {
	if i < 0 || i >= len(s.Bones) {
		return math3d.Identity()
	}
	local := s.Bones[i].BoneLocal()
	if s.Bones[i].Parent >= 0 {
		return math3d.Mul(s.BoneWorld(s.Bones[i].Parent), local)
	}
	return local
}

// BoneWorldPos 骨骼世界位置（根位置，用于可视化）。
func (s *Skeleton) BoneWorldPos(i int) math3d.Vec3 {
	return s.BoneWorld(i).Transform(math3d.Vec3{0, 0, 0})
}

// InverseBind 逆绑定矩阵（绑定姿势世界矩阵的逆）。
func (s *Skeleton) InverseBind(i int) math3d.Mat4 {
	inv, _ := s.bindWorld(i).Invert()
	return inv
}

// ---------- 顶点绑定 ----------

// SkinWeight 顶点权重（最多 4 骨骼）。
type SkinWeight struct {
	Bones [4]int
	W     [4]float32
}

// SkinWeights 为网格顶点自动计算权重：按最近骨骼距离倒数加权。
// 每顶点选最近 4 根骨骼（maxDist>0 时忽略更远的骨骼）。
func (s *Skeleton) SkinWeights(m *mesh.Mesh, maxDist float32) []SkinWeight {
	if len(s.Bones) == 0 {
		return nil
	}
	// 骨骼世界位置（绑定姿势）
	bpos := make([]math3d.Vec3, len(s.Bones))
	for i := range s.Bones {
		bpos[i] = s.BoneWorldPos(i)
	}
	ws := make([]SkinWeight, len(m.Positions))
	for vi, p := range m.Positions {
		type bd struct {
			idx int
			d   float32
		}
		// 收集所有骨骼距离
		var all []bd
		for bi := range s.Bones {
			d := p.Sub(bpos[bi]).Length()
			if maxDist > 0 && d > maxDist {
				continue
			}
			all = append(all, bd{bi, d})
		}
		if len(all) == 0 {
			ws[vi].Bones[0] = 0
			ws[vi].W[0] = 1
			continue
		}
		// 按距离升序排序（简单插入排序）
		for i := 1; i < len(all); i++ {
			for j := i; j > 0 && all[j].d < all[j-1].d; j-- {
				all[j], all[j-1] = all[j-1], all[j]
			}
		}
		n := len(all)
		if n > 4 {
			n = 4
		}
		total := float32(0)
		for k := 0; k < n; k++ {
			if all[k].d < 1e-6 {
				// 精确命中骨骼：独占权重
				ws[vi].Bones[0] = all[k].idx
				ws[vi].W[0] = 1
				ws[vi].W[1], ws[vi].W[2], ws[vi].W[3] = 0, 0, 0
				total = -1
				break
			}
			inv := 1 / all[k].d
			ws[vi].Bones[k] = all[k].idx
			ws[vi].W[k] = inv
			total += inv
		}
		if total > 0 {
			for k := 0; k < n; k++ {
				ws[vi].W[k] /= total
			}
		}
	}
	return ws
}

// SkinMesh 应用蒙皮：按骨骼当前世界矩阵与权重变换所有顶点，返回新 mesh。
func (s *Skeleton) SkinMesh(m *mesh.Mesh, ws []SkinWeight) *mesh.Mesh {
	if len(ws) != len(m.Positions) || len(s.Bones) == 0 {
		return m // 无绑定：原样
	}
	out := &mesh.Mesh{
		Positions: make([]math3d.Vec3, len(m.Positions)),
		Faces:     m.Faces,
	}
	// 预计算世界矩阵和逆绑定
	worlds := make([]math3d.Mat4, len(s.Bones))
	invBinds := make([]math3d.Mat4, len(s.Bones))
	for i := range s.Bones {
		worlds[i] = s.BoneWorld(i)
		invBinds[i] = s.InverseBind(i)
	}
	for vi, p := range m.Positions {
		var acc math3d.Vec3
		w := &ws[vi]
		for k := 0; k < 4; k++ {
			if w.W[k] <= 0 {
				continue
			}
			bi := w.Bones[k]
			if bi < 0 || bi >= len(s.Bones) {
				continue
			}
			// v' = W * B⁻¹ * v
			skin := math3d.Mul(worlds[bi], invBinds[bi])
			acc = acc.Add(skin.Transform(p).MulScalar(w.W[k]))
		}
		out.Positions[vi] = acc
	}
	return out
}

// ---------- 动画 ----------

// Keyframe 关键帧（骨骼局部变换）。
type Keyframe struct {
	Time        float32
	Pos         math3d.Vec3
	RotX, RotY  float32
	RotZ        float32
}

// AnimTrack 单骨骼动画轨道。
type AnimTrack struct {
	BoneIdx int
	Keys    []Keyframe
}

// Animation 骨骼动画。
type Animation struct {
	Name     string
	Duration float32
	Tracks   []AnimTrack
	Loop     bool
}

// AddKey 向轨道添加关键帧（按时间排序插入）。
func (a *Animation) AddKey(boneIdx int, t float32, pos math3d.Vec3, rx, ry, rz float32) {
	if t > a.Duration {
		a.Duration = t
	}
	for ti := range a.Tracks {
		if a.Tracks[ti].BoneIdx == boneIdx {
			tr := &a.Tracks[ti]
			insertAt := len(tr.Keys)
			for i := range tr.Keys {
				if tr.Keys[i].Time > t {
					insertAt = i
					break
				}
			}
			tr.Keys = append(tr.Keys, Keyframe{})
			copy(tr.Keys[insertAt+1:], tr.Keys[insertAt:])
			tr.Keys[insertAt] = Keyframe{Time: t, Pos: pos, RotX: rx, RotY: ry, RotZ: rz}
			return
		}
	}
	a.Tracks = append(a.Tracks, AnimTrack{BoneIdx: boneIdx, Keys: []Keyframe{{Time: t, Pos: pos, RotX: rx, RotY: ry, RotZ: rz}}})
}

// Sample 采样骨骼在 t 时刻的局部变换（线性插值；t 之前/之后取首/末帧）。
func (a *Animation) Sample(boneIdx int, t float32) (math3d.Vec3, float32, float32, float32) {
	for ti := range a.Tracks {
		tr := &a.Tracks[ti]
		if tr.BoneIdx != boneIdx {
			continue
		}
		ks := tr.Keys
		if len(ks) == 0 {
			return math3d.Vec3{}, 0, 0, 0
		}
		if t <= ks[0].Time {
			k := ks[0]
			return k.Pos, k.RotX, k.RotY, k.RotZ
		}
		if t >= ks[len(ks)-1].Time {
			k := ks[len(ks)-1]
			return k.Pos, k.RotX, k.RotY, k.RotZ
		}
		for i := 0; i < len(ks)-1; i++ {
			if t >= ks[i].Time && t <= ks[i+1].Time {
				a0, a1 := ks[i], ks[i+1]
				span := a1.Time - a0.Time
				if span <= 0 {
					return a0.Pos, a0.RotX, a0.RotY, a0.RotZ
				}
				f := (t - a0.Time) / span
				return a0.Pos.Lerp(a1.Pos, f),
					a0.RotX + (a1.RotX-a0.RotX)*f,
					a0.RotY + (a1.RotY-a0.RotY)*f,
					a0.RotZ + (a1.RotZ-a0.RotZ)*f
			}
		}
	}
	return math3d.Vec3{}, 0, 0, 0
}

// ApplyToSkeleton 把动画采样应用到骨骼（t 时刻姿态）。
func (a *Animation) ApplyToSkeleton(s *Skeleton, t float32) {
	for i := range s.Bones {
		p, rx, ry, rz := a.Sample(i, t)
		b := s.Bones[i]
		b.Pos = p
		b.RotX, b.RotY, b.RotZ = rx, ry, rz
	}
}

// ---------- 骨骼可视化 ----------

// SkeletonEdges 返回骨骼可视化线段（世界坐标，每根骨骼从父关节到自身关节）。
func (s *Skeleton) SkeletonEdges() [][2]math3d.Vec3 {
	var out [][2]math3d.Vec3
	for i := range s.Bones {
		p := s.Bones[i].Parent
		if p >= 0 && p < len(s.Bones) {
			out = append(out, [2]math3d.Vec3{s.BoneWorldPos(p), s.BoneWorldPos(i)})
		}
	}
	return out
}

// Deg2Rad 角度转弧度。
func Deg2Rad(d float32) float32 { return d * math.Pi / 180 }
