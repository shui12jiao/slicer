package util

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"text/template"
)

// RenderTemplateFromFile 从指定的模板文件渲染内容
func RenderTemplateFromFile(tplFile string, value any) ([]byte, error) {
	// 读取模板内容
	tplContent, err := os.ReadFile(tplFile)
	if err != nil {
		slog.Error("读取模板文件失败", "error", err, "file", tplFile)
		return nil, fmt.Errorf("读取模板文件 %s 失败: %v", tplFile, err)
	}

	// 解析模板
	tmpl, err := template.New(tplFile).Parse(string(tplContent))
	if err != nil {
		slog.Error("解析模板失败", "error", err, "file", tplFile)
		return nil, fmt.Errorf("解析模板失败[%s]: %w", tplFile, err)
	}

	// 渲染到内存缓冲区
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, value); err != nil {
		slog.Error("执行渲染失败", "error", err, "file", tplFile, "vars", value)
		return nil, fmt.Errorf("渲染失败[%s]: %w", tplFile, err)
	}

	return buf.Bytes(), nil
}
