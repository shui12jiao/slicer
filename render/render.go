package render

import (
	"log"
	"os"
	"path/filepath"
	"slicer/model"
	"slicer/util"
	"strconv"
	"text/template"
)

type Render struct {
	// slice转化为kubernetes配置文件
	config util.Config
}

func NewRender(config util.Config) *Render {
	return &Render{
		config: config,
	}

}

func (r *Render) SliceToKube(slice model.Slice) (dirPath string, err error) {
	_, _, smfcv, smfdv, smfsv, upfcv, upfdv := sliceToValue(slice)

	//从value中生成kubernetes配置文件
	dirPath = filepath.Join(r.config.KubePath, slice.ID())
	if err = os.MkdirAll(dirPath, 0755); err != nil {
		log.Printf("创建目录失败: %v", err)
		return
	}

	// 定义各资源对应的模板和输出文件
	resources := []struct {
		templateFile string
		outputFile   string
		data         interface{}
	}{
		{"smf-configmap.yaml.tpl", "smf-configmap.yaml", smfcv},
		{"smf-deployment.yaml.tpl", "smf-deployment.yaml", smfdv},
		{"smf-service.yaml.tpl", "smf-service.yaml", smfsv},
		{"upf-configmap.yaml.tpl", "upf-configmap.yaml", upfcv},
		{"upf-deployment.yaml.tpl", "upf-deployment.yaml", upfdv},
	}

	for _, res := range resources {
		// 构造模板文件路径
		tplPath := filepath.Join(r.config.TemplatePath, res.templateFile)
		// 读取模板内容
		tplContent, err := os.ReadFile(tplPath)
		if err != nil {
			log.Printf("读取模板文件 %s 失败: %v", tplPath, err)
			break
		}
		// 解析模板
		tmpl, err := template.New(res.outputFile).Parse(string(tplContent))
		if err != nil {
			log.Printf("解析模板 %s 失败: %v", res.templateFile, err)
			break
		}
		// 创建输出文件
		outputPath := filepath.Join(r.config.KubePath, res.outputFile)
		f, err := os.Create(outputPath)
		if err != nil {
			log.Printf("创建输出文件 %s 失败: %v", outputPath, err)
			break
		}
		defer f.Close()
		// 渲染模板
		if err = tmpl.Execute(f, res.data); err != nil {
			log.Printf("渲染模板 %s 失败: %v", res.templateFile, err)
			break
		}
	}

	return
}

func sliceToValue(slice model.Slice) (
	sv SliceValue,
	sevs SessionValues,
	smfcv SmfConfigmapValue,
	smfdv SmfDeploymentValue,
	smfsv SmfServiceValue,
	upfcv UpfConfigmapValue,
	upfdv UpfDeploymentValue,
) {
	sv.ID = slice.ID()
	sv.SST = strconv.Itoa(slice.SST)
	sv.SD = slice.SD

	for _, session := range slice.Sessions {
		sev := SessionValue{
			DNN: session.Name,
			// SessionSubnet: , //TODO 从ipam中获取
			// Dev:           ,           //TODO
		}
		sevs = append(sevs, sev)
	}

	smfcv.SliceValue = sv
	// smfcv.UPFAddr = , //TODO
	smfcv.SessionValues = sevs

	smfdv.SliceValue = sv
	// smfdv.N4Addr = , //TODO
	// smfdv.N3Addr = , //TODO

	smfsv.SliceValue = sv

	upfcv.SliceValue = sv
	upfcv.SessionValues = sevs

	upfdv.SliceValue = sv
	// upfdv.N4Addr = , //TODO
	// upfdv.N3Addr = , //TODO

	return
}
