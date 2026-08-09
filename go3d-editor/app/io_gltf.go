package app

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"go3d/math3d"
	"go3d/mesh"
)

// ---------- GLTF/GLB 导入 ----------

// gltfAccessor accessor 定义。
type gltfAccessor struct {
	BufferView    int `json:"bufferView"`
	ComponentType int `json:"componentType"`
	Count         int `json:"count"`
	Type          string `json:"type"`
}

// gltfJSON glTF 2.0 JSON 结构（只解析几何必需字段）。
type gltfJSON struct {
	Buffers []struct {
		URI        string `json:"uri"`
		ByteLength int    `json:"byteLength"`
	} `json:"buffers"`
	BufferViews []struct {
		Buffer     int `json:"buffer"`
		ByteOffset int `json:"byteOffset"`
		ByteLength int `json:"byteLength"`
	} `json:"bufferViews"`
	Accessors []gltfAccessor `json:"accessors"`
	Meshes []struct {
		Name       string `json:"name"`
		Primitives []struct {
			Attributes struct {
				Position int `json:"POSITION"`
			} `json:"attributes"`
			Indices int `json:"indices"`
			Mode    int `json:"mode"`
		} `json:"primitives"`
	} `json:"meshes"`
	Scene int `json:"scene"`
}

// LoadGLTF 导入 glTF 文件（.gltf 或 .glb，自动识别）。
// 返回第一个 mesh（合并其 primitive）。
func LoadGLTF(path string) (*mesh.Mesh, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".glb") || (len(data) >= 4 && string(data[0:4]) == "glTF") {
		return loadGLB(data, path)
	}
	return loadGLTFJSON(data, filepath.Dir(path), path)
}

// loadGLB 解析 GLB 二进制容器。
func loadGLB(data []byte, path string) (*mesh.Mesh, error) {
	if len(data) < 12 || string(data[0:4]) != "glTF" {
		return nil, fmt.Errorf("无效 GLB 文件头")
	}
	version := binary.LittleEndian.Uint32(data[4:8])
	if version != 2 {
		return nil, fmt.Errorf("不支持 GLB 版本 %d", version)
	}
	// 遍历 chunk
	off := 12
	var jsonBytes, binData []byte
	for off+8 <= len(data) {
		chLen := int(binary.LittleEndian.Uint32(data[off:]))
		chType := binary.LittleEndian.Uint32(data[off+4:])
		chData := data[off+8 : off+8+chLen]
		switch chType {
		case 0x4E4F534A: // JSON
			jsonBytes = chData
		case 0x004E4942: // BIN
			binData = chData
		}
		off += 8 + chLen
		// 4 字节对齐
		off += (4 - (chLen % 4)) % 4
	}
	if jsonBytes == nil {
		return nil, fmt.Errorf("GLB 缺少 JSON chunk")
	}
	return parseGLTFMesh(jsonBytes, binData, filepath.Dir(path), path)
}

// loadGLTFJSON 解析 .gltf（buffer 走 data URI 或外部文件）。
func loadGLTFJSON(data []byte, dir, path string) (*mesh.Mesh, error) {
	// 收集所有 buffer 数据
	var doc gltfJSON
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("GLTF JSON 解析失败: %v", err)
	}
	bufs := make([][]byte, len(doc.Buffers))
	for i, b := range doc.Buffers {
		if b.URI == "" {
			continue
		}
		bd, err := resolveBuffer(b.URI, dir)
		if err != nil {
			return nil, err
		}
		bufs[i] = bd
	}
	// 内联 buffer（无 URI，用整个 JSON 后无数据→空；GLB 场景已走 binData）
	_ = path
	return gltfToMesh(&doc, bufs)
}

// resolveBuffer 解析 buffer URI：data:base64 或外部文件。
func resolveBuffer(uri, dir string) ([]byte, error) {
	if strings.HasPrefix(uri, "data:") {
		comma := strings.Index(uri, ",")
		if comma < 0 {
			return nil, fmt.Errorf("坏 data URI")
		}
		meta := uri[:comma]
		if strings.Contains(meta, "base64") {
			return base64.StdEncoding.DecodeString(uri[comma+1:])
		}
		return []byte(uri[comma+1:]), nil
	}
	// 外部文件（相对 gltf 目录）
	return os.ReadFile(filepath.Join(dir, uri))
}

