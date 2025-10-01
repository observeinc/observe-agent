##This script is responsible for creating & starting the observe agent service 

$observeagent_install_dir="$env:ProgramFiles\Observe\observe-agent"

if(-not (Get-Service ObserveAgent -ErrorAction SilentlyContinue)){
    Write-Output "Creating ObserveAgent Service...."
    $params = @{
        Name = "ObserveAgent"
        BinaryPathName =  "`"${observeagent_install_dir}\observe-agent.exe`" `"${observeagent_install_dir}\observe-agent.yaml`""
        DisplayName = "Observe Agent"
        StartupType = "Automatic"
        Description = "Observe Agent based on OpenTelemetry collector"
    }
    $params | Select-Object *
    New-Service @params
    Write-Output "Starting ObserveAgent Service..."
    try {
        Start-Service ObserveAgent -ErrorAction Stop
    } catch {
        Write-Output "Error starting ObserveAgent service!"
        $_ | Select-Object *
        # Try to print the agent config to help debug (may not work on older versions)
        try {
            &"${observeagent_install_dir}\observe-agent.exe" --observe-config "${observeagent_install_dir}\observe-agent.yaml" config 2>&1
        } catch {
            Write-Output "Could not print agent config (this is normal for older versions)"
        }
    }
}
else{
    Write-Output "ObserveAgent Service already exists, restarting service..."
    Stop-Service ObserveAgent -ErrorAction SilentlyContinue
    try {
        Start-Service ObserveAgent -ErrorAction Stop
    } catch {
        Write-Output "Error starting ObserveAgent service!"
        $_ | Select-Object *
        # Try to print the agent config to help debug (may not work on older versions)
        try {
            &"${observeagent_install_dir}\observe-agent.exe" --observe-config "${observeagent_install_dir}\observe-agent.yaml" config 2>&1
        } catch {
            Write-Output "Could not print agent config (this is normal for older versions)"
        }
    }
}

## Placeholder below for future use 
## Delete ObserveAgent service if needed 

# if (Get-Service "ObserveAgent" -ErrorAction 'SilentlyContinue')
# {
#     $service = Get-WmiObject -Class Win32_Service -Filter "Name='ObserveAgent'"
#     $service.delete()
#     Write-Host "ObserveAgent service deleted"
# }
# else{
#     write-host "No service found."
# }
