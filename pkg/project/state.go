package project

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"

	"github.com/apex/log"
	"github.com/shalb/cluster.dev/pkg/colors"
	"github.com/shalb/cluster.dev/pkg/config"
	"github.com/shalb/cluster.dev/pkg/utils"
)

func (sp *StateProject) UpdateModule(mod Module) {
	sp.mux.Lock()
	defer sp.mux.Unlock()
	sp.Modules[mod.Key()] = mod
	sp.ChangedModules[mod.Key()] = mod
}

func (sp *StateProject) DeleteModule(mod Module) {
	delete(sp.Modules, mod.Key())
}

type StateProject struct {
	Project
	LoaderProjectPtr *Project
	ChangedModules   map[string]Module
}

func (p *Project) SaveState() error {
	p.mux.Lock()
	defer p.mux.Unlock()
	st := stateData{
		Markers: p.Markers,
		Modules: map[string]interface{}{},
	}
	for key, mod := range p.Modules {
		st.Modules[key] = mod.GetState()
	}
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent(" ", " ")
	err := encoder.Encode(st)
	if err != nil {
		return fmt.Errorf("saving project state: %v", err.Error())
	}
	return ioutil.WriteFile(config.Global.StateFileName, buffer.Bytes(), fs.ModePerm)
}

type stateData struct {
	Markers map[string]interface{} `json:"markers"`
	Modules map[string]interface{} `json:"modules"`
}

func (p *Project) LoadState() (*StateProject, error) {
	if _, err := os.Stat(config.Global.StateCacheDir); os.IsNotExist(err) {
		err := os.Mkdir(config.Global.StateCacheDir, 0755)
		if err != nil {
			return nil, err
		}
	}
	err := removeDirContent(config.Global.StateCacheDir)
	if err != nil {
		return nil, err
	}

	stateD := stateData{}

	loadedStateFile, err := ioutil.ReadFile(config.Global.StateFileName)
	if err == nil {
		err = utils.JSONDecode(loadedStateFile, &stateD)
		if err != nil {
			return nil, err
		}
	}
	statePrj := StateProject{
		Project: Project{
			name:            p.Name(),
			secrets:         p.secrets,
			configData:      p.configData,
			configDataFile:  p.configDataFile,
			objects:         p.objects,
			Modules:         make(map[string]Module),
			Markers:         stateD.Markers,
			Infrastructures: make(map[string]*Infrastructure),
			Backends:        p.Backends,
			codeCacheDir:    config.Global.StateCacheDir,
		},
		LoaderProjectPtr: p,
		ChangedModules:   make(map[string]Module),
	}

	if statePrj.Markers == nil {
		statePrj.Markers = make(map[string]interface{})
	}
	for key, m := range p.Markers {
		statePrj.Markers[key] = m
	}

	for mName, mState := range stateD.Modules {
		log.Debugf("Loading module from state: %v", mName)

		key, exists := mState.(map[string]interface{})["type"]
		if !exists {
			return nil, fmt.Errorf("loading state: internal error: bad module type in state")
		}
		mod, err := ModuleFactoriesMap[key.(string)].NewFromState(mState.(map[string]interface{}), mName, &statePrj)
		if err != nil {
			return nil, fmt.Errorf("loading state: error loading module from state: %v", err.Error())
		}
		statePrj.Modules[mName] = mod
	}
	err = statePrj.prepareModules()
	if err != nil {
		return nil, err
	}
	return &statePrj, nil
}

func (sp *StateProject) CheckModuleChanges(module Module) string {
	// log.Debugf("Check module: %v %+v", module.Key(), sp.Modules)
	moddInState, exists := sp.Modules[module.Key()]
	if !exists {
		return utils.Diff(nil, module.GetDiffData(), true)
	}
	var diffData interface{}
	if module != nil {
		diffData = module.GetDiffData()
	}
	df := utils.Diff(moddInState.GetDiffData(), diffData, true)
	if len(df) > 0 {
		return df
	}
	for _, dep := range *module.Dependencies() {
		if sp.checkModuleChangesRecursive(dep.Module) {
			return colors.Fmt(colors.Yellow).Sprintf("+/- There are changes in the module dependencies.")
		}
	}
	return ""
}

func (sp *StateProject) checkModuleChangesRecursive(module Module) bool {
	// log.Debugf("Check module recu: %v deps: %v", module.Key(), *module.Dependencies())
	modNew, exists := sp.Modules[module.Key()]
	if !exists {
		return true
	}
	var diffData interface{}
	if module != nil {
		diffData = module.GetDiffData()
	}
	df := utils.Diff(diffData, modNew.GetDiffData(), true)
	if len(df) > 0 {
		return true
	}
	// log.Debugf("Check module recu: %v deps: %v", module.Key(), *module.Dependencies())
	for _, dep := range *module.Dependencies() {
		if _, exists := sp.ChangedModules[dep.Module.Key()]; exists {
			return true
		}
		if sp.checkModuleChangesRecursive(dep.Module) {
			return true
		}
	}
	return false
}