package app

import (
	"fmt"
	"math"
	"os"

	"go3d/math3d"
	"go3d/mesh"
)

// ---------- 草图模式 ----------

// sketchClick 视口点击：按工具添加草图元素。
func (e *Editor) sketchClick(mx, my int) {
	if e.sketch == nil {
		e.sketch = NewSketch(PlaneXY)
	}
	// 反投影到草图平面
	wp, ok := e.unproject(float32(mx), float32(my), math3d.Vec3{0, 0, 0}, e.sketch.Normal())
	if !ok {
		return
	}
	p := e.sketch.WorldToLocal(wp)
	switch e.sketchTool {
	case 0: // 折线
		e.sketch.AddPoint(p.U, p.V)
		e.SetMessage("草图点 (%g, %g)  Enter 闭合", p.U, p.V)
	case 1: // 矩形
		if !e.sketchPtSet() {
			e.sketchPt0 = p
			e.SetMessage("矩形第一角 (%g, %g)", p.U, p.V)
			return
		}
		pts := RectSketch(e.sketchPt0.U, e.sketchPt0.V, p.U, p.V)
		e.sketch.Points = append(e.sketch.Points, pts...)
		e.sketch.Closed = true
		e.sketchPt0 = Vec2{}
		e.SetMessage("矩形完成，点击 拉伸 生成实体")
	case 2: // 圆
		if !e.sketchPtSet() {
			e.sketchPt0 = p
			e.SetMessage("圆心 (%g, %g)，再点半径", p.U, p.V)
			return
		}
		dx := p.U - e.sketchPt0.U
		dy := p.V - e.sketchPt0.V
		r := float32(mathSqrt(float64(dx*dx + dy*dy)))
		pts := CircleSketch(e.sketchPt0.U, e.sketchPt0.V, r, 32)
		e.sketch.Points = append(e.sketch.Points, pts...)
		e.sketch.Closed = true
		e.sketchPt0 = Vec2{}
		e.SetMessage("圆完成，点击 拉伸 生成实体")
	}
}

func (e *Editor) sketchPtSet() bool {
	return e.sketchPt0.U != 0 || e.sketchPt0.V != 0
}

// sketchClose 闭合草图（首尾相连）。
func (e *Editor) sketchClose() {
	if e.sketch == nil || len(e.sketch.Points) < 3 {
		return
	}
	e.sketch.Closed = true
	e.SetMessage("草图已闭合（%d 点），点击 拉伸", len(e.sketch.Points))
}

// sketchExtrude 把当前草图拉伸成实体对象。
func (e *Editor) sketchExtrude() {
	if e.sketch == nil || !e.sketch.Closed || len(e.sketch.Points) < 3 {
		e.SetMessage("草图未闭合或点数不足")
		return
	}
	m := ExtrudeMesh(e.sketch.Points, e.sketch.Plane, 1.0, mesh.Col(200, 170, 120))
	if m == nil {
		e.SetMessage("拉伸失败（轮廓无效）")
		return
	}
	o := e.doc.AddObjFromMesh("拉伸体", m)
	e.sel = len(e.doc.Objs) - 1
	e.SetMessage("拉伸完成: %s（%d 面）", o.Name, len(m.Faces))
}

// sketchSetPlane 切换草图平面。
func (e *Editor) sketchSetPlane(p SketchPlane) {
	if e.sketch == nil {
		e.sketch = NewSketch(p)
	} else {
		e.sketch.Plane = p
	}
	e.sketchPt0 = Vec2{}
	names := map[SketchPlane]string{PlaneXY: "前视 XY", PlaneXZ: "顶视 XZ", PlaneYZ: "右视 YZ"}
	e.SetMessage("草图平面: %s", names[p])
}

// sketchClear 清空草图。
func (e *Editor) sketchClear() {
	if e.sketch != nil {
		e.sketch.Clear()
		e.sketchPt0 = Vec2{}
	}
	e.SetMessage("草图已清空")
}

