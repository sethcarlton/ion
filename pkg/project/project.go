package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sst/ion/internal/fs"
	"github.com/sst/ion/pkg/js"
)

type Project struct {
	version       string
	root          string
	config        string
	name          string
	profile       string
	region        string
	stage         string
	removalPolicy string
	process       *js.Process

	AWS       *projectAws
	Bootstrap *bootstrap
	Stack     *stack
}

func New(version string) (*Project, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	cfgPath, err := fs.FindUp(cwd, "sst.config.ts")
	if err != nil {
		return nil, err
	}
	rootPath := filepath.Dir(cfgPath)

	process, err := js.Start(
		rootPath,
	)
	if err != nil {
		return nil, err
	}

	proj := &Project{
		version: version,
		root:    rootPath,
		config:  cfgPath,
		process: process,
	}
	proj.AWS = &projectAws{
		project: proj,
	}
	proj.Bootstrap = &bootstrap{
		project: proj,
	}
	proj.Stack = &stack{
		project: proj,
	}
	tmp := proj.PathTemp()

	_, err = os.Stat(tmp)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		err := os.Mkdir(tmp, 0755)
		if err != nil {
			return nil, err
		}
	}

	err = process.Eval(
		js.EvalOptions{
			Dir: tmp,
			Code: fmt.Sprintf(`
import mod from '%s';
console.log("~j" + JSON.stringify(mod.config()))`,
				cfgPath),
		},
	)
	if err != nil {
		return nil, err
	}

	for {
		cmd, line := process.Scan()

		if cmd == js.CommandDone {
			break
		}

		if cmd != js.CommandJSON {
			continue
		}

		parsed := struct {
			Name          string `json:"name"`
			Profile       string `json:"profile"`
			Stage         string `json:"stage"`
			Region        string `json:"region"`
			RemovalPolicy string `json:"removalPolicy"`
		}{}
		err = json.Unmarshal([]byte(line), &parsed)
		if err != nil {
			return nil, err
		}
		proj.name = parsed.Name
		proj.profile = parsed.Profile
		proj.stage = parsed.Stage
		proj.region = parsed.Region
		proj.removalPolicy = parsed.RemovalPolicy

		if proj.name == "" {
			return nil, fmt.Errorf("Project name is required")
		}

		if proj.region == "" {
			return nil, fmt.Errorf("Region is required")
		}

		if proj.removalPolicy == "" {
			proj.removalPolicy = "retain"
		}
		if proj.removalPolicy != "remove" && proj.removalPolicy != "retain" && proj.removalPolicy != "retain-all" {
			return nil, fmt.Errorf("RemovalPolicy must be one of: remove, retain, retain-all")
		}
	}

	return proj, nil
}

func Create() error {
	if _, err := os.Stat("sst.config.ts"); err == nil {
		return fmt.Errorf("sst.config.ts already exists")
	}

	return os.WriteFile("sst.config.ts", []byte(`
/// <reference path="./.sst/src/global.d.ts" />

export default {
  config() {
    return {
      name: "myapp"
    };
  },
  async run() {
    const a = new aws.s3.Bucket("my-bucket", {});
  },
};
`), 0644)
}

func (p *Project) getPath(path ...string) string {
	paths := append([]string{p.PathTemp()}, path...)
	return filepath.Join(paths...)
}

func (p *Project) PathTemp() string {
	return filepath.Join(p.root, ".sst")
}

func (p *Project) PathRoot() string {
	return p.root
}

func (p *Project) PathConfig() string {
	return p.config
}

func (p *Project) Name() string {
	return p.name
}

func (p *Project) Region() string {
	return p.region
}

func (p *Project) Profile() string {
	return p.profile
}

func (p *Project) RemovalPolicy() string {
	return p.removalPolicy
}

func (p *Project) Stage() string {
	return p.stage
}

func (p *Project) Version() string {
	return p.version
}
