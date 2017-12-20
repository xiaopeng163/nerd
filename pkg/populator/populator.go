//Package populator is a package that will help us to populate the kubernetes config file with the right credentials
package populator

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	v1payload "github.com/nerdalize/nerd/nerd/client/auth/v1/payload"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
)

//Prefix is used to know if a context comes from the cli.
var Prefix = "nerd-cli"

// P is an interface that we can use to read from and to write to the kube config file.
type P interface {
	PopulateKubeConfig(project string) error
}

//New instantiates a new P interface using the conf parameter. It can return a env, endpoint or oidc populator.
func New(conf, kubeConfigFile string, project *v1payload.GetProjectOutput) (P, error) {
	switch conf {
	case "oidc":
		return newOIDC(kubeConfigFile, project), nil
	case "endpoint":
		return newEndpoint(kubeConfigFile), nil
	case "env":
		return newEnv(kubeConfigFile), nil
	default:
		return nil, ErrNoSuchPopulator("populators implemented: oidc, endpoint, env")
	}
}

// ReadConfigOrNew retrieves Kubernetes client configuration from a file.
// If no files exists, an empty configuration is returned.
func ReadConfigOrNew(filename string) (*api.Config, error) {
	data, err := ioutil.ReadFile(filename)
	if os.IsNotExist(err) {
		return api.NewConfig(), nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "Error reading file %q", filename)
	}

	// decode config, empty if no bytes
	config, err := decode(data)
	if err != nil {
		return nil, errors.Errorf("could not read config: %v", err)
	}

	// initialize nil maps
	if config.AuthInfos == nil {
		config.AuthInfos = map[string]*api.AuthInfo{}
	}
	if config.Clusters == nil {
		config.Clusters = map[string]*api.Cluster{}
	}
	if config.Contexts == nil {
		config.Contexts = map[string]*api.Context{}
	}

	return config, nil
}

// ReadConfig retrieves Kubernetes client configuration from a file.
// If no files exists, it returns an error.
func ReadConfig(filename string) (*api.Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "Error reading file %q", filename)
	}

	// decode config, empty if no bytes
	config, err := decode(data)
	if err != nil {
		return nil, errors.Errorf("could not read config: %v", err)
	}

	// initialize nil maps
	if config.AuthInfos == nil {
		config.AuthInfos = map[string]*api.AuthInfo{}
	}
	if config.Clusters == nil {
		config.Clusters = map[string]*api.Cluster{}
	}
	if config.Contexts == nil {
		config.Contexts = map[string]*api.Context{}
	}

	return config, nil
}

// WriteConfig encodes the configuration and writes it to the given file.
// If the file exists, it's contents will be overwritten.
func WriteConfig(config *api.Config, filename string) error {
	if config == nil {
		glog.Errorf("could not write to '%s': config can't be nil", filename)
	}

	// encode config to YAML
	data, err := runtime.Encode(latest.Codec, config)
	if err != nil {
		return errors.Errorf("could not write to '%s': failed to encode config: %v", filename, err)
	}

	// create parent dir if doesn't exist
	dir := filepath.Dir(filename)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return errors.Wrapf(err, "Error creating directory: %s", dir)
		}
	}

	// write with restricted permissions
	if err := ioutil.WriteFile(filename, data, 0600); err != nil {
		return errors.Wrapf(err, "Error writing file %s", filename)
	}

	return nil
}

//Namespace get the namespace of the current context from the kube config file
func Namespace(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)

	if os.IsNotExist(err) {
		return "", err
	} else if err != nil {
		return "", errors.Wrapf(err, "Error reading file %q", filename)
	}

	// decode config, empty if no bytes
	config, err := decode(data)
	if err != nil {
		return "", errors.Errorf("could not read config: %v", err)
	}

	return config.Contexts[config.CurrentContext].Namespace, nil
}

//Context defines if the current context from the kube config file comes from the cli
func Context(filename string) bool {
	data, err := ioutil.ReadFile(filename)

	if os.IsNotExist(err) {
		return false
	} else if err != nil {
		return false
	}

	// decode config, empty if no bytes
	config, err := decode(data)
	if err != nil {
		return false
	}

	if strings.Contains(config.CurrentContext, Prefix) {
		return true
	}

	return false
}

// decode reads a Config object from bytes.
// Returns empty config if no bytes.
func decode(data []byte) (*api.Config, error) {
	// if no data, return empty config
	if len(data) == 0 {
		return api.NewConfig(), nil
	}

	config, _, err := latest.Codec.Decode(data, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "Error decoding config from data: %s", string(data))
	}

	return config.(*api.Config), nil
}