// ---------- 骨骼模式 ----------

// selObj 返回选中对象。
func (e *Editor) selObj() *Object {
	if e.sel < 0 || e.sel >= len(e.doc.Objs) {
		return nil
	}
	return e.doc.Objs[e.sel]
}

// boneEnsure 确保选中对象有骨骼。
func (e *Editor) boneEnsure() *Skeleton {
	o := e.selObj()
	if o == nil {
		return nil
	}
	if o.Skeleton == nil {
		o.Skeleton = NewSkeleton()
	}
	return o.Skeleton
}

// boneAdd 向选中对象添加骨骼（根骨骼或选中骨骼的子骨骼）。
func (e *Editor) boneAdd() {
	s := e.boneEnsure()
	if s == nil {
		e.SetMessage("请先选中一个对象")
		return
	}
	parent := e.selBone
	pos := math3d.Vec3{0, 1, 0}
	if parent >= 0 && parent < len(s.Bones) {
		pos = s.BoneWorldPos(parent).Add(math3d.Vec3{0, 1, 0})
	}
	idx := s.AddBone(fmt.Sprintf("骨骼%d", len(s.Bones)+1), parent, pos)
	e.selBone = idx
	e.SetMessage("添加骨骼 #%d（父 %d）", idx, parent)
}

// boneDelete 删除选中骨骼（子骨骼一并删除）。
func (e *Editor) boneDelete() {
	s := e.selObj()
	if s == nil || s.Skeleton == nil || e.selBone < 0 || e.selBone >= len(s.Skeleton.Bones) {
		e.SetMessage("无选中骨骼")
		return
	}
	removeBoneTree(s.Skeleton, e.selBone)
	e.selBone = -1
	e.SetMessage("删除骨骼")
}

// removeBoneTree 删除骨骼及其子树，重排索引。
func removeBoneTree(s *Skeleton, idx int) {
	// 标记要删的子树
	toDelete := map[int]bool{}
	var mark func(i int)
	mark = func(i int) {
		toDelete[i] = true
		for j, b := range s.Bones {
			if b.Parent == i && !toDelete[j] {
				mark(j)
			}
		}
	}
	mark(idx)
	// 重建数组（父索引重排）
	newBones := []*Bone{}
	newPoses := []Bone{}
	remap := map[int]int{}
	for i, b := range s.Bones {
		if toDelete[i] {
			continue
		}
		remap[i] = len(newBones)
		cp := *b
		if b.Parent >= 0 && !toDelete[b.Parent] {
			cp.Parent = remap[b.Parent]
		} else {
			cp.Parent = -1
		}
		newBones = append(newBones, &cp)
		newPoses = append(newPoses, cp)
	}
	s.Bones = newBones
	s.BindPoses = newPoses
}

// boneBind 自动计算顶点权重（绑定到骨骼）。
func (e *Editor) boneBind() {
	o := e.selObj()
	if o == nil || o.Skeleton == nil || len(o.Skeleton.Bones) == 0 {
		e.SetMessage("请先选中对象并添加骨骼")
		return
	}
	o.Skeleton.ResetBindPose()
	m := o.Type.NewMesh()
	o.Weights = o.Skeleton.SkinWeights(m, 0)
	e.SetMessage("绑定完成：%d 顶点 × %d 骨骼", len(o.Weights), len(o.Skeleton.Bones))
}

// boneResetPose 重置绑定姿势。
func (e *Editor) boneResetPose() {
	o := e.selObj()
	if o == nil || o.Skeleton == nil {
		e.SetMessage("无骨骼")
		return
	}
	o.Skeleton.ResetBindPose()
	e.SetMessage("绑定姿势已重置")
}

