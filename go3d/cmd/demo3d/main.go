// demo3d 是 go3d 引擎的演示程序：彩色立方体 + 公转球体 + 金字塔 + 地面网格。
//
// 交互（X11 桌面窗口）：
//   W/S 前进后退  A/D 左右平移  Q/E 升降
//   鼠标左键拖拽 旋转视角  F 线框/实体切换  O 自动环绕开关
//   R 重置相机  Esc/Q 退出
//
// 无头渲染（不需要 X，用于验证/截图）：
//   ./bin/demo3d --offscreen --out shot.png --res 480x270
//   ./bin/demo3d --offscreen --frames 36 --outdir shots/   # 序列帧
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go2dgame/engine"
	"go3d/math3d"
	"go3d/mesh"
	"go3d/render"
	"math"
)

// ---------- 场景 ----------

type scene struct {
	cube     render.Object
	sphere   render.Object
	pyramid  render.Object
	ground   render.Object
	groundWire render.Object // 地面网格线框覆盖（可选）
}

func buildScene() *scene {
	s := &scene{}
	s.cube = render.Object{Mesh: mesh.Cube(1.4), Pos: math3d.Vec3{0, 0.8, 0}, Scale: 1}
	s.sphere = render.Object{Mesh: mesh.Sphere(0.5, 20, 10), Pos: math3d.Vec3{0, 0.8, 0}, Scale: 1}
	s.pyramid = render.Object{Mesh: mesh.Pyramid(1.2, 1.6), Pos: math3d.Vec3{2.2, 0, 0.8}, Scale: 1}
	s.ground = render.Object{Mesh: mesh.Grid(0, 12, 18), Pos: math3d.Vec3{0, 0, 0}, Scale: 1}
	return s
}

// ---------- 游戏逻辑（X11 模式） ----------

// orbitAng 自动环绕累计角（模块级，跨帧保持）。
var orbitAng float32

type game struct {
	eng      *engine.Engine
	rd       *render.Renderer
	s        *scene
	cam      *render.Camera
	orbit    bool
	lastMX   int
	lastMY   int
	haveLast bool
	frames   int
	lastFPS  time.Time
	fpsCount int
	fps      int
	totalTri int
}

func newGame(eng *engine.Engine, w, h int) *game {
	g := &game{
		eng:   eng,
		rd:    render.NewRenderer(w, h),
		s:     buildScene(),
		cam:   render.DefaultCamera(),
		orbit: true,
	}
	return g
}

func (g *game) Update(dt float64) {
	in := g.eng.Input()
	speed := float32(3.5 * dt)

	// 自动环绕：相机绕场景中心旋转并始终看向原点
	if g.orbit {
		orbitAng += float32(0.35 * dt)
		g.cam.Pos = math3d.Vec3{6.5 * float32(math.Sin(float64(orbitAng))), 2.2, 6.5 * float32(math.Cos(float64(orbitAng)))}
		g.cam.Yaw = -orbitAng
		g.cam.Pitch = -0.12
	}

	// 手动模式（orbit 关闭时）：WASD 移动 + Q/E 升降
	f := math3d.Forward(g.cam.Yaw, 0)
	right := f.Cross(math3d.Vec3{0, 1, 0})
	if !g.orbit {
		if in.Down(engine.KeyChar('w')) || in.Down(engine.KeyChar('W')) {
			g.cam.Pos = g.cam.Pos.Add(f.MulScalar(speed))
		}
		if in.Down(engine.KeyChar('s')) || in.Down(engine.KeyChar('S')) {
			g.cam.Pos = g.cam.Pos.Sub(f.MulScalar(speed))
		}
		if in.Down(engine.KeyChar('a')) || in.Down(engine.KeyChar('A')) {
			g.cam.Pos = g.cam.Pos.Sub(right.MulScalar(speed))
		}
		if in.Down(engine.KeyChar('d')) || in.Down(engine.KeyChar('D')) {
			g.cam.Pos = g.cam.Pos.Add(right.MulScalar(speed))
		}
		if in.Down(engine.KeyChar('q')) || in.Down(engine.KeyChar('Q')) {
			g.cam.Pos.Y += speed
		}
		if in.Down(engine.KeyChar('e')) || in.Down(engine.KeyChar('E')) {
			g.cam.Pos.Y -= speed
		}
	}
	// 鼠标拖拽旋转（自动环绕时拖拽会退出环绕）
	if in.MouseLeft {
		if g.orbit {
			g.orbit = false
			g.haveLast = false
		}
		if g.haveLast {
			dx := in.MouseX - g.lastMX
			dy := in.MouseY - g.lastMY
			g.cam.Yaw += float32(dx) * 0.006
			g.cam.Pitch -= float32(dy) * 0.006
			if g.cam.Pitch > 1.4 {
				g.cam.Pitch = 1.4
			}
			if g.cam.Pitch < -1.4 {
				g.cam.Pitch = -1.4
			}
		}
		g.lastMX, g.lastMY = in.MouseX, in.MouseY
		g.haveLast = true
	} else {
		g.haveLast = false
	}
	if in.Pressed(engine.KeyChar('f')) || in.Pressed(engine.KeyChar('F')) {
		g.rd.Wireframe = !g.rd.Wireframe
	}
	if in.Pressed(engine.KeyChar('o')) || in.Pressed(engine.KeyChar('O')) {
		g.orbit = !g.orbit
	}
	if in.Pressed(engine.KeyChar('r')) || in.Pressed(engine.KeyChar('R')) {
		g.cam = render.DefaultCamera()
	}
	// 物体动画
	g.s.cube.RotY += float32(0.8 * dt)
	g.s.cube.RotX += float32(0.35 * dt)
	// 球体公转
	ang := float64(time.Now().UnixMilli()) / 1000.0 * 0.9
	g.s.sphere.Pos = math3d.Vec3{float32(2.1 * math3dCos(ang)), 0.8, float32(2.1 * math3dSin(ang))}
	g.s.pyramid.RotY += float32(0.5 * dt)

	// FPS 统计
	g.frames++
	now := time.Now()
	if now.Sub(g.lastFPS) >= time.Second {
		g.fps = g.fpsCount
		g.fpsCount = 0
		g.lastFPS = now
	}
	g.fpsCount++
}

