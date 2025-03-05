package render

import "slicer/model"

const TemplatePath = "template"
const KustomizePath = "kustomize"

type Render interface {
	// slice转化为kustomize配置文件
	SliceToKustomize(slice model.Slice, filename string) ([]byte, error)
}
