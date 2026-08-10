package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go2dgame/engine"
	"go3deditor/ui"
)

// dirEntry 文件对话框目录条目。
type dirEntry struct {
	Name  string
	IsDir bool
}

// 文件对话框布局常量。
const (
	fileDlgW     = 760
	fileDlgH     = 480
	fileDlgRowH  = 32
	fileDlgListY = 66 // 列表区起始（对话框内 y 偏移）
	fileDlgBtnY  = 420 // 按钮行 y 偏移
)

// openFileDlg 打开文件选择对话框（内置浏览器，全鼠标操作）。
// mode: 0=OBJ 模型 1=建模 JSON 2=贴图素材；dir 初始目录；exts 文件扩展名过滤（小写）。
func (e *Editor) openFileDlg(mode int, title, dir string, exts []string) {
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		dir = "/"
	}
	e.fileDlgOpen = true
	e.fileDlgMode = mode
	e.fileDlgTitle = title
	e.fileDlgDir = dir
	e.fileDlgExts = exts
	e.fileDlgSel = -1
	e.fileDlgScroll = 0
	e.reloadFileDlg()
}

// reloadFileDlg 重新读取当前目录：先".."（非根）再目录再文件（按扩展名过滤），排序。
func (e *Editor) reloadFileDlg() {
	ents, err := os.ReadDir(e.fileDlgDir)
	if err != nil {
		e.fileDlgList = nil
		return
	}
	var dirs, files []dirEntry
	for _, en := range ents {
		if en.IsDir() {
			dirs = append(dirs, dirEntry{Name: en.Name(), IsDir: true})
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	for _, en := range ents {
		if en.IsDir() {
			continue
		}
		lower := strings.ToLower(en.Name())
		if len(e.fileDlgExts) > 0 && !fileDlgMatchExt(lower, e.fileDlgExts) {
			continue
		}
		files = append(files, dirEntry{Name: en.Name(), IsDir: false})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	e.fileDlgList = nil
	if filepath.Dir(e.fileDlgDir) != e.fileDlgDir {
		e.fileDlgList = append(e.fileDlgList, dirEntry{Name: "..", IsDir: true})
	}
	e.fileDlgList = append(e.fileDlgList, dirs...)
	e.fileDlgList = append(e.fileDlgList, files...)
	if e.fileDlgSel >= len(e.fileDlgList) {
		e.fileDlgSel = -1
	}
	if e.fileDlgScroll > e.fileDlgMaxScroll() {
		e.fileDlgScroll = e.fileDlgMaxScroll()
	}
}

// fileDlgMatchExt 文件名（小写）是否匹配扩展名列表。
func fileDlgMatchExt(lowerName string, exts []string) bool {
	for _, ext := range exts {
		if strings.HasSuffix(lowerName, ext) {
			return true
		}
	}
	return false
}

// fileDlgVisibleRows 列表区可见行数。
func (e *Editor) fileDlgVisibleRows() int {
	return (fileDlgH - fileDlgListY - 22 - (fileDlgH - fileDlgBtnY)) / fileDlgRowH
}

// fileDlgMaxScroll 最大滚动偏移。
func (e *Editor) fileDlgMaxScroll() int {
	v := e.fileDlgVisibleRows()
	m := len(e.fileDlgList) - v
	if m < 0 {
		return 0
	}
	return m
}

// fileDlgSelected 当前选中文件的完整路径（选中目录或未选中返回空）。
func (e *Editor) fileDlgSelected() string {
	if e.fileDlgSel < 0 || e.fileDlgSel >= len(e.fileDlgList) {
		return ""
	}
	en := e.fileDlgList[e.fileDlgSel]
	if en.IsDir {
		return ""
	}
	return filepath.Join(e.fileDlgDir, en.Name)
}

// fileDlgEnter 点击列表项：目录进入 / ".." 上级 / 文件选中。
func (e *Editor) fileDlgEnter(idx int) {
	if idx < 0 || idx >= len(e.fileDlgList) {
		return
	}
	en := e.fileDlgList[idx]
	if en.Name == ".." {
		parent := filepath.Dir(e.fileDlgDir)
		if parent != e.fileDlgDir {
			e.fileDlgDir = parent
			e.fileDlgSel = -1
			e.fileDlgScroll = 0
			e.reloadFileDlg()
		}
		return
	}
	if en.IsDir {
		e.fileDlgDir = filepath.Join(e.fileDlgDir, en.Name)
		e.fileDlgSel = -1
		e.fileDlgScroll = 0
		e.reloadFileDlg()
		return
	}
	e.fileDlgSel = idx
}

// fileDlgConfirm 确认导入当前选中文件。
func (e *Editor) fileDlgConfirm() {
	p := e.fileDlgSelected()
	if p == "" {
		e.SetMessage("请先点击选择一个文件，再点导入")
		return
	}
	e.fileDlgOpen = false
	switch e.fileDlgMode {
	case 0:
		if err := e.ImportOBJModel(p); err != nil {
			e.SetMessage("OBJ 导入失败: %v", err)
		}
	case 1:
		if err := e.ImportModelDoc(p); err != nil {
			e.SetMessage("建模 JSON 导入失败: %v", err)
		}
	case 2:
		if err := e.ImportSprite(p); err != nil {
			e.SetMessage("素材导入失败: %v", err)
		}
	}
}

// drawFileDialog 绘制文件选择对话框。
func (e *Editor) drawFileDialog(c *engine.Canvas) {
	x := (c.W - fileDlgW) / 2
	y := (c.H - fileDlgH) / 2
	c.FillRect(x, y, fileDlgW, fileDlgH, uiColorPanel2)
	c.Rect(x, y, fileDlgW, fileDlgH, uiColorAccent)
	ui.DrawText(c, x+16, y+14, e.fileDlgTitle, uiColorText)
	ui.DrawText(c, x+16, y+40, "目录: "+e.fileDlgDir, uiColorDim)
	// 列表（滚轮滚动）
	visible := e.fileDlgVisibleRows()
	listX := x + 12
	listY := y + fileDlgListY
	sel := e.fileDlgSel
	for i := 0; i < visible; i++ {
		idx := i + e.fileDlgScroll
		if idx >= len(e.fileDlgList) {
			break
		}
		en := e.fileDlgList[idx]
		rowY := listY + i*fileDlgRowH
		if idx == sel {
			c.FillRect(x+8, rowY, fileDlgW-16, fileDlgRowH, uiColorSel)
		}
		icon := "📄"
		if en.IsDir {
			icon = "📁"
		}
		col := uiColorText
		if en.IsDir {
			col = uiColorAccent
		}
		ui.DrawText(c, listX, rowY+7, icon+" "+en.Name, col)
	}
	// 选中预览 / 滚动提示（列表区下方、按钮上方）
	infoY := y + fileDlgH - 62
	if p := e.fileDlgSelected(); p != "" {
		ui.DrawText(c, x+16, infoY, "已选: "+p, uiColorAccent)
	} else if len(e.fileDlgList) > e.fileDlgVisibleRows() {
		ui.DrawText(c, x+16, infoY, "滚轮滚动列表", uiColorDim)
	}
	if len(e.fileDlgList) > e.fileDlgVisibleRows() {
		ui.DrawTextRight(c, x+fileDlgW-16, infoY,
			"滚动 "+fmtItoa(e.fileDlgScroll)+"/"+fmtItoa(e.fileDlgMaxScroll()), uiColorDim)
	}
	// 按钮
	bw, bh := 90, 30
	drawBtn(c, x+fileDlgW-2*bw-30, y+fileDlgBtnY, bw, bh, "导入", false, func() { e.fileDlgConfirm() })
	drawBtn(c, x+fileDlgW-bw-16, y+fileDlgBtnY, bw, bh, "取消", false, func() { e.fileDlgOpen = false })
	ui.DrawText(c, x+16, y+fileDlgH-22, "点击目录进入 · 点击文件选中 · 滚轮滚动", uiColorDim)
}

// fileDialogClick 文件对话框点击：外部关闭；按钮优先；列表行选择/进入。
func (e *Editor) fileDialogClick(x, y int) {
	cw, ch := 1100, 680
	if e.eng != nil && e.eng.Canvas() != nil {
		cw, ch = e.eng.Canvas().W, e.eng.Canvas().H
	}
	dx := (cw - fileDlgW) / 2
	dy := (ch - fileDlgH) / 2
	if x < dx || x >= dx+fileDlgW || y < dy || y >= dy+fileDlgH {
		e.fileDlgOpen = false
		return
	}
	for _, b := range uiBtns {
		if x >= b.x && x < b.x+b.w && y >= b.y && y < b.y+b.h {
			b.cb()
			return
		}
	}
	listY := dy + fileDlgListY
	if y >= listY && y < listY+e.fileDlgVisibleRows()*fileDlgRowH {
		row := (y - listY) / fileDlgRowH
		e.fileDlgEnter(row + e.fileDlgScroll)
	}
}

// fmtItoa 简易整数转字符串。
func fmtItoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

var _ = engine.KeyHelp // 保持 engine import（绘制用 Canvas/Color 时不需要，但防未来裁剪）
