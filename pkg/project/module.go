package project

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/apex/log"

	json "github.com/json-iterator/go"
	"github.com/rodaine/hclencoder"
)

type Module struct {
	InfraPtr     *Infrastructure
	ProjectPtr   *Project
	BackendPtr   Backend
	Name         string
	Type         string
	Source       string
	Inputs       map[string]interface{}
	Dependencies []Dependency
}

type Dependency struct {
	Module *Module
	Output string
}

func (m *Module) GenMainCodeBlockHCL() ([]byte, error) {
	type ModuleVars map[string]interface{}

	type HCLModule struct {
		Name       string `hcl:",key"`
		ModuleVars `hcl:",squash"`
	}
	type Config struct {
		Mod HCLModule `hcl:"module"`
	}

	inp, err := json.Marshal(m.Inputs)
	if err != nil {
		log.Fatal(err.Error())
	}
	unmInputs := ModuleVars{}
	err = json.Unmarshal(inp, &unmInputs)
	if err != nil {
		log.Fatal(err.Error())
	}

	unmInputs["source"] = m.Source
	mod := HCLModule{
		Name:       m.Name,
		ModuleVars: unmInputs,
	}

	input := Config{
		Mod: mod,
	}
	return hclencoder.Encode(input)

}

func (m *Module) GenBackendCodeBlockHCL() ([]byte, error) {

	res, err := m.BackendPtr.GetBackendHCL(*m)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (m *Module) GetDepsRemoteStatesHCL() ([]byte, error) {
	var res []byte
	for _, dep := range m.Dependencies {
		rs, err := dep.Module.GetRemoteStateToSelfHCL()
		if err != nil {
			return nil, err
		}
		res = append(res, rs...)
	}
	return res, nil
}

func (m *Module) GetRemoteStateToSelfHCL() ([]byte, error) {
	return m.BackendPtr.GetBackendHCL(*m)
}

func (m *Module) checkDependMarker(data reflect.Value) (reflect.Value, error) {
	subVal := reflect.ValueOf(data.Interface())
	resString := subVal.String()
	for key, marker := range m.ProjectPtr.DependencyMarkers {
		if strings.Contains(resString, key) {
			if marker.InfraName == "this" {
				marker.InfraName = m.InfraPtr.Name
			}
			modKey := fmt.Sprintf("%s.%s", marker.InfraName, marker.ModuleName)
			depModule, exists := m.ProjectPtr.Modules[modKey]
			if !exists {
				return reflect.ValueOf(nil), fmt.Errorf("Depend module does not exists. Src: '%s.%s', depend: '%s'", m.InfraPtr.Name, m.Name, modKey)
			}
			m.Dependencies = append(m.Dependencies, Dependency{
				Module: depModule,
				Output: marker.Output,
			})
			remoteStateRef := fmt.Sprintf("${data.terraform_remote_state.%s-%s.%s}", marker.InfraName, marker.ModuleName, marker.Output)
			// log.Debugf("Module: %v\nDep: %v", depModule, remoteStateRef)
			replacer := strings.NewReplacer(key, remoteStateRef)
			resString = replacer.Replace(resString)
			return reflect.ValueOf(resString), nil
		}
	}
	return subVal, nil
}

func (m *Module) checkYAMLBlockMarker(data reflect.Value) (reflect.Value, error) {
	subVal := reflect.ValueOf(data.Interface())
	for hash := range m.ProjectPtr.InsertYAMLMarkers {
		if subVal.String() == hash {
			return reflect.ValueOf(m.ProjectPtr.InsertYAMLMarkers[hash]), nil
		}
	}
	return subVal, nil
}

// ReplaceMarkers replace all templated markers with values.
func (m *Module) ReplaceMarkers() error {
	err := processingMarkers(m.Inputs, m.checkYAMLBlockMarker)
	if err != nil {
		return err
	}
	err = processingMarkers(m.Inputs, m.checkDependMarker)
	if err != nil {
		return err
	}
	return nil
}