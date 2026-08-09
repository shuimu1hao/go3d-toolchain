package app

// ---------- 插件扩展点（未来深入发展预留） ----------
//
// 插件机制：编译时注册（纯 Go 无 cgo，无法 dlopen）。
// 新插件在 main.go 里 RegisterPlugin(MyPlugin{}) 注册，
// 编辑器工具栏"插件"按钮弹出插件面板，列出各插件的动作。
// 扩展方向：导入器/导出器、建模命令、渲染增强、脚本执行等。

// PluginAction 是插件提供的一个可执行动作（插件面板菜单项）。
type PluginAction struct {
	Label  string // 显示名
	Action func(e *Editor)
}

// Plugin 是编辑器插件接口（未来扩展点）。
type Plugin interface {
	Name() string                 // 插件名（显示在插件面板）
	Actions() []PluginAction      // 动作列表
}

// pluginRegistry 全局插件注册表。
var pluginRegistry []Plugin

// RegisterPlugin 注册插件（main 启动时调用）。
func RegisterPlugin(p Plugin) {
	pluginRegistry = append(pluginRegistry, p)
}

// Plugins 返回已注册插件（nil 安全）。
func Plugins() []Plugin { return pluginRegistry }

// ---------- 内置示例插件 ----------

// StatsPlugin 示例插件：场景统计。
type StatsPlugin struct{}

func (StatsPlugin) Name() string { return "统计" }

func (StatsPlugin) Actions() []PluginAction {
	return []PluginAction{
		{Label: "场景统计", Action: func(e *Editor) {
			objs := len(e.doc.Objs)
			tris := 0
			verts := 0
			for _, o := range e.doc.Objs {
				m := o.RenderMesh()
				if m != nil {
					tris += len(m.Faces)
					verts += len(m.Positions)
				}
			}
			e.SetMessage("插件[统计]: %d 对象 / %d 三角形 / %d 顶点", objs, tris, verts)
		}},
	}
}
