# go2dgame - Termux 图形桌面 2D 游戏引擎

纯 Go、零 cgo 依赖的软件渲染 2D 游戏引擎，专门为
**Termux 图形桌面（termux-x11 + XFCE）** 打造 —— 开的是**真实图形窗口**，
不是终端 ASCII 界面。同一份代码也能在任何带 X11 的 Linux 桌面上编译运行。

- 图形协议：X11（github.com/BurntSushi/xgb，纯 Go 实现，无 cgo、无系统依赖）
- 渲染方式：软件渲染到像素缓冲（RGBA），分块 PutImage 上屏
- 窗口形态：固定横屏 960x540@60fps，启动自动居中、自动聚焦
- 输入：键盘（WASD/方向键/回车/空格/Esc，按 keysym 语义键，不依赖布局）+ 鼠标

## 为什么有它

`gocligames`（~/hermes11/gocligames）是终端 ASCII 引擎，在 Termux 命令行里跑。
装上 termux-x11 + XFCE 桌面后，就有了真正的图形桌面环境 —— 于是写了这个
**桌面图形版** 引擎：窗口、像素、精灵、字体，一套 API 画真·游戏画面。

## 快速开始

    cd ~/hermes11/go2dgame
    bash build.sh          # 构建三个程序到 bin/
    ./bin/pong             # 运行示例游戏「Pong 双人乒乓」

操作：W/S 移动左拍，↑/↓ 移动右拍，空格发球/重开，Esc 退出。先到 7 分获胜。

需要桌面环境在线（start-x11.sh 启动）；直接无桌面跑会报 X 连接失败。

## 目录结构

    go2dgame/
    ├── go.mod / go.sum   模块定义（module go2dgame，依赖 xgb + golang.org/x/image）
    ├── build.sh          一键构建脚本（bash build.sh [cross]）
    ├── README.md         本文档
    ├── engine/           引擎核心（包名 engine）
    │   ├── engine.go     Engine：X 连接/窗口/主循环/事件分发/键盘映射
    │   ├── canvas.go     Canvas：像素缓冲 + 绘图原语（点/矩形/线/圆/贴图）+ 分块上屏
    │   ├── input.go      Input：键盘鼠标状态（Down/Pressed/Released + 鼠标）
    │   ├── font.go       Text：内置位图字体渲染（7x13，可整数倍放大）
    │   └── sprite.go     Sprite：RGBA 像素精灵 + MakePixelArt 像素画生成
    ├── cmd/
    │   ├── pong/         示例游戏「Pong 双人乒乓」（完整玩法：计分/胜利/重开）
    │   ├── smoke/        X11 最小验证工具（开窗/渐变/动画方块，auto 模式 5 秒自退）
    │   └── xinfo/        X server 信息查询（root visual / 像素格式，排障用）
    └── bin/              构建产物

## 引擎 API

### 启动引擎

    cfg := engine.DefaultConfig("我的游戏")   // 960x540@60fps 横屏
    // 自定义：cfg.Width/Height/FPS/FullWin/WinX/WinY
    e, err := engine.New(cfg)                 // 打开 X 连接
    if err != nil { panic(err) }
    defer e.Close()
    err = e.Run(myGame)                       // 阻塞主循环，直到关窗/Esc

### 游戏逻辑（实现 engine.Game 接口）

    type Game interface {
        Update(dt float64)        // 每帧逻辑，dt 为秒（已 clamp 到 0.05）
        Draw(c *Canvas)           // 每帧渲染
    }

    func (g *MyGame) Update(dt float64) {
        in := g.eng.Input()
        if in.Down(engine.KeyChar('w')) { g.y -= speed * dt }
        if in.Pressed(engine.KeySpace) { g.fire() }
    }
    func (g *MyGame) Draw(c *engine.Canvas) {
        c.Clear(engine.Color{R: 18, G: 22, B: 30})
        c.FillRect(x, y, w, h, engine.ColCyan)
        c.Circle(cx, cy, r, engine.ColWhite)
        c.Text(10, 10, "SCORE 7", engine.ColYellow, 2)   // scale=2 放大
    }

