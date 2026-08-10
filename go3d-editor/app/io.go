package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go3d/math3d"
	"go3d/mesh"
)

// objJSON 对象序列化格式。
type objJSON struct {
	Name    string    `json:"name"`
	Type    int       `json:"type"`
	Pos     [3]float32 `json:"pos"`
	Rot     [3]float32 `json:"rot"` // 弧度
	Scale   float32   `json:"scale"`
	Color   [3]uint8  `json:"color"`
	Visible bool      `json:"visible"`

	// 自定义网格（草图拉伸/CSG/导入）
	CustomVerts [][]float32 `json:"custom_verts,omitempty"`
	CustomFaces [][]int     `json:"custom_faces,omitempty"`
	// 骨骼
	Bones []boneJSON `json:"bones,omitempty"`
	// 动画
	Anim *animJSON `json:"anim,omitempty"`
}

type boneJSON struct {
	Name   string    `json:"name"`
	Parent int       `json:"parent"`
	Pos    [3]float32 `json:"pos"`
	Rot    [3]float32 `json:"rot"`
}

type keyJSON struct {
	T   float32   `json:"t"`
	Pos [3]float32 `json:"pos"`
	Rot [3]float32 `json:"rot"`
}

type trackJSON struct {
	Bone int       `json:"bone"`
	Keys []keyJSON `json:"keys"`
}

type animJSON struct {
	Name     string      `json:"name"`
	Duration float32     `json:"duration"`
	Loop     bool        `json:"loop"`
	Tracks   []trackJSON `json:"tracks"`
}

// docJSON 文档序列化格式。
type docJSON struct {
	Name string    `json:"name"`
	Objs []objJSON `json:"objs"`
}

// LoadDocumentFile 从文件加载文档（关卡编辑器复用）。
func LoadDocumentFile(path string) (*Document, error) {
	e := New(320, 200)
	if err := e.Load(path); err != nil {
		return nil, err
	}
	return e.doc, nil
}

// SaveDocumentFile 保存文档到文件（关卡编辑器复用）。
func SaveDocumentFile(doc *Document, path string) error {
	e := New(320, 200)
	e.doc = doc
	return e.Save(path)
}

