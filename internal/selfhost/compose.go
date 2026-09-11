package selfhost

import "fmt"

func BuildComposeArgs(envFile, action, detail string) ([]string, error) {
	if envFile == "" { return nil, fmt.Errorf("env file is required") }
	args := []string{"compose", "--project-name", "agentops", "--env-file", envFile, "-f", "deploy/windows/compose.yaml"}
	switch action {
	case "start": return append(args, "up", "-d"), nil
	case "stop": return append(args, "stop"), nil
	case "status": return append(args, "ps"), nil
	case "logs": return append(args, "logs", "--tail", "200"), nil
	case "backup": return append(args, "exec", "-T", "mysql", "mysqldump", "-u", "agentscope", "agentscope"), nil
	default: return nil, fmt.Errorf("unsupported action")
	}
}
