# Windows development: Native Runtime and application blocks

CopyTyGo runs a full native Go application with `ctg dev`. On Windows, CopyTyGo now compiles the app to `build/dev/app.exe` and starts it from the project directory instead of using `go run`'s temporary executable. On Linux/macOS, `ctg dev` keeps using `go run`; `ctg dev --build` opts into the same project-local build path.

**Important:** A fixed path can make error reports reproducible, but it does not make an unsigned app trusted and cannot bypass Windows security policy.

If the blocked file is Windows `go.exe` itself, use [full Native Runtime within WSL2](WSL_DEVELOPMENT.md) as an alternative to the limited `--lite` mode. WSL keeps Windows Smart App Control enabled; Linux Go builds do not run as Windows `go.exe`.

## Normal workflow

Run from the generated project directory:

```powershell
ctg version
ctg doctor
ctg dev
```

`ctg dev` runs the complete Native Runtime (Go backend + Vite when using the auth starter). `ctg dev --lite` is a separate limited development option, not a substitute for the full auth dashboard or arbitrary Go project code.

## If Windows blocks app.exe

1. Note the full error and the file path. Look at **Windows Security > Virus & threat protection > Protection history** to determine whether Microsoft Defender Antivirus detected or quarantined a file.
2. Open **Event Viewer > Applications and Services Logs > Microsoft > Windows > CodeIntegrity > Operational**. Event **3077** signals an enforced App Control denial, while **3076** is an audit/evaluation event. Match the event timestamp and file name to your attempted `ctg dev` run.
3. In PowerShell, you can read recent App Control events without changing policy:

```powershell
Get-WinEvent -FilterHashtable @{
    LogName='Microsoft-Windows-CodeIntegrity/Operational'
    Id=3077
    StartTime=(Get-Date).AddMinutes(-30)
} -ErrorAction SilentlyContinue |
    Select-Object TimeCreated, Id, Message |
    Format-List
```

4. If a block is confirmed, distinguish **Smart App Control**, an organization's **App Control for Business/WDAC** policy, and an **antivirus detection**. For enforced trust policy, an approved publisher-signed build or an administrator-approved rule may be required. Verify the app's origin and security posture first.
5. If your device is organization-managed, contact the administrator with the Event ID, policy name, and executable path. Avoid arbitrary exclusions or disabling global real-time protection.

The separate `ctg build` command creates `build/app.exe`, which is intended for production packaging. The development command does not claim to change Windows trust decisions.

## Common distinctions

| Symptom | Likely next check |
| --- | --- |
| `go build` fails with compiler errors | Fix Go code or dependencies; this is not evidence of Defender blocking execution |
| `fork/exec ... app.exe: An Application Control policy has blocked this file` | Inspect CodeIntegrity event 3077 |
| Defender shows a threat name and quarantined file | Inspect Protection history and verify the detected file |
| `Access is denied` for a source file | Check file/directory permissions or another process holding it; do not assume an App Control block |
| `ctg dev --lite` works but full auth fails | Lite cannot execute the full Go app and browser auth; diagnose the native block |

## Official references

- Microsoft App Control event IDs: https://learn.microsoft.com/en-us/windows/security/application-security/application-control/app-control-for-business/operations/event-id-explanations
- Microsoft App Control troubleshooting: https://learn.microsoft.com/en-us/windows/security/application-security/application-control/app-control-for-business/operations/appcontrol-debugging-and-troubleshooting
- Smart App Control logs: https://learn.microsoft.com/en-us/windows/apps/develop/smart-app-control/test-your-app-with-smart-app-control

Keep sensitive environment variables, personal paths and account identifiers out of public issues and screenshots.