// Save 保存文档到 JSON 文件。
func (e *Editor) Save(path string) error {
	dj := docJSON{Name: e.doc.Name}
	for _, o := range e.doc.Objs {
		oj := objJSON{
			Name:    o.Name,
			Type:    int(o.Type),
			Pos:     [3]float32{o.Pos.X, o.Pos.Y, o.Pos.Z},
			Rot:     [3]float32{o.RotX, o.RotY, o.RotZ},
			Scale:   o.Scale,
			Color:   [3]uint8{o.Color.R, o.Color.G, o.Color.B},
			Visible: o.Visible,
		}
		// 自定义网格
		if o.CustomMesh != nil {
			for _, p := range o.CustomMesh.Positions {
				oj.CustomVerts = append(oj.CustomVerts, []float32{p.X, p.Y, p.Z})
			}
			for i := range o.CustomMesh.Faces {
				f := &o.CustomMesh.Faces[i]
				oj.CustomFaces = append(oj.CustomFaces, []int{f.A, f.B, f.C})
			}
		}
		// 骨骼
		if o.Skeleton != nil {
			for i := range o.Skeleton.Bones {
				b := o.Skeleton.Bones[i]
				oj.Bones = append(oj.Bones, boneJSON{
					Name:   b.Name,
					Parent: b.Parent,
					Pos:    [3]float32{b.Pos.X, b.Pos.Y, b.Pos.Z},
					Rot:    [3]float32{b.RotX, b.RotY, b.RotZ},
				})
			}
		}
		// 动画
		if o.Anim != nil {
			aj := &animJSON{Name: o.Anim.Name, Duration: o.Anim.Duration, Loop: o.Anim.Loop}
			for _, tr := range o.Anim.Tracks {
				tj := trackJSON{Bone: tr.BoneIdx}
				for _, k := range tr.Keys {
					tj.Keys = append(tj.Keys, keyJSON{
						T:   k.Time,
						Pos: [3]float32{k.Pos.X, k.Pos.Y, k.Pos.Z},
						Rot: [3]float32{k.RotX, k.RotY, k.RotZ},
					})
				}
				aj.Tracks = append(aj.Tracks, tj)
			}
			oj.Anim = aj
		}
		dj.Objs = append(dj.Objs, oj)
	}
	data, err := json.MarshalIndent(dj, "", "  ")
	if err != nil {
		return err
	}
	// 目标目录不存在时自动创建（保存对话框允许自定义路径）
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

// Load 从 JSON 文件加载文档（失败返回错误，不改变当前文档）。
func (e *Editor) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var dj docJSON
	if err := json.Unmarshal(data, &dj); err != nil {
		return err
	}
	doc := NewDocument(dj.Name)
	if doc.Name == "" {
		doc.Name = "零件1"
	}
	for _, o := range dj.Objs {
		obj := &Object{
			Name:    o.Name,
			Type:    ObjType(o.Type),
			Pos:     math3d.Vec3{o.Pos[0], o.Pos[1], o.Pos[2]},
			RotX:    o.Rot[0],
			RotY:    o.Rot[1],
			RotZ:    o.Rot[2],
			Scale:   o.Scale,
			Color:   mesh.Col(o.Color[0], o.Color[1], o.Color[2]),
			Visible: o.Visible,
		}
		if obj.Scale == 0 {
			obj.Scale = 1
		}
		if obj.Name == "" {
			obj.Name = obj.Type.ShortName()
		}
		// 自定义网格
		if len(o.CustomVerts) > 0 && len(o.CustomFaces) > 0 {
			m := &mesh.Mesh{}
			for _, v := range o.CustomVerts {
				if len(v) >= 3 {
					m.Positions = append(m.Positions, math3d.Vec3{v[0], v[1], v[2]})
				}
			}
			for _, f := range o.CustomFaces {
				if len(f) >= 3 {
					m.Faces = append(m.Faces, mesh.Face{A: f[0], B: f[1], C: f[2], Col: obj.Color})
				}
			}
			if len(m.Positions) > 0 {
				obj.CustomMesh = m
			}
		}
		// 骨骼
		if len(o.Bones) > 0 {
			sk := NewSkeleton()
			for _, bj := range o.Bones {
				sk.AddBone(bj.Name, bj.Parent, math3d.Vec3{bj.Pos[0], bj.Pos[1], bj.Pos[2]})
				b := sk.Bones[len(sk.Bones)-1]
				b.RotX, b.RotY, b.RotZ = bj.Rot[0], bj.Rot[1], bj.Rot[2]
			}
			obj.Skeleton = sk
			sk.ResetBindPose()
		}
		// 动画
		if o.Anim != nil {
			a := &Animation{Name: o.Anim.Name, Duration: o.Anim.Duration, Loop: o.Anim.Loop}
			for _, tj := range o.Anim.Tracks {
				for _, kj := range tj.Keys {
					a.AddKey(tj.Bone, kj.T, math3d.Vec3{kj.Pos[0], kj.Pos[1], kj.Pos[2]}, kj.Rot[0], kj.Rot[1], kj.Rot[2])
				}
			}
			obj.Anim = a
		}
		// 权重：绑定姿势下重算（当前自动权重，重算与保存一致）
		if obj.Skeleton != nil && len(obj.Skeleton.Bones) > 0 {
			obj.Weights = obj.Skeleton.SkinWeights(obj.RenderMesh(), 0)
		}
		doc.Objs = append(doc.Objs, obj)
	}
	e.doc = doc
	e.sel = -1
	e.drag = DragNone
	e.fieldFocus = -1
	e.renaming = false
	return nil
}

// saveFmtName 格式名称。
func saveFmtName(f int) string {
	switch f {
	case 0:
		return "JSON 场景"
	case 2:
		return "GLB"
	default:
		return "OBJ"
	}
}

// saveFmtExt 格式默认后缀。
func saveFmtExt(f int) string {
	switch f {
	case 0:
		return ".json"
	case 2:
		return ".glb"
	default:
		return ".obj"
	}
}

