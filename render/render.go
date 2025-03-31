package render

import (
	"bytes"
	"fmt"
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

func (r *Render) RenderKpiCalc(sliceID string) (content []byte, err error) {
	v := KpiCalc{SliceID: sliceID}

	return r.render("kpi_calculator.yaml.tpl", v)
}

func (r *Render) RenderMde(sliceID string) (content []byte, err error) {
	v := MdeValue{SliceID: sliceID}

	return r.render("metrics-servicemonitor.yaml.tpl", v)
}

func (r *Render) RenderSlice(slice model.SliceAndAddress) (contents [][]byte, err error) {
	_, _, smfcv, smfdv, smfsv, upfcv, upfdv := sliceToValue(slice)

	//从value中生成kubernetes配置文件

	// 定义各资源对应的模板和输出文件
	resources := []struct {
		templateFile string
		// outputFile   string
		data interface{}
	}{
		// {"smf-configmap.yaml.tpl", "smf-configmap.yaml", smfcv},
		// {"smf-deployment.yaml.tpl", "smf-deployment.yaml", smfdv},
		// {"smf-service.yaml.tpl", "smf-service.yaml", smfsv},
		// {"upf-configmap.yaml.tpl", "upf-configmap.yaml", upfcv},
		// {"upf-deployment.yaml.tpl", "upf-deployment.yaml", upfdv},
		{"smf-configmap.yaml.tpl", smfcv},
		{"smf-deployment.yaml.tpl", smfdv},
		{"smf-service.yaml.tpl", smfsv},
		{"upf-configmap.yaml.tpl", upfcv},
		{"upf-deployment.yaml.tpl", upfdv},
	}

	for _, res := range resources {
		content, err := r.render(res.templateFile, res.data)
		if err != nil {
			return nil, fmt.Errorf("渲染失败[%s]: %w", res.templateFile, err)
		}

		contents = append(contents, content)
	}

	return
}

func (r *Render) render(tplFile string, value any) ([]byte, error) {
	// 构造模板文件路径
	tplPath := filepath.Join(r.config.TemplatePath, tplFile)

	// 读取模板内容
	tplContent, err := os.ReadFile(tplPath)
	if err != nil {
		return nil, fmt.Errorf("读取模板文件 %s 失败: %v", tplPath, err)

	}

	// 解析模板
	tmpl, err := template.New(tplFile).Parse(string(tplContent))
	if err != nil {
		return nil, fmt.Errorf("解析模板失败[%s]: %w", tplFile, err)
	}

	// 渲染到内存缓冲区
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, value); err != nil {
		return nil, fmt.Errorf("渲染失败[%s]: %w", tplFile, err)
	}

	return buf.Bytes(), nil
}

func sliceToValue(ws model.SliceAndAddress) (
	sv SliceValue,
	sevs SessionValues,
	smfcv SmfConfigmapValue,
	smfdv SmfDeploymentValue,
	smfsv SmfServiceValue,
	upfcv UpfConfigmapValue,
	upfdv UpfDeploymentValue,
) {
	sv.ID = ws.SliceID()
	sv.SST = strconv.Itoa(ws.SST)
	sv.SD = ws.SD

	if len(ws.Sessions) != len(ws.SessionSubnets) {
		log.Printf("会话数和会话子网数不一致")
		return
	}

	for idx, session := range ws.Sessions {
		sev := SessionValue{
			DNN:    session.Name,
			Subnet: ws.SessionSubnets[idx],
			Dev:    "ogstun" + strconv.Itoa(idx),
		}
		sevs = append(sevs, sev)
	}

	smfcv.SliceValue = sv
	smfcv.UPFN4Addr = ws.UPFN4Addr
	smfcv.SessionValues = sevs

	smfdv.SliceValue = sv
	smfdv.N4Addr = ws.SMFN4Addr
	smfdv.N3Addr = ws.SMFN3Addr

	smfsv.SliceValue = sv

	upfcv.SliceValue = sv
	upfcv.SessionValues = sevs

	upfdv.SliceValue = sv
	upfdv.N4Addr = ws.UPFN4Addr
	upfdv.N3Addr = ws.UPFN3Addr

	return
}
