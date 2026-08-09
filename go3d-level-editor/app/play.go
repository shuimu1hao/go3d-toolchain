package app

import (
	"math"

	"go2dgame/engine"
	"go3d/math3d"
)

// updatePlay 运行模式：玩家控制 + 简化物理。
func (e *Editor) updatePlay(in *engine.Input, dt float32) {
	// Esc 返回编辑
	if in.Pressed(engine.KeyEscape) {
		e.playing = false
		e.SetMessage("返回编辑模式")
		return
	}
	if e.player == nil {
		e.SetMessage("未设置玩家（选中实例→设为玩家）")
		e.playing = false
		return
	}
	p := e.player
	// 移动（相对相机朝向，水平）
	speed := float32(4)
	if e.inputMap.Held(in, ActRun) {
		speed = 8
	}
	cam := e.cam.Camera()
	// 相机水平前向
	cp := float32(math.Cos(float64(cam.Pitch)))
	fwd := math3d.Vec3{cp * float32(math.Sin(float64(cam.Yaw))), 0, cp * float32(math.Cos(float64(cam.Yaw)))}
	right := math3d.Vec3{fwd.Z, 0, -fwd.X}
	move := math3d.Vec3{}
	if e.inputMap.Held(in, ActForward) {
		move = move.Add(fwd)
	}
	if e.inputMap.Held(in, ActBack) {
		move = move.Sub(fwd)
	}
	if e.inputMap.Held(in, ActLeft) {
		move = move.Sub(right)
	}
	if e.inputMap.Held(in, ActRight) {
		move = move.Add(right)
	}
	if move.Length() > 0 {
		move = move.Normalized().MulScalar(speed * dt)
		p.Pos = p.Pos.Add(move)
	}
	// 跳跃（简化：落地时起跳）
	if e.inputMap.Held(in, ActJump) && p.Pos.Y < 0.01 && p.Pos.Y > -0.5 {
		p.Vel.Y = 8
	}
	// 重力 + 积分
	p.Vel.Y += e.gravity * dt
	p.Pos.Y += p.Vel.Y * dt
	// 落地
	if p.Pos.Y < 0 {
		p.Pos.Y = 0
		p.Vel.Y = 0
	}
	// 面向移动方向（简单：绕 Y 旋转）
	// 相机跟随玩家
	e.cam.Target = p.Pos
	e.cam.Dist = 8
	// 摄像机跟随：目标跟着玩家，yaw 保持
	e.SetMessage("运行中: 位置 (%.1f, %.1f, %.1f)", p.Pos.X, p.Pos.Y, p.Pos.Z)
}
