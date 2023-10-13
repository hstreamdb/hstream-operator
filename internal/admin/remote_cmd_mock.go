package admin

import (
	"time"
)

type mockExecutor struct {
	getAPIByService func(namespace, serviceName, path string) (output []byte, statusCode int, err error)
}

func (e *mockExecutor) getPodNameByLabel(namespace string, label map[string]string) (name string, err error) {
	panic("implement me")
}

func (e *mockExecutor) ExecToPodByLabel(namespace string, label map[string]string, containerName, command string, timeout time.Duration) (output string, err error) {
	panic("implement me")
}

func (e *mockExecutor) GetAPIByService(namespace, serviceName, path string) ([]byte, int, error) {
	return e.getAPIByService(namespace, serviceName, path)
}
