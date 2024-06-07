param (
    [Parameter(Mandatory)]
    $observe_collection_endpoint, 
    [Parameter(Mandatory)]
    $observe_token
)

$installer_url="https://github.com/observeinc/observe-agent/releases/download/v0.1.49/observe-agent_Windows_x86_64.zip"
$local_installer="C:\temp\observe-agent_Windows_x86_64.zip"
$program_data_filestorage="C:\ProgramData\Observe\observe-agent\filestorage"
$observeagent_install_dir="$env:ProgramFiles\Observe\observe-agent"
$temp_dir="C:\temp"

New-Item -ItemType Directory -Force -Path $temp_dir
New-Item -ItemType Directory -Force -Path $observeagent_install_dir
New-Item -ItemType Directory -Force -Path $observeagent_install_dir\config
New-Item -ItemType Directory -Force -Path $program_data_filestorage

Invoke-WebRequest -Uri $installer_url -OutFile $local_installer

# Stop the observe agent if its running so that we can copy the new .exe
if((Get-Service ObserveAgent -ErrorAction SilentlyContinue)){
    Stop-Service ObserveAgent
}

Expand-Archive -Force -LiteralPath $local_installer -DestinationPath "$temp_dir\observe-agent_Windows_x86_64"
Copy-Item -Force -Path $temp_dir\observe-agent_Windows_x86_64\observe-agent.exe -Destination $observeagent_install_dir
Copy-Item -Force -Path $temp_dir\observe-agent_Windows_x86_64\observe-agent.yaml -Destination $observeagent_install_dir
Copy-Item -Force -Path $temp_dir\observe-agent_Windows_x86_64\otel-collector.yaml -Destination $observeagent_install_dir\config\otel-collector.yaml
Copy-Item -Force -Path $temp_dir\observe-agent_Windows_x86_64\connections\ -Destination $observeagent_install_dir\connections -Recurse

# Read the content of the config.yaml file
$configContent = Get-Content -Path $observeagent_install_dir\observe-agent.yaml -Raw

# Replace ${myhost} with the actual value
$configContent = $configContent -replace '\${OBSERVE_COLLECTION_ENDPOINT}', $observe_collection_endpoint
$configContent = $configContent -replace '\${OBSERVE_TOKEN}', $observe_token

# Write the modified content back to the config.yaml file
$configContent | Set-Content -Path $observeagent_install_dir\observe-agent.yaml

if(-not (Get-Service ObserveAgent -ErrorAction SilentlyContinue)){
    $params = @{
        Name = "ObserveAgent"
        BinaryPathName =  "`"${observeagent_install_dir}\observe-agent.exe`" `"${observeagent_install_dir}\observe-agent.yaml`""
        DisplayName = "Observe Agent"
        StartupType = "Automatic"
        Description = "Observe Agent based on OpenTelemetry collector"
      }
      
    New-Service @params
    Start-Service ObserveAgent
    }
else{
    Stop-Service ObserveAgent
    Start-Service ObserveAgent
}