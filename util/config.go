package util

type Config struct {
	KubeconfigPath string

	HTTPServerAddress string
	SliceBucket       string
	KubeBucket        string
}