// gltfToMesh 从解析后的 gltf 提取网格。
func gltfToMesh(doc *gltfJSON, bufs [][]byte) (*mesh.Mesh, error) {
	if len(doc.Meshes) == 0 {
		return nil, fmt.Errorf("GLTF 无网格")
	}
	// 合并第一个 mesh 的所有 primitive
	col := mesh.Col(190, 180, 160)
	m := &mesh.Mesh{}
	vertOff := 0
	for _, prim := range doc.Meshes[0].Primitives {
		if prim.Mode != 0 && prim.Mode != 4 {
			continue // 只要 TRIANGLES(4)/TRIANGLE_STRIP(5 忽略)
		}
		posAcc := prim.Attributes.Position
		if posAcc < 0 || posAcc >= len(doc.Accessors) {
			continue
		}
		acc := doc.Accessors[posAcc]
		verts, err := readAccessorFloats(doc, bufs, acc)
		if err != nil {
			return nil, err
		}
		// 顶点追加
		start := len(m.Positions)
		for _, v := range verts {
			m.Positions = append(m.Positions, v)
		}
		// 索引
		if prim.Indices >= 0 && prim.Indices < len(doc.Accessors) {
			idxs, err := readAccessorUints(doc, bufs, doc.Accessors[prim.Indices])
			if err == nil {
				for i := 0; i+2 < len(idxs); i += 3 {
					m.Faces = append(m.Faces, mesh.Face{A: start + int(idxs[i]), B: start + int(idxs[i+1]), C: start + int(idxs[i+2]), Col: col})
				}
				continue
			}
		}
		// 无索引：顺序三角形
		for i := 0; i+2 < len(m.Positions)-start; i += 3 {
			m.Faces = append(m.Faces, mesh.Face{A: start + i, B: start + i + 1, C: start + i + 2, Col: col})
		}
		_ = vertOff
	}
	if len(m.Faces) == 0 {
		return nil, fmt.Errorf("GLTF 无三角形面")
	}
	return m, nil
}

// readAccessorFloats 读 VEC3 FLOAT 顶点。
func readAccessorFloats(doc *gltfJSON, bufs [][]byte, acc gltfAccessor) ([]math3d.Vec3, error) {
	if acc.ComponentType != 5126 || acc.Type != "VEC3" {
		return nil, fmt.Errorf("POSITION 需 FLOAT VEC3")
	}
	buf, off, err := accessorBuffer(doc, bufs, acc.BufferView)
	if err != nil {
		return nil, err
	}
	out := make([]math3d.Vec3, acc.Count)
	for i := 0; i < acc.Count; i++ {
		p := off + i*12
		if p+12 > len(buf) {
			return nil, fmt.Errorf("顶点越界")
		}
		out[i] = math3d.Vec3{
			X: float32frombits(binary.LittleEndian.Uint32(buf[p:])),
			Y: float32frombits(binary.LittleEndian.Uint32(buf[p+4:])),
			Z: float32frombits(binary.LittleEndian.Uint32(buf[p+8:])),
		}
	}
	return out, nil
}

// readAccessorUints 读索引（USHORT/UINT/UBYTE）。
func readAccessorUints(doc *gltfJSON, bufs [][]byte, acc gltfAccessor) ([]uint32, error) {
	if acc.Type != "SCALAR" {
		return nil, fmt.Errorf("索引需 SCALAR")
	}
	buf, off, err := accessorBuffer(doc, bufs, acc.BufferView)
	if err != nil {
		return nil, err
	}
	out := make([]uint32, acc.Count)
	stride := 1
	switch acc.ComponentType {
	case 5121: // UBYTE
		stride = 1
	case 5123: // USHORT
		stride = 2
	case 5125: // UINT
		stride = 4
	default:
		return nil, fmt.Errorf("不支持索引类型 %d", acc.ComponentType)
	}
	for i := 0; i < acc.Count; i++ {
		p := off + i*stride
		if p+stride > len(buf) {
			return nil, fmt.Errorf("索引越界")
		}
		switch stride {
		case 1:
			out[i] = uint32(buf[p])
		case 2:
			out[i] = uint32(binary.LittleEndian.Uint16(buf[p:]))
		case 4:
			out[i] = binary.LittleEndian.Uint32(buf[p:])
		}
	}
	return out, nil
}

// accessorBuffer 返回 accessor 的 bufferView 数据与偏移。
func accessorBuffer(doc *gltfJSON, bufs [][]byte, bv int) ([]byte, int, error) {
	if bv < 0 || bv >= len(doc.BufferViews) {
		return nil, 0, fmt.Errorf("bufferView 越界")
	}
	v := doc.BufferViews[bv]
	if v.Buffer < 0 || v.Buffer >= len(bufs) || bufs[v.Buffer] == nil {
		return nil, 0, fmt.Errorf("buffer 数据缺失")
	}
	if v.ByteOffset+v.ByteLength > len(bufs[v.Buffer]) {
		return nil, 0, fmt.Errorf("bufferView 越界")
	}
	return bufs[v.Buffer], v.ByteOffset, nil
}

