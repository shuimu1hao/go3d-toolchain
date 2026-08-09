# go3d-toolchain — 纯 Go 3D 游戏工具链

从零实现的**纯 Go、零 cgo、无第三方图形依赖**的 3D 游戏开发工具链，
全程在 Android / Termux（X11 软渲染）上开发与验证。

```
┌─────────────────────────────────────────────────────────────┐
│                     go3d 游戏工具链                           │
│                                                             │
│  go3d (3D 引擎) ──→ go3d-editor ──→ go3d-level-editor       │
│   soft-render 3D     建模编辑器      关卡编辑器               │
│  math3d/mesh/render  SolidWorks 风  场景/素材/按键/运行       │
│                      草绘/布尔/骨骼                          │
│                                                             │
│  go2dgame (基础层) ── X11 窗口/输入/画布                     │
│       ↑                                                    │
│  go3d-demo-game (纯代码游戏示例，证明引擎可直接写游戏)        │
└─────────────────────────────────────────────────────────────┘
```

## 模块一览

| 模块 | 说明 | 构建 |
|---|---|---|
| `go2dgame` | 基础层：X11 窗口、输入、2D 画布（纯 Go，xgb 客户端） | `cd go2dgame && ./build.sh` |
| `go3d` | 3D 软渲染引擎：math3d / mesh / render（透视校正 z-buffer） | `cd go3d && ./build.sh` |
| `go3d-editor` | 3D 建模编辑器：基本体/草图拉伸/布尔/骨骼动画，OBJ/STL/GLB；
   v3：保存对话框、CAD 吸附、主题切换、插件扩展、帮助会话、组合键 | `cd go3d-editor && ./build.sh` |
| `go3d-level-editor` | 关卡编辑器：建模产出 → 场景搭建/贴图素材/按键/运行 | `cd go3d-level-editor && ./build.sh` |
| `go3d-demo-game` | 纯代码 3D 小游戏示例（WASD 收集金币） | `cd go3d-demo-game && go build -o bin/... .` |

## 快速开始

```bash
# 各模块相互依赖，使用 go.mod replace 指向仓库内子目录（monorepo 内天然生效）
cd go3d && go test ./... && ./build.sh          # 引擎 + 测试
cd ../go3d-editor && go test ./... && ./build.sh # 建模编辑器
cd ../go3d-level-editor && ./build.sh            # 关卡编辑器
cd ../go3d-demo-game && ./build.sh               # 游戏示例
```

编辑器启动（需要 X11 环境，如 Termux:X11 + XFCE）：

```bash
cd go3d-editor && ./bin/go3d-editor              # 建模编辑器
cd go3d-level-editor && ./bin/go3d-level-editor  # 关卡编辑器
```

## 依赖关系

```
go3d-demo-game ─┬─ go2dgame
                └─ go3d
go3d-level-editor ─┬─ go3d-editor (app 复用)
                   └─ go3d
go3d-editor ── go3d
go3d ── (纯 stdlib + golang.org/x/image 字体)
go2dgame ── (纯 stdlib + xgb X11)
```

> 所有 `replace` 均为仓库内相对路径（`../go3d` 等），在 monorepo 内克隆即可直接构建，
> 无外部依赖、无 cgo。

## 开发验证环境

- 设备：小米手机（MIUI / Android 13）
- 环境：Termux + termux-x11 + XFCE，`DISPLAY=:0` 软渲染
- 自动化测试：xdotool 模拟点击/按键 + scrot 截图 + PIL 颜色分析
- 各模块均有单元测试（算法层：数学/网格/CSG/骨骼/GLTF 解析等）

## 协议

MIT License（见 LICENSE）

## 更新记录

### 2026-08-10 v3（go3d-editor 工程能力升级）
- 保存对话框：Ctrl+S 询问文件名/路径（相对 home 或绝对路径），目录自动创建
- 草图画圆/矩形实时预览；修复圆心在原点时第一点哨兵失效 bug
- 吸附（CAD OSNAP 风格）：F 开关 + 类型面板多选（网格/端点/中点/圆心），草图点对齐网格
- 帮助系统：F1/按钮打开新终端会话，分模块详细说明（HELP.txt）
- 主题切换：暗色/白色（T 键/按钮），暗底白字/白底黑字，持久化到 config.json
- 插件扩展点：app.Plugin 接口 + 插件面板，内置统计示例
- UI 全覆盖：三行工具栏（SolidWorks 风格），高级操作快捷键改为 Ctrl/Alt 组合键
- 布尔运算修复：CAD 选择逻辑（先选 B 再选 A，模型树双高亮）
- go3d 引擎：Renderer.ClearColor 支持主题清屏色
- go2dgame：新增 KeyAltL/KeyAltR 修饰键支持（组合键基础）
- 归档路径适配：编辑器默认保存路径/帮助文件定位改为 go3d-toolchain 内路径（帮助文件
  优先从可执行文件位置推导，项目移动后仍可用）

### 2026-08-10 v3.1（新建/清除 + 布局与测试修复）
- 新增 [新建] 按钮（工具栏行1 右端，Ctrl+N）：新建文档清空场景 + 重置全部编辑状态
- 新增 [清除] 按钮（工具栏行3 删除旁）：一键清空场景全部对象
- 修复第一行右端按钮（新建/保存/载入）与格式按钮（GLB↑）重叠：右端按钮固定窗口
  右缘定位（原 c.W-PropW 定位导致点击新建实际触发 GLB↑ 导出）
- 修复 scene.json 神秘消失根因：TestSceneFileCleanup 曾用 defaultScenePath() 删除
  真实项目文件（每次 go test 都删），已改为只验证路径格式
