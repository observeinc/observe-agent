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
      
    New-Service @params
    Write-Output "Starting ObserveAgent Service..."
    Start-Service ObserveAgent
    }
else{
    Write-Output "ObserveAgent Service already exists, restarting service..."
    Stop-Service ObserveAgent
    Start-Service ObserveAgent
}


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