func math3dCos(a float64) float32 { return float32(math.Cos(a)) }
func math3dSin(a float64) float32 { return float32(math.Sin(a)) }

func (g *game) Draw(c *engine.Canvas) {
	g.rd.Clear(c.Pixels)
	objs := []render.Object{
		g.s.cube, g.s.sphere, g.s.pyramid, g.s.ground,
	}
	g.totalTri = g.rd.Render(c.Pixels, g.cam, objs)
	// 顶栏提示
	help := "WASD move  Q/E up-down  Drag rotate  F wire  O orbit  R reset  Esc quit"
	c.Text(8, 6, help, engine.ColWhite, 1)
	c.Text(8, 18, fmt.Sprintf("FPS %d  tris %d  wire %v  yaw %.2f", g.fps, g.totalTri, g.rd.Wireframe, g.cam.Yaw), engine.ColYellow, 1)
}

// ---------- 无头渲染（offscreen，不需要 X） ----------

func renderOffscreen(w, h int, frames int, out string) error {
	cv := engine.NewCanvas(w, h)
	rd := render.NewRenderer(w, h)
	rd.Wireframe = false
	s := buildScene()
	cam := render.DefaultCamera()

	if frames <= 0 {
		frames = 1
	}
	// 帧号填充宽度
	pad := len(strconv.Itoa(frames))
	for i := 0; i < frames; i++ {
		t := float64(i) * (2 * 3.14159265 / float64(frames))
		cam.Yaw = float32(t)
		// 球体公转
		ang := t * 2.5
		s.sphere.Pos = math3d.Vec3{float32(2.1 * math.Cos(ang)), 0.8, float32(2.1 * math.Sin(ang))}
		s.cube.RotY = float32(t)
		s.cube.RotX = float32(0.4 * t)
		s.pyramid.RotY = float32(0.6 * t)

		rd.Clear(cv.Pixels)
		objs := []render.Object{s.cube, s.sphere, s.pyramid, s.ground}
		tris := rd.Render(cv.Pixels, cam, objs)

		var path string
		if frames == 1 {
			path = out
		} else {
			path = filepath.Join(out, fmt.Sprintf("frame_%0*d.png", pad, i))
		}
		if err := savePixelsPNG(path, cv); err != nil {
			return err
		}
		if frames == 1 {
			fmt.Printf("saved %s (%dx%d, %d tris)\n", path, w, h, tris)
		}
	}
	if frames > 1 {
		fmt.Printf("saved %d frames to %s/\n", frames, out)
	}
	return nil
}

// savePixelsPNG 把 B,G,R,pad 像素缓冲写成 PNG。
func savePixelsPNG(path string, cv *engine.Canvas) error {
	img := image.NewRGBA(image.Rect(0, 0, cv.W, cv.H))
	for y := 0; y < cv.H; y++ {
		for x := 0; x < cv.W; x++ {
			i := (y*cv.W + x) * 4
			img.SetRGBA(x, y, color.RGBA{
				R: cv.Pixels[i+2], G: cv.Pixels[i+1], B: cv.Pixels[i], A: 255,
			})
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// ---------- main ----------

func parseRes(s string) (int, int, error) {
	parts := strings.SplitN(s, "x", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("bad res %q (want WxH)", s)
	}
	w, err1 := strconv.Atoi(parts[0])
	h, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("bad res %q", s)
	}
	return w, h, nil
}

func main() {
	var (
		offscreen = flag.Bool("offscreen", false, "离屏渲染一帧/序列存 PNG（不需要 X）")
		out       = flag.String("out", "shot.png", "offscreen 单帧输出路径")
		frames    = flag.Int("frames", 1, "offscreen 帧数（>1 时输出到 --outdir）")
		outdir    = flag.String("outdir", "shots", "offscreen 多帧输出目录")
		res       = flag.String("res", "", "分辨率 WxH（默认 640x360）")
	)
	flag.Parse()

	w, h := 640, 360
	if *res != "" {
		var err error
		w, h, err = parseRes(*res)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}

	if *offscreen {
		target := *out
		if *frames > 1 {
			target = *outdir
		}
		if err := renderOffscreen(w, h, *frames, target); err != nil {
			fmt.Fprintln(os.Stderr, "offscreen error:", err)
			os.Exit(1)
		}
		return
	}

	cfg := engine.DefaultConfig("go3d demo")
	cfg.Width, cfg.Height = w, h
	eng, err := engine.New(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "engine:", err)
		os.Exit(1)
	}
	defer eng.Close()
	g := newGame(eng, w, h)
	if err := eng.Run(g); err != nil {
		fmt.Fprintln(os.Stderr, "run:", err)
		os.Exit(1)
	}
}
