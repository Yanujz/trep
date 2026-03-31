package main

import (
	"encoding/json"
	"os"

	"gopkg.in/yaml.v3"
)

type GlobalConfig struct {
	Title            string `yaml:"title" json:"title"`
	Format           string `yaml:"format" json:"format"`
	OutputFormat     string `yaml:"output-format" json:"output-format"`
	Quiet            bool   `yaml:"quiet" json:"quiet"`
	Annotate         bool   `yaml:"annotate" json:"annotate"`
	AnnotatePlatform string `yaml:"annotate-platform" json:"annotate-platform"`

	Test struct {
		NoMerge bool `yaml:"no-merge" json:"no-merge"`
		Fail    bool `yaml:"fail" json:"fail"`
	} `yaml:"test" json:"test"`

	Cov struct {
		ThresholdLine   float64  `yaml:"threshold-line" json:"threshold-line"`
		ThresholdBranch float64  `yaml:"threshold-branch" json:"threshold-branch"`
		ThresholdFunc   float64  `yaml:"threshold-func" json:"threshold-func"`
		Exclude         []string `yaml:"exclude" json:"exclude"`
		Fail            bool     `yaml:"fail" json:"fail"`
	} `yaml:"cov" json:"cov"`

	Report struct {
		Threshold float64 `yaml:"threshold" json:"threshold"`
		FailTests bool    `yaml:"fail-tests" json:"fail-tests"`
		FailCov   bool    `yaml:"fail-cov" json:"fail-cov"`
	} `yaml:"report" json:"report"`
}

func loadConfig() *GlobalConfig {
	cfg := &GlobalConfig{
		Format:           "auto",
		OutputFormat:     "html",
		AnnotatePlatform: "auto",
	}

	files := []string{".trep.yml", ".trep.yaml", ".trep.json"}
	var data []byte
	var isJSON bool

	for _, f := range files {
		if b, err := os.ReadFile(f); err == nil {
			data = b
			isJSON = (f == ".trep.json")
			break
		}
	}

	if len(data) > 0 {
		if isJSON {
			_ = json.Unmarshal(data, cfg)
		} else {
			_ = yaml.Unmarshal(data, cfg)
		}
	}
	return cfg
}