### Canvas 绘图原语

    Clear(col) / SetPixel(x,y,col) / GetPixel(x,y)
    FillRect(x,y,w,h,col) / Rect(x,y,w,h,col)         // 实心 / 边框
    Line(x0,y0,x1,y1,col) / Circle(cx,cy,r,col) / CircleOutline(...)
    Blit(x,y,src,sw,sh)                              // RGBA 贴图（A=0 透明）
    Text(x,y,text,col,scale) / TextCentered / TextRight

内置颜色：ColBlack/White/Red/Green/Blue/Yellow/Cyan/Pink/Gray，或自定 Color{R,G,B}。

### Input 输入

    Down(k)      按住（适合持续移动）
    Pressed(k)   本帧刚按下（边缘触发，适合单发动作）
    Released(k)  本帧刚松开
    AnyPressed() 本帧任意键
    MouseX/MouseY/MouseLeft/MouseRight/MouseLeftPressed

按键常量：KeyLeft/Up/Right/Down、KeyEnter、KeyEscape、KeySpace、KeyTab、
KeyShiftL/R、KeyCtrlL/R，字符键用 KeyChar('w')。

### Sprite 精灵

    spr := engine.MakePixelArt([]string{
        ".RR.",
        "RYYR",
        "RBBR",
        ".RR.",
    }, map[byte]engine.Color{'R': engine.ColRed, 'Y': engine.ColYellow, 'B': engine.ColBlue})
    spr.Draw(c, x, y)          // 原尺寸
    spr.DrawScaled(c, x, y, 4) // 4 倍放大（最近邻像素风）

## 关键技术点

- **分块 PutImage**：xgb 生成代码用 16 位长度字段，单请求最多约 262KB；
  960x540x4B ≈ 2MB 一帧，必须按行切成小块上传（canvas.go 里 rowsPerChunk 自动切）。
- **keysym 语义键**：通过 GetKeyboardMapping 把 keycode 转 keysym，
  不受手机外接键盘布局影响，WASD/方向键稳定可识别。
- **WM_DELETE_WINDOW**：点窗口 X 按钮优雅退出。
- **X socket 探测**：Termux 无 /tmp，X socket 在 $TMPDIR/.X11-unix/ 下，
  dialX 依次探测 DISPLAY/TMPDIR/标准路径。

## 实测验证（2026-08-08，Termux + XFCE 真机）

- smoke/pong/xinfo 全部编译通过，go vet 零告警
- pong 在 XFCE 桌面开出 960x540 横屏窗口（xdotool 确认几何 960x540）
- 截图像素分析：背景 (18,22,30)、球拍 (227,157,165) 与代码颜色一致，画面真实渲染
- xdotool 发送 space/w/s/k/j/esc 键全部无崩溃，输入→更新→渲染链路通
- timeout 15 秒稳定运行不崩溃（exit=124 为 timeout 正常杀）

## 与 gocligames 的关系

`gocligames`（~/hermes11/gocligames）是终端 ASCII 引擎（跨平台四系统，
Termux 命令行里跑）。go2dgame 是**图形桌面版**，需要 termux-x11 + XFCE。
两者独立共存：命令行里玩用 gocligames，桌面上玩用 go2dgame。

## 已知限制

- 必须运行在 X11 环境下（termux-x11 或 Linux 桌面）；无桌面报 X 连接失败
- 软件渲染，960x540@60fps 是舒适区；画布再大或帧率再高会吃 CPU
- 文本用内置位图字体（7x13，只含 ASCII），中文需要自绘或扩展字体支持
- 不做横竖屏自动切换：游戏启动即横屏，其余应用锁定竖屏
- Windows 原生不支持（xgb 走 X11 协议；Windows 上需自装 X server 如 VcXsrv）

## 测试

    go vet ./...    # 静态检查
    go build ./...  # 编译检查

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
