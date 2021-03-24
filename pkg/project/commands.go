package project

import (
	"fmt"
	"os"

	"github.com/apex/log"
	"github.com/shalb/cluster.dev/pkg/colors"
	"github.com/shalb/cluster.dev/pkg/config"
	"github.com/shalb/cluster.dev/pkg/utils"
)

// Build generate all terraform code for project.
func (p *Project) Build() error {
	for _, module := range p.Modules {
		if err := module.Build(p.codeCacheDir); err != nil {
			return err
		}
	}

	return nil
}

// Destroy all modules.
func (p *Project) Destroy() error {
	fProject, err := p.LoadState()
	if err != nil {
		return err
	}
	grph := grapher{}
	if config.Global.Force {
		grph.Init(p, 1, true)
	} else {
		grph.Init(&fProject.Project, 1, true)
	}

	for _, md := range grph.GetSequenceSet() {
		log.Infof(colors.Fmt(colors.LightWhiteBold).Sprintf("Destroying module '%v'", md.Key()))
		err := md.Build(md.ProjectPtr().codeCacheDir)
		if err != nil {
			log.Errorf("project destroy: destroying deleted module: %v", err.Error())
		}
		err = md.Destroy()
		if err != nil {
			return fmt.Errorf("project destroy: %v", err.Error())
		}
		fProject.DeleteModule(md)
		err = fProject.SaveState()
		if err != nil {
			return fmt.Errorf("project destroy: saving state: %v", err.Error())
		}
	}
	os.Remove(config.Global.StateFileName)
	return nil
}

// Apply all modules.
func (p *Project) Apply() error {

	grph := grapher{}
	grph.Init(p, config.Global.MaxParallel, false)

	fProject, err := p.LoadState()
	if err != nil {
		return err
	}

	StateDestroyGrph := grapher{}
	err = StateDestroyGrph.Init(&fProject.Project, 1, true)
	if err != nil {
		return err
	}

	for _, md := range StateDestroyGrph.GetSequenceSet() {
		_, exists := p.Modules[md.Key()]
		if exists {
			continue
		}
		err := md.Build(md.ProjectPtr().codeCacheDir)
		if err != nil {
			log.Errorf("project apply: destroying deleted module: %v", err.Error())
		}
		err = md.Destroy()
		if err != nil {
			return fmt.Errorf("project apply: destroying deleted module: %v", err.Error())
		}
		fProject.DeleteModule(md)
		err = fProject.SaveState()
		if err != nil {
			return fmt.Errorf("project apply: saving state: %v", err.Error())
		}
	}

	for {
		if grph.Len() == 0 {
			p.SaveState()
			return nil
		}
		md, fn, err := grph.GetNextAsync()
		if err != nil {
			log.Errorf("error in module %v, waiting for all running modules done.", md.Key())
			grph.Wait()
			return fmt.Errorf("error in module %v:\n%v", md.Key(), err.Error())
		}
		if md == nil {
			p.SaveState()
			return nil
		}

		go func(mod Module, finFunc func(error), stateP *StateProject) {
			diff := stateP.CheckModuleChanges(mod)
			var res error
			if len(diff) > 0 || config.Global.Force {
				log.Infof(colors.Fmt(colors.LightWhiteBold).Sprintf("Applying module '%v':", md.Key()))
				err := mod.Build(mod.ProjectPtr().codeCacheDir)
				if err != nil {
					log.Errorf("project apply: module build error: %v", err.Error())
				}
				res := mod.Apply()
				if res == nil {
					stateP.UpdateModule(mod)
					err := stateP.SaveState()
					if err != nil {
						finFunc(err)
						return
					}
				}
				finFunc(res)
				return
			}
			log.Infof(colors.Fmt(colors.LightWhiteBold).Sprintf("Module '%v' has not changed. Skip applying.", md.Key()))
			finFunc(res)
		}(md, fn, fProject)
	}
}

// Plan and output result.
func (p *Project) Plan() error {
	fProject, err := p.LoadState()
	if err != nil {
		return err
	}

	CurrentGrph := grapher{}
	err = CurrentGrph.Init(p, 1, false)
	if err != nil {
		return err
	}

	StateGrph := grapher{}
	err = StateGrph.Init(&fProject.Project, 1, true)
	if err != nil {
		return err
	}
	log.Infof(colors.Fmt(colors.LightWhiteBold).Sprintf("Checking modules in state"))
	for _, md := range StateGrph.GetSequenceSet() {
		_, exists := p.Modules[md.Key()]
		if exists {
			continue
		}
		diff := utils.Diff(md.GetDiffData(), nil, true)
		log.Info(colors.Fmt(colors.Red).Sprintf("Module '%v' will be destroyed:", md.Key()))
		fmt.Printf("%v\n", diff)
	}

	for _, md := range CurrentGrph.GetSequenceSet() {

		diff := fProject.CheckModuleChanges(md)

		log.Infof(colors.Fmt(colors.LightWhiteBold).Sprintf("Planning module '%v':", md.Key()))
		if len(diff) > 0 || config.Global.Force {
			if len(diff) == 0 {
				diff = colors.Fmt(colors.GreenBold).Sprint("Not changed.")
			}
			fmt.Printf("%v\n", diff)

			if config.Global.ShowTerraformPlan {
				allDepsDeployed := true
				for _, planModDep := range *md.Dependencies() {
					_, exists := fProject.Modules[planModDep.Module.Key()]
					if !exists {
						allDepsDeployed = false
						break
					}
				}
				if allDepsDeployed {
					err := md.Build(md.ProjectPtr().codeCacheDir)
					if err != nil {
						log.Errorf("terraform plan: module build error: %v", err.Error())
					}
					err = md.Plan()
					if err != nil {
						log.Errorf("Module '%v' terraform plan return an error: %v", md.Key(), err.Error())
					}
				} else {
					log.Warnf("The module '%v' has dependencies that have not yet been deployed. Can't show terraform plan.", md.Key())
				}
			}
		} else {
			log.Infof(colors.Fmt(colors.GreenBold).Sprint("Not changed."))
		}
	}
	return nil
}