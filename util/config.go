package util

type Config struct {
	Namespace string

	// for kubernetes client
	KubeconfigPath string

	// for http server
	HTTPServerAddress string
	SliceStoreName    string
	KubeStoreName     string

	// for render
	TemplatePath string
	KubePath     string
}