// setSaveFmt 切换保存格式：同步替换 saveBuf 后缀（无后缀则补）。
func (e *Editor) setSaveFmt(f int) {
	e.saveFmt = f
	p := strings.TrimSpace(e.saveBuf)
	lower := strings.ToLower(p)
	for _, old := range []string{".json", ".obj", ".glb", ".stl"} {
		if strings.HasSuffix(lower, old) {
			p = p[:len(p)-len(old)]
			break
		}
	}
	if p == "" {
		p = "model"
	}
	e.saveBuf = p + saveFmtExt(f)
	e.saveBufEdited = true
	e.SetMessage("保存格式: %s", saveFmtName(f))
}

// saveDoc 打开保存对话框（询问保存位置和文件名）。
// 默认导出 OBJ（实际可用模型）；点格式按钮可切换 JSON 场景/GLB。
func (e *Editor) saveDoc() {
	if e.lastSaveRel == "" {
		e.lastSaveRel = "hermes11/go3d-toolchain/go3d-editor/model.obj"
	}
	e.saveFmt = 1 // 默认 OBJ
	e.saveBuf = e.lastSaveRel
	e.setSaveFmt(e.saveFmt)
	e.saveBufEdited = false
	e.saveDialogOpen = true
	e.SetMessage("保存: 选格式 → 确认文件名 → 保存（默认导出 model.obj）")
}

// exportMesh 返回导出网格：有选中可见对象则导出它，否则合并全部可见对象。
func (e *Editor) exportMesh() (*mesh.Mesh, bool) {
	var objs []*Object
	if sel := e.selObj(); sel != nil && sel.Visible {
		objs = []*Object{sel}
	} else {
		objs = e.doc.VisibleObjs()
	}
	if len(objs) == 0 {
		return nil, false
	}
	m := &mesh.Mesh{}
	for _, o := range objs {
		om := o.RenderMesh()
		base := len(m.Positions)
		m.Positions = append(m.Positions, om.Positions...)
		for _, f := range om.Faces {
			m.Faces = append(m.Faces, mesh.Face{A: f.A + base, B: f.B + base, C: f.C + base, Col: f.Col})
		}
	}
	return m, true
}

// doSave 执行保存：saveBuf 为相对 home 或绝对路径；按 saveFmt 分发格式。
func (e *Editor) doSave() {
	p := strings.TrimSpace(e.saveBuf)
	if p == "" {
		e.saveDialogOpen = false
		e.SetMessage("保存取消：文件名为空")
		return
	}
	// 自动补后缀（用户没写时按当前格式）
	ext := saveFmtExt(e.saveFmt)
	if !strings.HasSuffix(strings.ToLower(p), ext) {
		p += ext
	}
	path := p
	if !strings.HasPrefix(p, "/") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			path = filepath.Join(home, p)
		}
	}
	var err error
	switch e.saveFmt {
	case 0: // JSON 场景（配合"载入"恢复编辑状态）
		err = e.Save(path)
	case 1: // OBJ 导出
		m, ok := e.exportMesh()
		if !ok {
			e.SetMessage("无可见对象可导出")
			return
		}
		err = SaveOBJ(path, m)
	case 2: // GLB 导出
		m, ok := e.exportMesh()
		if !ok {
			e.SetMessage("无可见对象可导出")
			return
		}
		err = SaveGLB(path, m)
	default:
		e.SetMessage("未知保存格式")
		return
	}
	if err != nil {
		e.SetMessage("保存失败: %v", err)
		return
	}
	e.lastSaveRel = p
	e.saveDialogOpen = false
	if e.saveFmt == 0 {
		e.SetMessage("已保存场景: %s（%d 特征）", path, len(e.doc.Objs))
	} else {
		e.SetMessage("已导出 %s: %s", saveFmtName(e.saveFmt), path)
	}
}

// loadDoc 从默认路径载入。
func (e *Editor) loadDoc() {
	path := defaultScenePath()
	if err := e.Load(path); err != nil {
		e.SetMessage("载入失败: %v", err)
		return
	}
	e.SetMessage("已载入: %s（%d 特征）", path, len(e.doc.Objs))
}

// defaultScenePath 默认场景文件路径。
func defaultScenePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "scene.json"
	}
	return fmt.Sprintf("%s/hermes11/go3d-toolchain/go3d-editor/scene.json", home)
}
