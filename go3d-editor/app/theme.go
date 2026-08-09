package app

import (
	"encoding/json"
	"os"
	"path/filepath"

	"go2dgame/engine"
	"go3d/mesh"
)

// Theme 定义 UI 配色（SolidWorks 风格：暗色 / 白色）。
type Theme struct {
	Name     string
	Bg       engine.Color // 视口背景
	Panel    engine.Color // 工具栏/状态栏
	Panel2   engine.Color // 树/属性面板
	Border   engine.Color
	Text     engine.Color
	Dim      engine.Color
	Accent   engine.Color // 强调（标题/选中态按钮）
	Sel      engine.Color // 选中行高亮（A/当前）
	Sel2     engine.Color // 上次选中弱高亮（B，布尔运算第二对象）
	Btn      engine.Color
	BtnHover engine.Color
}

// ThemeDark 默认暗色主题（文字用白色，保证暗底对比度）。
var ThemeDark = Theme{
	Name:     "暗色",
	Bg:       engine.Color{R: 18, G: 22, B: 30},
	Panel:    engine.Color{R: 46, G: 50, B: 58},
	Panel2:   engine.Color{R: 52, G: 57, B: 66},
	Border:   engine.Color{R: 28, G: 31, B: 37},
	Text:     engine.Color{R: 255, G: 255, B: 255},
	Dim:      engine.Color{R: 185, G: 192, B: 200},
	Accent:   engine.Color{R: 90, G: 150, B: 220},
	Sel:      engine.Color{R: 70, G: 110, B: 170},
	Sel2:     engine.Color{R: 55, G: 78, B: 112},
	Btn:      engine.Color{R: 66, G: 72, B: 84},
	BtnHover: engine.Color{R: 84, G: 92, B: 108},
}

// ThemeLight 白色主题（SolidWorks 经典浅色，文字用黑色，保证白底对比度）。
var ThemeLight = Theme{
	Name:     "白色",
	Bg:       engine.Color{R: 236, G: 240, B: 244},
	Panel:    engine.Color{R: 244, G: 247, B: 250},
	Panel2:   engine.Color{R: 250, G: 252, B: 254},
	Border:   engine.Color{R: 160, G: 170, B: 182},
	Text:     engine.Color{R: 20, G: 24, B: 30},
	Dim:      engine.Color{R: 88, G: 96, B: 108},
	Accent:   engine.Color{R: 0, G: 110, B: 190},
	Sel:      engine.Color{R: 200, G: 220, B: 245},
	Sel2:     engine.Color{R: 222, G: 232, B: 248},
	Btn:      engine.Color{R: 230, G: 234, B: 240},
	BtnHover: engine.Color{R: 210, G: 220, B: 235},
}

// 当前主题。
var CurTheme = ThemeDark

// SetTheme 应用主题：更新包级 UI 颜色变量（panels.go 引用）与渲染器清屏色。
func SetTheme(t Theme) {
	CurTheme = t
	uiColorBg = t.Bg
	uiColorPanel = t.Panel
	uiColorPanel2 = t.Panel2
	uiColorBorder = t.Border
	uiColorText = t.Text
	uiColorDim = t.Dim
	uiColorAccent = t.Accent
	uiColorSel = t.Sel
	uiColorSel2 = t.Sel2
	uiColorBtn = t.Btn
	uiColorBtnHover = t.BtnHover
}

// themeBgMesh 返回主题视口背景的 mesh 颜色（渲染器清屏用）。
func themeBgMesh(t Theme) mesh.Color {
	return mesh.Col(t.Bg.R, t.Bg.G, t.Bg.B)
}

// themeName 主题索引 → 名称。
func themeName(idx int) string {
	if idx == 1 {
		return "白色"
	}
	return "暗色"
}

// themeIndex 名称 → 索引。
func themeIndex(name string) int {
	if name == "白色" {
		return 1
	}
	return 0
}

// configPath 配置文件路径（主题等持久化）。
func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".go3d-editor-config.json"
	}
	return home + "/.config/go3d-editor/config.json"
}

// loadThemeName 读取配置中的主题名（不存在/损坏返回默认"暗色"）。
func loadThemeName() string {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return "暗色"
	}
	var cfg struct {
		Theme string `json:"theme"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		return "暗色"
	}
	return cfg.Theme
}

// saveThemeName 保存主题名到配置。
func saveThemeName(name string) error {
	p := configPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	cfg := struct {
		Theme string `json:"theme"`
	}{Theme: name}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// setTheme 切换主题并持久化（Editor 方法，同时更新渲染器清屏色）。
func (e *Editor) setTheme(idx int) {
	e.theme = idx
	t := ThemeDark
	if idx == 1 {
		t = ThemeLight
	}
	SetTheme(t)
	if e.rd != nil {
		e.rd.ClearColor = themeBgMesh(t)
	}
	_ = saveThemeName(themeName(idx))
	e.SetMessage("主题: %s", themeName(idx))
}

// toggleTheme 切换主题（按钮/快捷键 T）。
func (e *Editor) toggleTheme() {
	e.setTheme(1 - e.theme)
}
