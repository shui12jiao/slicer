package kubeclient

// MDE
func (kc *KubeClient) ApplyMDE(mde []byte) error {
	return kc.Apply(mde, kc.config.Namespace)
}

func (kc *KubeClient) DeleteMDE(mde []byte) error {
	return kc.Delete(mde, kc.config.Namespace)
}

// kpi calculator
func (kc *KubeClient) ApplyKpic(kpi []byte) error {
	return kc.Apply(kpi, kc.config.MonitorNamespace)
}

func (kc *KubeClient) DeleteKpic(kpi []byte) error {
	return kc.Delete(kpi, kc.config.MonitorNamespace)
}

// slice
func (kc *KubeClient) ApplySlice(slice [][]byte) error {
	return kc.ApplyMulti(slice, kc.config.Namespace)
}

func (kc *KubeClient) DeleteSlice(slice [][]byte) error {
	return kc.DeleteMulti(slice, kc.config.Namespace)
}
