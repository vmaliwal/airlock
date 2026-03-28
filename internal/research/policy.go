package research

import "os"

const HostExecutionExceptionEnv = "AIRLOCK_ALLOW_HOST_EXEC_EXCEPTION"

func HostExecutionExceptionDeclared() bool {
	return os.Getenv(HostExecutionExceptionEnv) == "1"
}
