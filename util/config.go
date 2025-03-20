package util

type Config struct {
	// for monitor
	MonarchThanosURI            string
	MonarchRequestTranslatorURI string
	MonitorTimeout              uint8

	// for mongodb
	MongoURI     string
	MongoDBName  string
	MongoTimeout uint8 // 单位秒

	// for kubernetes client
	Namespace      string
	KubeconfigPath string

	// for http server
	HTTPServerAddress string
	SliceStoreName    string
	KubeStoreName     string

	// for render
	TemplatePath string

	// for ipam
	N3Network           string
	N4Network           string
	SessionNetwork      string
	SessionSubnetLength uint8
	IPAMTimeout         uint8 // 单位秒
}