// bonePick 视口点击拾取骨骼（最近投影距离）。
func (e *Editor) bonePick(mx, my int) {
	o := e.selObj()
	if o == nil || o.Skeleton == nil {
		e.selBone = -1
		return
	}
	fx, fy := float32(mx), float32(my)
	best, bestD := -1, float32(1e9)
	for i := range o.Skeleton.Bones {
		bp := o.Skeleton.BoneWorldPos(i)
		sx, sy, ok := e.project(bp)
		if !ok {
			continue
		}
		d := (fx-sx)*(fx-sx) + (fy-sy)*(fy-sy)
		if d < bestD {
			bestD = d
			best = i
		}
	}
	if bestD < 20*20 {
		e.selBone = best
		e.SetMessage("选中骨骼 #%d %s", best, o.Skeleton.Bones[best].Name)
	} else {
		e.selBone = -1
	}
}

// boneWorld 当前选中骨骼的世界位置（gizmo 定位）。
func (e *Editor) boneWorld() (math3d.Vec3, bool) {
	o := e.selObj()
	if o == nil || o.Skeleton == nil || e.selBone < 0 || e.selBone >= len(o.Skeleton.Bones) {
		return math3d.Vec3{}, false
	}
	return o.Skeleton.BoneWorldPos(e.selBone), true
}

// ---------- 动画模式 ----------

// animEnsure 确保选中对象有动画。
func (e *Editor) animEnsure() *Animation {
	o := e.selObj()
	if o == nil {
		return nil
	}
	if o.Anim == nil {
		o.Anim = &Animation{Name: "动画1", Loop: true}
	}
	return o.Anim
}

// animAddKey 为当前选中骨骼在当前时间添加关键帧。
func (e *Editor) animAddKey() {
	a := e.animEnsure()
	if a == nil {
		e.SetMessage("请先选中对象")
		return
	}
	o := e.selObj()
	if o.Skeleton == nil || len(o.Skeleton.Bones) == 0 {
		e.SetMessage("对象无骨骼")
		return
	}
	if e.selBone < 0 || e.selBone >= len(o.Skeleton.Bones) {
		e.selBone = 0
	}
	b := o.Skeleton.Bones[e.selBone]
	a.AddKey(e.selBone, o.AnimTime, b.Pos, b.RotX, b.RotY, b.RotZ)
	e.SetMessage("关键帧: 骨骼%d @ t=%.2f", e.selBone, o.AnimTime)
}

// animPlay 播放/暂停。
func (e *Editor) animPlay() {
	o := e.selObj()
	if o == nil || !o.HasAnim() {
		e.SetMessage("对象无动画")
		return
	}
	o.AnimPlaying = !o.AnimPlaying
	if o.AnimPlaying {
		e.SetMessage("播放: %s", o.Anim.Name)
	} else {
		e.SetMessage("暂停 @ t=%.2f", o.AnimTime)
	}
}

// animStop 停止并回到 0。
func (e *Editor) animStop() {
	o := e.selObj()
	if o == nil || o.Anim == nil {
		return
	}
	o.AnimPlaying = false
	o.AnimTime = 0
	o.Anim.ApplyToSkeleton(o.Skeleton, 0)
	e.SetMessage("停止，回到 t=0")
}

// animClear 清除动画关键帧。
func (e *Editor) animClear() {
	o := e.selObj()
	if o == nil || o.Anim == nil {
		return
	}
	o.Anim.Tracks = nil
	o.Anim.Duration = 0
	o.AnimTime = 0
	o.AnimPlaying = false
	o.Anim.ApplyToSkeleton(o.Skeleton, 0)
	e.SetMessage("动画已清除")
}

// mathSqrt 浮点开方。
func mathSqrt(x float64) float64 { return math.Sqrt(x) }