// float32frombits 便捷。
func float32frombits(b uint32) float32 { return math.Float32frombits(b) }

// parseGLTFMesh GLB 场景：解析 JSON + 内联 bin。
func parseGLTFMesh(jsonBytes, binData []byte, dir, path string) (*mesh.Mesh, error) {
	var doc gltfJSON
	if err := json.Unmarshal(jsonBytes, &doc); err != nil {
		return nil, fmt.Errorf("GLB JSON 解析失败: %v", err)
	}
	// 内联 buffer（GLB 的 bin chunk 是 buffer[0]）
	bufs := make([][]byte, len(doc.Buffers))
	for i, b := range doc.Buffers {
		if b.URI == "" {
			if i == 0 {
				bufs[i] = binData
			}
			continue
		}
		bd, err := resolveBuffer(b.URI, dir)
		if err != nil {
			return nil, err
		}
		bufs[i] = bd
	}
	_ = path
	return gltfToMesh(&doc, bufs)
}

// ---------- GLB 导出 ----------

// SaveGLB 把网格导出为 GLB（glTF 2.0 二进制容器，内联 buffer）。
func SaveGLB(path string, m *mesh.Mesh) error {
	// BIN 数据：顶点（FLOAT VEC3）+ 索引（UINT）
	bin := []byte{}
	posOff := 0
	for _, p := range m.Positions {
		var b [12]byte
		binary.LittleEndian.PutUint32(b[0:], math.Float32bits(p.X))
		binary.LittleEndian.PutUint32(b[4:], math.Float32bits(p.Y))
		binary.LittleEndian.PutUint32(b[8:], math.Float32bits(p.Z))
		bin = append(bin, b[:]...)
	}
	posLen := len(m.Positions) * 12
	idxOff := posLen
	idxLen := len(m.Faces) * 3 * 4
	for i := range m.Faces {
		f := &m.Faces[i]
		var b [12]byte
		binary.LittleEndian.PutUint32(b[0:], uint32(f.A))
		binary.LittleEndian.PutUint32(b[4:], uint32(f.B))
		binary.LittleEndian.PutUint32(b[8:], uint32(f.C))
		bin = append(bin, b[:]...)
	}
	// 4 字节对齐
	for len(bin)%4 != 0 {
		bin = append(bin, 0)
	}
	// JSON
	doc := map[string]any{
		"asset": map[string]any{"version": "2.0", "generator": "go3d"},
		"scene": 0,
		"scenes": []any{map[string]any{"nodes": []any{0}}},
		"nodes": []any{map[string]any{"mesh": 0, "name": "model"}},
		"meshes": []any{map[string]any{
			"name": "mesh",
			"primitives": []any{map[string]any{
				"attributes": map[string]any{"POSITION": 0},
				"indices":    1,
				"mode":       4,
			}},
		}},
		"accessors": []any{
			map[string]any{"bufferView": 0, "componentType": 5126, "count": len(m.Positions), "type": "VEC3"},
			map[string]any{"bufferView": 1, "componentType": 5125, "count": len(m.Faces) * 3, "type": "SCALAR"},
		},
		"bufferViews": []any{
			map[string]any{"buffer": 0, "byteOffset": posOff, "byteLength": posLen},
			map[string]any{"buffer": 0, "byteOffset": idxOff, "byteLength": idxLen},
		},
		"buffers": []any{map[string]any{"byteLength": len(bin)}},
	}
	jsonData, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	// GLB 组装
	out := []byte{}
	hdr := make([]byte, 12)
	copy(hdr, "glTF")
	binary.LittleEndian.PutUint32(hdr[4:], 2)
	total := 12 + 8 + len(jsonData) + 8 + len(bin)
	// JSON chunk 4 字节对齐
	for (8+len(jsonData))%4 != 0 {
		jsonData = append(jsonData, 0x20)
		total++
	}
	for (8+len(bin))%4 != 0 {
		bin = append(bin, 0)
		total++
	}
	binary.LittleEndian.PutUint32(hdr[8:], uint32(total))
	out = append(out, hdr...)
	// JSON chunk
	var ch0 [8]byte
	binary.LittleEndian.PutUint32(ch0[0:], uint32(len(jsonData)))
	binary.LittleEndian.PutUint32(ch0[4:], 0x4E4F534A)
	out = append(out, ch0[:]...)
	out = append(out, jsonData...)
	// BIN chunk
	var ch1 [8]byte
	binary.LittleEndian.PutUint32(ch1[0:], uint32(len(bin)))
	binary.LittleEndian.PutUint32(ch1[4:], 0x004E4942)
	out = append(out, ch1[:]...)
	out = append(out, bin...)
	return os.WriteFile(path, out, 0o644)
}
