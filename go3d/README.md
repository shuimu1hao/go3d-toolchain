# go3d — 纯 Go 软渲染 3D 游戏引擎

在 Termux 图形桌面（termux-x11 + XFCE）上运行的小型 3D 游戏引擎。
**纯 Go 零 cgo**：自研 3D 数学库 + 软渲染管线，复用 go2dgame 的 X11 窗口/输入/画布。

## 架构

```
go3d/
├── math3d/        3D 数学库：Vec3 / Mat4（列主序）/ 透视投影 / LookAt / FPSView
├── mesh/          程序化网格：Cube / Sphere / Pyramid / Grid（面颜色 flat shading）
├── render/        软渲染管线
│   └── renderer.go  模型→视图→near裁剪→背面剔除→方向光→重心坐标光栅化+z-buffer
└── cmd/demo3d/    演示程序（场景 + 交互相机）
```

## 渲染管线（render.Renderer）

1. 模型变换（平移×旋转×缩放）→ 视图变换（FPSView/LookAt）
2. **near 平面裁剪**（Sutherland-Hodgman，透视除法前）
3. **背面剔除**（视图空间面法线 vs 视线方向）
4. **方向光** flat shading（环境光 + 漫反射）
5. 透视投影 + 视口变换
6. **重心坐标光栅化** + z-buffer 深度测试
7. 输出到 32bpp B,G,R,pad 像素缓冲（与 go2dgame/engine.Canvas 布局一致，直接上屏）

渲染器不依赖 X11——离屏模式可无头渲染 PNG（验证/测试用）。

## 快速开始

```bash
# 构建 + 测试
bash build.sh

# X11 桌面运行（termux-x11 开着时）
./bin/demo3d

# 离屏渲染一帧存 PNG（不需要 X）
./bin/demo3d --offscreen --out shot.png --res 640x360

# 离屏渲染动画序列帧
./bin/demo3d --offscreen --frames 36 --outdir shots/ --res 320x180

# 线框模式 / 自定义分辨率
./bin/demo3d --res 960x540
```

## 交互（X11 窗口）

| 键 | 功能 |
|---|---|
| W/S | 前进/后退 |
| A/D | 左右平移 |
| Q/E | 升降 |
| 鼠标左键拖拽 | 旋转视角 |
| F | 线框/实体切换 |
| O | 自动环绕开关（默认开） |
| R | 重置相机 |
| Esc/Q | 退出 |

顶部显示 FPS / 三角形数 / 线框状态。

## 引擎 API（30 秒上手）

```go
rd := render.NewRenderer(640, 360)          // 渲染器
cam := render.DefaultCamera()                // 相机

obj := render.Object{
    Mesh:  mesh.Cube(1.4),
    Pos:   math3d.Vec3{0, 0.8, 0},
    RotY:  0.5,                              // 弧度
    Scale: 1,
}

// X11 模式：Draw 回调里
rd.Clear(canvas.Pixels)
rd.Render(canvas.Pixels, cam, []render.Object{obj})

// 离屏模式：渲染到内存像素缓冲（B,G,R,pad）→ 存 PNG / 直接上屏
```

生成新网格：`mesh.Sphere(radius, segments, rings)`、`mesh.Pyramid(base, h)`、
`mesh.Grid(y, size, cells)`。自定义网格：填 `mesh.Mesh{Positions, Faces}`，
面环绕顺序逆时针（外侧看），法线由面计算。

## 验证记录（2026-08-09）

- `go test ./...`：math3d（矩阵/透视/LookAt/FPSView 约定）+ render（立方体渲染/遮挡/
  背面剔除/near 裁剪/线框）全过
- 无头渲染：640x360 单帧 212 tris，11058 非背景像素；12 帧序列动画帧间差异正常
- X11 实测：窗口 640x360 创建、3D 内容上屏（scrot+PIL 验证）、F 线框切换（内容 11000→2953）、
  O 环绕关闭 + W 移动（画面变化）、timeout 25s 稳定 exit 124

## 已知限制（小型引擎取舍）

- 软渲染：性能受 CPU 限制（640x360 场景 ~200 三角形轻松 60fps；960x540 降帧）
- flat shading（逐面光照），无纹理/顶点插值光照
- 无三角形级屏幕裁剪（包围盒裁剪，超大三角形跨屏幕时效率下降）
- 无 3D 音效/物理/骨骼动画（引擎聚焦渲染核心）

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
