package render

type SliceValue struct {
	ID  string // 切片ID= SST-SD
	SST string
	SD  string
}

type SessionValue struct {
	SessionSubnet string
	DNN           string
	Dev           string
}

type SessionValues = []SessionValue

type SmfConfigmapValue struct {
	SliceValue
	UPFAddr string
	SessionValues
}

type SmfDeploymentValue struct {
	SliceValue
	N4Addr string
	N3Addr string
}

type SmfServiceValue struct {
	SliceValue
}

type UpfConfigmapValue struct {
	SliceValue
	SessionValues
}

type UpfDeploymentValue struct {
	SliceValue
	N4Addr string
	N3Addr string
}
