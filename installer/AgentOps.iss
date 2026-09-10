[Setup]
AppName=AgentOps
AppVersion=0.1.0
DefaultDirName={autopf}\AgentOps
DefaultGroupName=AgentOps
ArchitecturesAllowed=x64
ArchitecturesInstallIn64BitMode=x64
PrivilegesRequired=admin
OutputBaseFilename=AgentOps-Setup-x64
OutputDir=..\dist

[Files]
Source: "..\dist\agentopsctl.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\deploy\windows\compose.yaml"; DestDir: "{app}\deploy\windows"; Flags: ignoreversion
Source: "scripts\*.cmd"; DestDir: "{app}\scripts"; Flags: ignoreversion

[Icons]
Name: "{group}\Start AgentOps"; Filename: "{app}\scripts\agentops-start.cmd"
Name: "{group}\Stop AgentOps"; Filename: "{app}\scripts\agentops-stop.cmd"
Name: "{group}\AgentOps Logs"; Filename: "{app}\scripts\agentops-logs.cmd"
Name: "{group}\Diagnose AgentOps"; Filename: "{app}\scripts\agentops-diagnose.cmd"

[Code]
function ExecChecked(const FileName, Params: String): Boolean;
var ResultCode: Integer;
begin
  Result := Exec(FileName, Params, '', SW_HIDE, ewWaitUntilTerminated, ResultCode) and (ResultCode = 0);
end;

function InitializeSetup(): Boolean;
begin
  if not ExecChecked(ExpandConstant('{cmd}'), '/c wsl.exe --status') then begin
    MsgBox('WSL 2 is required. Install and restart Windows before running AgentOps Setup.', mbError, MB_OK);
    Result := False; exit;
  end;
  if not ExecChecked(ExpandConstant('{cmd}'), '/c docker compose version') then begin
    MsgBox('Docker Desktop must be installed and running before AgentOps Setup.', mbError, MB_OK);
    Result := False; exit;
  end;
  Result := True;
end;

procedure CurStepChanged(CurStep: TSetupStep);
var ResultCode: Integer;
begin
  if CurStep = ssPostInstall then begin
    ForceDirectories(ExpandConstant('{commonappdata}\AgentOps\config'));
    Exec(ExpandConstant('{app}\agentopsctl.exe'), 'configure', '', SW_SHOWNORMAL, ewWaitUntilTerminated, ResultCode);
  end;
end;

[UninstallDelete]
; Application data in {commonappdata}\AgentOps is intentionally preserved.
