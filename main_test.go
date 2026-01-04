package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain_Integration(t *testing.T) {
	// 1. 准备测试路径和临时输出目录
	testDataPath := "D:\\Work\\Code\\go-treesitter-dependency-analyzer\\x\\java\\testdata\\com"
	if _, err := os.Stat(testDataPath); os.IsNotExist(err) {
		t.Skipf("跳过测试：找不到路径 %s", testDataPath)
	}

	tmpOutDir, err := os.MkdirTemp("", "analyzer-main-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpOutDir)

	// 2. 模拟命令行参数
	// 格式: go run main.go -lang=java -path=... -out-dir=... -format=mermaid
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }() // 测试结束后还原参数

	os.Args = []string{
		"cmd",
		"-lang=java",
		"-path=" + testDataPath,
		"-out-dir=" + tmpOutDir,
		"-format=mermaid",
	}

	// 3. 执行 main 函数
	// 注意：如果 main 中调用了 os.Exit，测试进程会崩溃。
	t.Logf("运行分析器，输入路径: %s, 输出路径: %s", testDataPath, tmpOutDir)
	main()

	// 4. 验证生成结果
	mermaidFile := filepath.Join(tmpOutDir, "visualization.html")
	if _, err := os.Stat(mermaidFile); os.IsNotExist(err) {
		t.Errorf("期望生成 Mermaid 文件 %s，但未找到", mermaidFile)
	}
}
