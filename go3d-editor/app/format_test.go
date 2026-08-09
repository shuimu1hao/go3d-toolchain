package app

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"go3d/mesh"
)

// TestSTLBinaryLoad 二进制 STL 导入（2 个三角形）。
func TestSTLBinaryLoad(t *testing.T) {
	// 三角形 1：(0,0,0)(1,0,0)(0,1,0)；三角形 2：(0,0,0)(0,1,0)(0,0,1)
	data := make([]byte, 84+2*50)
	copy(data[:80], "binary stl test")
	binary.LittleEndian.PutUint32(data[80:], 2)
	writeTri := func(base int, v [3][3]float32) {
		for k := 0; k < 3; k++ {
			p := base + 12 + k*12
			binary.LittleEndian.PutUint32(data[p:], mathFloat32bits(v[k][0]))
			binary.LittleEndian.PutUint32(data[p+4:], mathFloat32bits(v[k][1]))
			binary.LittleEndian.PutUint32(data[p+8:], mathFloat32bits(v[k][2]))
		}
	}
	writeTri(84, [3][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}})
	writeTri(84+50, [3][3]float32{{0, 0, 0}, {0, 1, 0}, {0, 0, 1}})

	path := filepath.Join(t.TempDir(), "t.stl")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadSTL(path)
	if err != nil {
		t.Fatal("load:", err)
	}
	if len(m.Faces) != 2 {
		t.Fatalf("expect 2 faces, got %d", len(m.Faces))
	}
	// 共享顶点去重：6 顶点 - 共享 2 = 4 唯一顶点
	if len(m.Positions) != 4 {
		t.Fatalf("expect 4 unique verts, got %d", len(m.Positions))
	}
}

// TestSTLAsciiLoad ASCII STL 导入。
func TestSTLAsciiLoad(t *testing.T) {
	ascii := `solid test
facet normal 0 0 1
  outer loop
    vertex 0 0 0
    vertex 2 0 0
    vertex 0 2 0
  endloop
endfacet
facet normal 0 0 1
  outer loop
    vertex 0 0 0
    vertex 0 2 0
    vertex 0 0 3
  endloop
endfacet
endsolid test
`
	path := filepath.Join(t.TempDir(), "a.stl")
	if err := os.WriteFile(path, []byte(ascii), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadSTL(path)
	if err != nil {
		t.Fatal("load:", err)
	}
	if len(m.Faces) != 2 {
		t.Fatalf("expect 2 faces, got %d", len(m.Faces))
	}
	if len(m.Positions) != 4 {
		t.Fatalf("expect 4 unique verts, got %d", len(m.Positions))
	}
}

// TestGLBRoundTrip GLB 导出 → 导入往返。
func TestGLBRoundTrip(t *testing.T) {
	m := mesh.Cube(1.0)
	path := filepath.Join(t.TempDir(), "cube.glb")
	if err := SaveGLB(path, m); err != nil {
		t.Fatal("save:", err)
	}
	m2, err := LoadGLTF(path)
	if err != nil {
		t.Fatal("load:", err)
	}
	if len(m2.Faces) != len(m.Faces) {
		t.Fatalf("face count mismatch: %d vs %d", len(m2.Faces), len(m.Faces))
	}
	if len(m2.Positions) != len(m.Positions) {
		t.Fatalf("vert count mismatch: %d vs %d", len(m2.Positions), len(m.Positions))
	}
	// 顶点坐标一致
	for i := range m.Positions {
		if m2.Positions[i] != m.Positions[i] {
			t.Fatalf("vert %d mismatch: %v vs %v", i, m2.Positions[i], m.Positions[i])
		}
	}
}

// TestGLTFExternalBuffer .gltf + .bin 外部 buffer 导入。
func TestGLTFExternalBuffer(t *testing.T) {
	dir := t.TempDir()
	// BIN：2 三角形（4 顶点）+ 6 索引
	bin := []byte{}
	verts := [4][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
	for _, v := range verts {
		for _, c := range v {
			var b [4]byte
			binary.LittleEndian.PutUint32(b[:], mathFloat32bits(c))
			bin = append(bin, b[:]...)
		}
	}
	idxs := []uint32{0, 1, 2, 0, 2, 3}
	for _, i := range idxs {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], i)
		bin = append(bin, b[:]...)
	}
	binPath := filepath.Join(dir, "mesh.bin")
	if err := os.WriteFile(binPath, bin, 0o644); err != nil {
		t.Fatal(err)
	}
	gltf := `{
  "asset": {"version": "2.0"},
  "buffers": [{"uri": "mesh.bin", "byteLength": ` + itoa(len(bin)) + `}],
  "bufferViews": [
    {"buffer": 0, "byteOffset": 0, "byteLength": 48},
    {"buffer": 0, "byteOffset": 48, "byteLength": 24}
  ],
  "accessors": [
    {"bufferView": 0, "componentType": 5126, "count": 4, "type": "VEC3"},
    {"bufferView": 1, "componentType": 5125, "count": 6, "type": "SCALAR"}
  ],
  "meshes": [{"name": "tri", "primitives": [
    {"attributes": {"POSITION": 0}, "indices": 1, "mode": 4}
  ]}],
  "scene": 0,
  "scenes": [{"nodes": [0]}],
  "nodes": [{"mesh": 0}]
}`
	gltfPath := filepath.Join(dir, "tri.gltf")
	if err := os.WriteFile(gltfPath, []byte(gltf), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadGLTF(gltfPath)
	if err != nil {
		t.Fatal("load:", err)
	}
	if len(m.Faces) != 2 {
		t.Fatalf("expect 2 faces, got %d", len(m.Faces))
	}
	if len(m.Positions) != 4 {
		t.Fatalf("expect 4 verts, got %d", len(m.Positions))
	}
}

func mathFloat32bits(f float32) uint32 {
	return math.Float32bits(f)
}

// itoa 整数转字符串。
func itoa(n int) string {
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
