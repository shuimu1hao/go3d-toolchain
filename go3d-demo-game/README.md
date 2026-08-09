# go3d-demo-game — 纯代码 3D 小游戏

证明 go3d 引擎可以**完全用纯 Go 代码**做游戏（不依赖建模/关卡编辑器）。

## 玩法
- WASD 移动，F 冲刺，收集金币（+10 分），避开箱子（撞到掉血，HP 归零游戏结束）
- 收集全部 6 个金币胜利
- R 重开，Esc 退出

> **⚠️ Termux:X11 输入说明（2026-08-09 实测）**：在 Termux:X11 图形环境里，
> **手机屏幕虚拟键盘不会把 WASD 等按键转发给 X11 应用**，游戏会"没反应"。
> 需要**物理/蓝牙键盘**（或外接键盘）才能玩。诊断方法：`xdotool keydown w` 能移动
> 说明游戏正常，是输入通道问题；用 `xdotool` 可远程代为操作。

## 技术点（纯代码游戏模板）
- 场景对象直接构造 `render.Object{Mesh: ..., Pos: ..., ColorTint: ...}`
- 相机跟随玩家（`LookTarget: &player`）
- 简化物理：边界限制 + 碰撞检测（距离判定）+ 碰撞弹开（完全分离）
- HUD 用 5x7 位图字体（font5x7 map，无中文字体依赖）
- 薄地面用 thinBox（非等比自定义网格），避免厚板包住场景

## 运行
```
cd ~/hermes11/go3d-demo-game && ./bin/go3d-demo-game
```

## 验证记录（2026-08-09）
- X11 实测：玩家/金币/箱子渲染、WASD 移动 + 相机跟随、收集金币（对象数 13→12）、
  玩家贴地 z-fight 修复后始终显示
- 依赖：go3d（软渲染）+ go2dgame（X11/输入），go.mod replace 引用

## 协议

MIT License（见 LICENSE）

## 开发环境

- 设备：小米手机（MIUI / Android 13）
- 环境：Termux（Android 终端）+ termux-x11 + XFCE 图形桌面
- 语言：Go / Python 为主，纯 CLI 开发
- 注意：本项目在 Android / Termux 上开发与测试，其他平台运行可能需要调整

## 生成声明

本项目全部代码与文档由 AI 生成（Hermes Agent + DeepSeek 模型），不含一丝人类手写代码。仅供学习交流。

## 寻求帮助

本项目是 AI 生成的实验性游戏/引擎，仍需社区帮助测试与改进：
- 欢迎提交 Issue 反馈 Bug、卡关、体验问题
- 欢迎 PR 改进玩法、数值、画面、平台兼容性
- 目前主要在 Android / Termux 上测试，欢迎在其他平台测试反馈
