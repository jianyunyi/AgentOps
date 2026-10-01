package selfhost

import "testing"

func TestBuildComposeArgsUsesFixedContract(t *testing.T) {
	args, err := BuildComposeArgs("C:/ProgramData/AgentOps/config/agentops.env", "start", "")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"compose", "--project-name", "agentops", "--env-file", "C:/ProgramData/AgentOps/config/agentops.env", "-f", "deploy/windows/compose.yaml", "up", "-d"}
	if len(args) != len(want) {
		t.Fatalf("args = %v", args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args = %v", args)
		}
	}
}

func TestBuildComposeArgsRejectsUnknownCommand(t *testing.T) {
	if _, err := BuildComposeArgs("config.env", "rm -rf", ""); err == nil {
		t.Fatal("expected rejection")
	}
}
