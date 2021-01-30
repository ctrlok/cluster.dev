package kubernetes

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/shalb/cluster.dev/pkg/hcltools"
	"github.com/shalb/cluster.dev/pkg/modules/terraform/common"
	"github.com/shalb/cluster.dev/pkg/project"
	"github.com/zclconf/go-cty/cty"
)

type kubernetes struct {
	common.Module
	source string
	inputs map[string]interface{}
}

func (m *kubernetes) ModKindKey() string {
	return "kubernetes"
}

func (m *kubernetes) GenMainCodeBlock(mod project.Module) ([]byte, error) {
	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()
	providerBlock := rootBody.AppendNewBlock("provider", []string{"kubernetes-alpha"})
	providerBody := providerBlock.Body()
	providerBody.SetAttributeValue("config_path", cty.StringVal("~/.kube/config"))
	log.Debugf("%+v", m.inputs)
	for key, manifest := range m.inputs {
		moduleBlock := rootBody.AppendNewBlock("resource", []string{"kubernetes_manifest", key})
		moduleBody := moduleBlock.Body()
		moduleBody.SetAttributeValue("provider", cty.StringVal("kubernetes-alpha"))
		ctyVal, err := hcltools.InterfaceToCty(manifest)
		if err != nil {
			return nil, err
		}
		moduleBody.SetAttributeValue("manifest", ctyVal)
		depMarkers, ok := m.ProjectPtr().Markers["remoteStateMarkerCatName"]
		if ok {
			for hash, marker := range depMarkers.(map[string]*project.Dependency) {
				if marker.Module == nil {
					continue
				}
				remoteStateRef := fmt.Sprintf("data.terraform_remote_state.%s-%s.outputs.%s", marker.Module.InfraName(), marker.Module.Name(), marker.Output)
				hcltools.ReplaceStingMarkerInBody(moduleBody, hash, remoteStateRef)
			}
		}
	}

	return f.Bytes(), nil
}

// genOutputsBlock generate output code block for this module.
func (m *kubernetes) GenOutputs(mod project.Module) ([]byte, error) {
	if len(m.ExpectedOutputs()) > 0 {
		return nil, fmt.Errorf("kubernetes module has no outputs, you cannot use references to it remote states in other modules")
	}
	return nil, nil

}

func (m *kubernetes) ReadConfig(spec map[string]interface{}) error {

	source, ok := spec["source"].(string)
	if !ok {
		return fmt.Errorf("Incorrect module source")
	}
	tmplDir := filepath.Dir(m.InfraPtr().TemplateSrc)
	var absSource string
	if source[1:2] == "/" {
		absSource = filepath.Join(tmplDir, source)
	} else {
		absSource = source
	}
	fileInfo, err := os.Stat(absSource)
	if err != nil {
		return err
	}
	var filesList []string
	if fileInfo.IsDir() {
		filesList, err = filepath.Glob(absSource + "/*.yaml")
		if err != nil {
			return err
		}
	} else {
		filesList = append(filesList, absSource)
	}
	for _, fileName := range filesList {
		file, err := ioutil.ReadFile(fileName)
		if err != nil {
			return err
		}
		manifest, err := m.InfraPtr().DoTemplate(file)
		if err != nil {
			return err
		}
		manifests, err := project.ReadYAMLObjects(manifest)
		if err != nil {
			return err
		}

		for i, manifest := range manifests {
			key := project.ConvertToTfVarName(strings.TrimSuffix(filepath.Base(fileName), ".yaml"))
			key = fmt.Sprintf("%s_%v", key, i)
			m.inputs[key] = manifest
		}
	}
	if len(m.inputs) < 1 {
		return fmt.Errorf("the kubernetes module must contain at least one manifest")
	}
	m.source = source
	return nil
}

// ReplaceMarkers replace all templated markers with values.
func (m *kubernetes) ReplaceMarkers() error {
	err := project.ScanMarkers(m.inputs, m.YamlBlockMarkerScanner, m)
	if err != nil {
		return err
	}
	err = project.ScanMarkers(m.inputs, m.RemoteStatesScanner, m)
	if err != nil {
		return err
	}
	return nil
}