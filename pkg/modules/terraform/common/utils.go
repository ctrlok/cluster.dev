package common

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/shalb/cluster.dev/pkg/config"
	"github.com/shalb/cluster.dev/pkg/project"
)

func readDeps(depsData interface{}, infra *project.Infrastructure) ([]*project.Dependency, error) {
	rawDepsList := []string{}
	switch depsData.(type) {
	case string:
		rawDepsList = append(rawDepsList, depsData.(string))
	case []string:
		rawDepsList = append(rawDepsList, depsData.([]string)...)
	}
	var res []*project.Dependency
	for _, dep := range rawDepsList {
		splDep := strings.Split(dep, ".")
		if len(splDep) != 2 {
			return nil, fmt.Errorf("Incorrect module dependency '%v'", dep)
		}
		infNm := splDep[0]
		if infNm == "this" {
			infNm = infra.Name
		}
		res = append(res, &project.Dependency{
			InfraName:  infNm,
			ModuleName: splDep[1],
		})
		log.Debugf("Dep added: %v.%v", infNm, splDep[1])
	}
	return res, nil
}

func readHook(hookData interface{}, hookType string) ([]byte, error) {
	hook, ok := hookData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%s configuration error", hookType)
	}
	cmd, cmdExists := hook["command"].(string)
	script, scrExists := hook["script"].(string)
	if cmdExists && scrExists {
		return nil, fmt.Errorf("Error in %s config, use 'script' or 'command' option, not both", hookType)
	}
	if !cmdExists && !scrExists {
		return nil, fmt.Errorf("Error in %s config, use one of 'script' or 'command' option", hookType)
	}
	var ScriptData []byte
	var err error
	if cmdExists {
		ScriptData = []byte(fmt.Sprintf("#!/usr/bin/env bash\nset -e\n\n%s", cmd))
	} else {
		ScriptData, err = ioutil.ReadFile(filepath.Join(config.Global.WorkingDir, script))
		if err != nil {
			return nil, fmt.Errorf("can't load %s script: %v", hookType, err.Error())
		}
	}
	return ScriptData, nil

}