// csgApply 对选中对象（A）和上一个选中（B）执行布尔运算。
func (e *Editor) csgApply(op CSGOp) {
	a := e.selObj()
	if a == nil || len(e.doc.Objs) < 2 {
		e.SetMessage("布尔运算需要选中两个对象")
		return
	}
	// 找 B：选中列表里的另一个（简单：A 的前一个对象）
	bi := e.sel - 1
	if bi < 0 {
		bi = e.sel + 1
	}
	if bi < 0 || bi >= len(e.doc.Objs) || bi == e.sel {
		e.SetMessage("需要第二个对象")
		return
	}
	b := e.doc.Objs[bi]
	ma := a.RenderMesh()
	mb := b.RenderMesh()
	res := CSGBoolean(ma, mb, op, a.Color, b.Color)
	if res == nil || len(res.Faces) == 0 {
		e.SetMessage("布尔运算结果为空")
		return
	}
	names := map[CSGOp]string{CSGUnion: "并集", CSGSubtract: "差集", CSGIntersect: "交集"}
	o := e.doc.AddObjFromMesh(names[op]+"_"+a.Name, res)
	e.sel = len(e.doc.Objs) - 1
	e.SetMessage("%s 完成: %s（%d 面）", names[op], o.Name, len(res.Faces))
}

// importOBJ 从默认目录导入 OBJ。
func (e *Editor) importOBJ() {
	path := defaultOBJPath()
	m, err := LoadOBJ(path)
	if err != nil {
		e.SetMessage("导入失败: %v", err)
		return
	}
	_ = e.doc.AddObjFromMesh("导入模型", m)
	e.sel = len(e.doc.Objs) - 1
	e.SetMessage("导入 %s：%d 顶点 %d 面", path, len(m.Positions), len(m.Faces))
}

// importSTL 导入 STL（二进制/ASCII）。
func (e *Editor) importSTL() {
	path := defaultSTLPath()
	m, err := LoadSTL(path)
	if err != nil {
		e.SetMessage("STL 导入失败: %v", err)
		return
	}
	_ = e.doc.AddObjFromMesh("STL模型", m)
	e.sel = len(e.doc.Objs) - 1
	e.SetMessage("导入 STL：%d 顶点 %d 面", len(m.Positions), len(m.Faces))
}

// importGLB 导入 GLTF/GLB。
func (e *Editor) importGLB() {
	path := defaultGLBPath()
	m, err := LoadGLTF(path)
	if err != nil {
		e.SetMessage("GLB 导入失败: %v", err)
		return
	}
	_ = e.doc.AddObjFromMesh("GLTF模型", m)
	e.sel = len(e.doc.Objs) - 1
	e.SetMessage("导入 GLB：%d 顶点 %d 面", len(m.Positions), len(m.Faces))
}

// exportGLB 导出选中对象为 GLB。
func (e *Editor) exportGLB() {
	sel := e.selObj()
	if sel == nil {
		e.SetMessage("请先选中对象")
		return
	}
	path := defaultGLBPath()
	m := sel.RenderMesh()
	if err := SaveGLB(path, m); err != nil {
		e.SetMessage("GLB 导出失败: %v", err)
		return
	}
	e.SetMessage("已导出 GLB %s（%d 顶点）", path, len(m.Positions))
}

// defaultSTLPath 默认 STL 文件路径。
func defaultSTLPath() string {
	return defaultModelPath("model.stl")
}

// defaultGLBPath 默认 GLB 文件路径。
func defaultGLBPath() string {
	return defaultModelPath("model.glb")
}

// defaultModelPath 组装默认模型文件路径。
func defaultModelPath(name string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return name
	}
	return fmt.Sprintf("%s/hermes11/go3d-editor/%s", home, name)
}

// exportOBJ 导出选中对象为 OBJ。
func (e *Editor) exportOBJ() {
	sel := e.selObj()
	if sel == nil {
		e.SetMessage("请先选中对象")
		return
	}
	path := defaultOBJPath()
	m := sel.RenderMesh()
	if err := SaveOBJ(path, m); err != nil {
		e.SetMessage("导出失败: %v", err)
		return
	}
	e.SetMessage("已导出 %s（%d 顶点）", path, len(m.Positions))
}

// defaultOBJPath 默认 OBJ 文件路径。
func defaultOBJPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "model.obj"
	}
	return fmt.Sprintf("%s/hermes11/go3d-editor/model.obj", home)
}
