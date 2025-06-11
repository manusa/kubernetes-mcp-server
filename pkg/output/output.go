package output

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

type Output interface {
	GetName() string
	PrintObj(obj any) (string, error)
}

var Outputs = []Output{
	&YamlOutput{},
}

var Names []string

func FromString(name string) Output {
	for _, output := range Outputs {
		if output.GetName() == name {
			return output
		}
	}
	return nil
}

type YamlOutput struct{}

func (p *YamlOutput) GetName() string {
	return "yaml"
}
func (p *YamlOutput) PrintObj(obj any) (string, error) {
	return MarshalYaml(obj)
}

func MarshalYaml(v any) (string, error) {
	switch t := v.(type) {
	case []unstructured.Unstructured:
		for i := range t {
			t[i].SetManagedFields(nil)
		}
	case []*unstructured.Unstructured:
		for i := range t {
			t[i].SetManagedFields(nil)
		}
	case unstructured.Unstructured:
		t.SetManagedFields(nil)
	case *unstructured.Unstructured:
		t.SetManagedFields(nil)
	}
	ret, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(ret), nil
}

func init() {
	Names = make([]string, 0)
	for _, output := range Outputs {
		Names = append(Names, output.GetName())
	}
}
