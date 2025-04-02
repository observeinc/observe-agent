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
        # Print the agent config to help debug
        &"${observeagent_install_dir}\observe-agent.exe" --observe-config "${observeagent_install_dir}\observe-agent.yaml" config
    }
}
else{
    Write-Output "ObserveAgent Service already exists, restarting service..."
    Stop-Service ObserveAgent
    Start-Service ObserveAgent
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
